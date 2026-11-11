package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// A phone pairs once with the token the app offers, then reads the
// state with its cookie; other hosts' names, scripts elsewhere and
// unpaired phones get nothing.
func TestPhonePairsThenReads(t *testing.T) {
	srv := startServer(t)
	if err := srv.phone.store.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	h := srv.phoneHandler()
	do := func(method, target, body, cookie string, header bool, host string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, target, strings.NewReader(body))
		r.Host = host
		if header {
			r.Header.Set("X-Hustlex", "1")
		}
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: phoneCookie, Value: cookie})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	const host = "100.101.102.103:7651"
	token, code, _ := srv.phone.store.Offer(time.Now())

	if w := do("GET", "/", "", "", false, "evil.example.com"); w.Code != http.StatusForbidden {
		t.Fatalf("another host's name: %d", w.Code)
	}
	if w := do("GET", "/", "", "", false, host); w.Code != 200 || !strings.Contains(w.Body.String(), "hustleX") {
		t.Fatalf("the page: %d", w.Code)
	}
	w := do("GET", "/api/pair?t="+url.QueryEscape(token), "", "", false, host)
	if w.Code != 200 || !strings.Contains(w.Body.String(), code) {
		t.Fatalf("pair info: %d %s", w.Code, w.Body)
	}
	if w := do("GET", "/api/state", "", "", false, host); w.Code != http.StatusUnauthorized {
		t.Fatalf("unpaired state: %d", w.Code)
	}
	body := `{"token":"` + token + `","name":"iPhone"}`
	if w := do("POST", "/api/pair", body, "", false, host); w.Code != http.StatusForbidden {
		t.Fatalf("a write without the header: %d", w.Code)
	}
	w = do("POST", "/api/pair", body, "", true, host)
	if w.Code != 200 {
		t.Fatalf("pair: %d %s", w.Code, w.Body)
	}
	var secret string
	for _, c := range w.Result().Cookies() {
		if c.Name == phoneCookie && c.HttpOnly {
			secret = c.Value
		}
	}
	if secret == "" {
		t.Fatal("no cookie")
	}
	if w := do("POST", "/api/pair", body, "", true, host); w.Code != http.StatusGone {
		t.Fatalf("the token paired twice: %d", w.Code)
	}
	w = do("GET", "/api/state", "", secret, false, host)
	var st phoneStateView
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &st) != nil || st.Inbox == nil || st.Sessions == nil {
		t.Fatalf("state: %d %s", w.Code, w.Body)
	}
	if got := srv.phoneStatus(); len(got.Devices) != 1 || got.Devices[0].Name != "iPhone" {
		t.Fatalf("devices %+v", got.Devices)
	}
	if err := srv.phone.store.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if w := do("GET", "/api/state", "", secret, false, host); w.Code != http.StatusUnauthorized {
		t.Fatalf("read while off: %d", w.Code)
	}
}

func TestPhoneHostOK(t *testing.T) {
	for host, want := range map[string]bool{
		"100.64.0.1": true, "100.127.255.254": true, "mac.tail1234.ts.net": true, "localhost": true, "[fd7a:115c:a1e0::1]": true,
		"192.168.1.5": true, "10.0.0.7": true, "172.16.4.2": true,
		"100.128.0.1": false, "8.8.8.8": false, "example.com": false, "ts.net.example.com": false,
	} {
		if got := phoneHostOK(host); got != want {
			t.Errorf("phoneHostOK(%q) = %v", host, got)
		}
	}
}
