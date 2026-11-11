package daemon

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hustlex/hustlex/core/phonepair"
	"github.com/hustlex/hustlex/core/tailnet"
	"github.com/hustlex/hustlex/proto"
)

// The daemon serves hustleX's web app to the user's phones, on the host's
// Tailscale address, else on its address on the local network (phones on
// the same Wi-Fi): what waits on the user (answered from the
// phone), the sessions (read and written to), the accounts. A phone pairs
// once with a token the app shows as a QR code; it then holds a secret in
// a cookie. See docs/phone.md.

//go:embed phoneweb
var phoneWeb embed.FS

// phonePort is the port the web app listens on (tests pick their own).
var phonePort = 7651

// phoneCookie names the paired phone's secret.
const phoneCookie = "hx"

// phoneServer belongs to runPhone but for what status reads, under mu.
type phoneServer struct {
	store *phonepair.Store
	wake  chan struct{}

	mu      sync.Mutex
	srv     *http.Server
	addr    string // what it listens on, "" while it doesn't
	wifi    bool   // addr is on the local network, not Tailscale
	problem string
}

func (ps *phoneServer) poke() {
	select {
	case ps.wake <- struct{}{}:
	default:
	}
}

// openPhone reads the phones paired with this host.
func (s *Server) openPhone(dir string) error {
	st, err := phonepair.Open(filepath.Join(dir, "phone.json"))
	if err != nil {
		return err
	}
	s.phone.store, s.phone.wake = st, make(chan struct{}, 1)
	return nil
}

// runPhone serves the web app while it is on, on the host's Tailscale
// address as it is now, until done closes.
func (s *Server) runPhone(done <-chan struct{}) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		s.reconcilePhone()
		select {
		case <-done:
			s.stopPhone("")
			return
		case <-s.phone.wake:
		case <-t.C:
		}
	}
}

// reconcilePhone starts, moves or stops the web app as the setting and
// the Tailscale address say.
func (s *Server) reconcilePhone() {
	ps := &s.phone
	if !ps.store.Enabled() {
		s.stopPhone("")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	self, err := tailnet.Self(ctx)
	if err == nil && !phoneOnTailnet(ctx) && localIP() != "" {
		err = errors.New("no phone on the tailnet") // the Wi-Fi reaches phones without Tailscale
	}
	cancel()
	ip, wifi := self.IP, false
	if err != nil {
		// No Tailscale: phones on the same Wi-Fi reach the local address.
		if ip, wifi = localIP(), true; ip == "" {
			s.stopPhone("This computer isn't on a network: connect it to Wi-Fi, the same as your phone's")
			return
		}
	}
	addr := net.JoinHostPort(ip, strconv.Itoa(phonePort))
	ps.mu.Lock()
	same := ps.addr == addr && ps.srv != nil
	ps.mu.Unlock()
	if same {
		return
	}
	s.stopPhone("")
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		s.stopPhone("Couldn't listen on " + addr + ": " + err.Error())
		return
	}
	srv := &http.Server{Handler: s.phoneHandler(), ReadHeaderTimeout: 10 * time.Second}
	ps.mu.Lock()
	ps.srv, ps.addr, ps.wifi, ps.problem = srv, addr, wifi, ""
	ps.mu.Unlock()
	go func() { _ = srv.Serve(ln) }()
	s.logger().Info("serving the phone web app", "addr", addr)
}

// stopPhone stops serving, saying why when it should be served.
func (s *Server) stopPhone(problem string) {
	ps := &s.phone
	ps.mu.Lock()
	srv := ps.srv
	ps.srv, ps.addr, ps.wifi, ps.problem = nil, "", false, problem
	ps.mu.Unlock()
	if srv != nil {
		_ = srv.Close()
	}
}

func (s *Server) phoneStatus() proto.PhoneStatus {
	ps := &s.phone
	st := proto.PhoneStatus{Enabled: ps.store.Enabled()}
	ps.mu.Lock()
	if ps.addr != "" {
		st.URL, st.WiFi = "http://"+ps.addr, ps.wifi
	}
	st.Problem = ps.problem
	ps.mu.Unlock()
	for _, d := range ps.store.Devices() {
		st.Devices = append(st.Devices, proto.PhoneDevice{ID: d.ID, Name: d.Name, Added: d.Added, Seen: d.Seen})
	}
	return st
}

func (s *Server) setPhone(p proto.PhoneSetParams) (proto.PhoneStatus, error) {
	if err := s.phone.store.SetEnabled(p.Enabled); err != nil {
		return proto.PhoneStatus{}, err
	}
	// runPhone alone starts and stops the server; wait a little for it.
	s.phone.poke()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		if st := s.phoneStatus(); !p.Enabled && st.URL == "" || p.Enabled && (st.URL != "" || st.Problem != "") {
			return st, nil
		}
	}
	return s.phoneStatus(), nil
}

// pairPhone offers a pairing: the address a phone opens, with its token.
func (s *Server) pairPhone() (proto.PhonePairing, error) {
	st := s.phoneStatus()
	switch {
	case !st.Enabled:
		return proto.PhonePairing{}, errors.New("turn on the phone web app first")
	case st.URL == "" && st.Problem != "":
		return proto.PhonePairing{}, errors.New(st.Problem)
	case st.URL == "":
		return proto.PhonePairing{}, errors.New("the phone web app is starting: try again in a moment")
	}
	token, code, expires := s.phone.store.Offer(time.Now())
	return proto.PhonePairing{URL: st.URL + "/pair?t=" + token, Code: code, Expires: expires}, nil
}

// phoneHandler is the web app: its page, the pairing, and the API a
// paired phone uses.
func (s *Server) phoneHandler() http.Handler {
	mux := http.NewServeMux()
	page := func(w http.ResponseWriter, r *http.Request) {
		data, _ := phoneWeb.ReadFile("phoneweb/index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(data)
	}
	mux.HandleFunc("GET /{$}", page)
	mux.HandleFunc("GET /pair", page)
	for _, f := range []struct{ path, kind string }{
		{"manifest.webmanifest", "application/manifest+json"}, {"icon-180.png", "image/png"}, {"icon-512.png", "image/png"},
	} {
		mux.HandleFunc("GET /"+f.path, func(w http.ResponseWriter, r *http.Request) {
			data, _ := phoneWeb.ReadFile("phoneweb/" + f.path)
			w.Header().Set("Content-Type", f.kind)
			_, _ = w.Write(data)
		})
	}
	mux.HandleFunc("GET /api/pair", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("t")
		if !s.phone.store.Valid(token, time.Now()) {
			http.Error(w, phonepair.ErrToken.Error(), http.StatusGone)
			return
		}
		host, _ := os.Hostname()
		writeJSON(w, map[string]string{"code": phonepair.Code(token), "host": strings.TrimSuffix(host, ".local")})
	})
	mux.HandleFunc("POST /api/pair", func(w http.ResponseWriter, r *http.Request) {
		var p struct{ Token, Name string }
		if !readJSON(w, r, &p) {
			return
		}
		secret, _, err := s.phone.store.Pair(p.Token, strings.TrimSpace(p.Name), time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusGone)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: phoneCookie, Value: secret, Path: "/", MaxAge: 400 * 24 * 3600, HttpOnly: true, SameSite: http.SameSiteStrictMode})
		writeJSON(w, map[string]bool{"ok": true})
	})
	auth := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(phoneCookie)
			if err != nil {
				http.Error(w, "not paired", http.StatusUnauthorized)
				return
			}
			if _, ok := s.phone.store.Check(c.Value, time.Now()); !ok {
				http.Error(w, "not paired", http.StatusUnauthorized)
				return
			}
			h(w, r)
		}
	}
	mux.HandleFunc("GET /api/state", auth(func(w http.ResponseWriter, r *http.Request) { writeJSON(w, maskLentIn(s, s.phoneState())) }))
	mux.HandleFunc("GET /api/session", auth(func(w http.ResponseWriter, r *http.Request) {
		v, err := s.phoneSession(r.URL.Query().Get("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, maskLentIn(s, v))
	}))
	mux.HandleFunc("GET /api/history", auth(func(w http.ResponseWriter, r *http.Request) {
		days, _ := strconv.Atoi(r.URL.Query().Get("days"))
		h, err := s.usageHistoryReport(days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for i, c := range h.Conversations {
			if c.Session != "" {
				if sess, err := s.sessions.Get(c.Session); err == nil {
					h.Conversations[i].Title = phoneName(sess.Info())
				}
			}
		}
		writeJSON(w, maskLentIn(s, h))
	}))
	mux.HandleFunc("POST /api/answer", auth(func(w http.ResponseWriter, r *http.Request) {
		var p struct{ Session, Key string }
		if !readJSON(w, r, &p) {
			return
		}
		if err := s.input(proto.SessionInputParams{ID: p.Session, Text: p.Key}); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]bool{"ok": true})
	}))
	mux.HandleFunc("POST /api/send", auth(func(w http.ResponseWriter, r *http.Request) {
		var p struct{ Session, Text string }
		if !readJSON(w, r, &p) {
			return
		}
		if strings.TrimSpace(p.Text) == "" {
			http.Error(w, "nothing to send", http.StatusBadRequest)
			return
		}
		if err := s.input(proto.SessionInputParams{ID: p.Session, Text: p.Text, Paste: true, Enter: true}); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]bool{"ok": true})
	}))
	return phoneGuard(mux)
}

// phoneGuard lets in requests to the host as Tailscale names it (its
// address, or a ts.net name), refusing other names (a page rebinding its
// own name to the host); writes must say they come from the web app's own
// script, which pages elsewhere can't without asking.
func phoneGuard(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if !phoneHostOK(host) {
			http.Error(w, "unknown host", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost && r.Header.Get("X-Hustlex") != "1" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		h.ServeHTTP(w, r)
	})
}

// phoneHostOK reports whether host names this host: one of Tailscale's
// addresses or names, a private address on the local network, or this
// machine itself.
func phoneHostOK(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if strings.HasSuffix(host, ".ts.net") || host == "localhost" {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil {
		return false
	}
	_, cgnat, _ := net.ParseCIDR("100.64.0.0/10")
	_, ula, _ := net.ParseCIDR("fd7a:115c:a1e0::/48")
	return ip.IsLoopback() || ip.IsPrivate() || cgnat.Contains(ip) || ula.Contains(ip)
}

// phoneOnTailnet reports whether a phone is on the tailnet to open the
// web app over Tailscale.
func phoneOnTailnet(ctx context.Context) bool {
	peers, err := tailnet.Peers(ctx)
	if err != nil {
		return false
	}
	for _, p := range peers {
		if os := strings.ToLower(p.OS); os == "ios" || os == "android" {
			return true
		}
	}
	return false
}

// localIP is this computer's private IPv4 address on the network it
// reaches others by, "" when it isn't on one. Dialing UDP sends nothing.
func localIP() string {
	conn, err := net.Dial("udp4", "192.0.2.1:9")
	if err != nil {
		return ""
	}
	defer conn.Close()
	ip := conn.LocalAddr().(*net.UDPAddr).IP
	if !ip.IsPrivate() {
		return ""
	}
	return ip.String()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(v); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return false
	}
	return true
}
