// Package phonepair keeps which phones may open hustleX's web app on a
// host, and pairs new ones: a token shown as a QR code on the computer,
// good once and for a few minutes, with a code to check on both screens.
// A paired phone holds a secret; the host keeps only its hash.
package phonepair

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"
)

// pairFor is how long a pairing token is good for.
const pairFor = 10 * time.Minute

// Device is a paired phone.
type Device struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Hash  string    `json:"hash"` // of its secret
	Added time.Time `json:"added"`
	Seen  time.Time `json:"seen,omitzero"`
}

type stored struct {
	Enabled bool     `json:"enabled"`
	Devices []Device `json:"devices,omitempty"`
}

// Store is the host's phones, saved in a file. Its methods are safe from
// any goroutine.
type Store struct {
	path string

	mu      sync.Mutex
	st      stored
	token   string    // the pairing token offered, "" for none
	expires time.Time // when it goes
}

// Open reads the store at path; a missing file is an empty store.
func Open(path string) (*Store, error) {
	s := &Store{path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("phonepair: %w", err)
	}
	if err := json.Unmarshal(data, &s.st); err != nil {
		return nil, fmt.Errorf("phonepair: read %s: %w", path, err)
	}
	return s, nil
}

func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("phonepair: %w", err)
	}
	return nil
}

// Enabled reports whether phones may open the web app.
func (s *Store) Enabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st.Enabled
}

// SetEnabled turns the web app on or off.
func (s *Store) SetEnabled(on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.st.Enabled = on
	if !on {
		s.token = ""
	}
	return s.saveLocked()
}

// Devices are the paired phones, the newest first.
func (s *Store) Devices() []Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := slices.Clone(s.st.Devices)
	slices.Reverse(out)
	return out
}

// Forget unpairs device id: its secret opens nothing anymore.
func (s *Store) Forget(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.st.Devices = slices.DeleteFunc(s.st.Devices, func(d Device) bool { return d.ID == id })
	return s.saveLocked()
}

// Offer makes a new pairing token, good for a few minutes, in place of
// the last one, with the code both screens show and when it goes.
func (s *Store) Offer(now time.Time) (token, code string, expires time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token, s.expires = random(16), now.Add(pairFor)
	return s.token, Code(s.token), s.expires
}

// Code is the six digits both screens show for token, to check that the
// phone opened the computer's own page.
func Code(token string) string {
	sum := sha256.Sum256([]byte("hustlex pair " + token))
	n := binary.BigEndian.Uint32(sum[:4]) % 1_000_000
	return fmt.Sprintf("%03d %03d", n/1000, n%1000)
}

// Valid reports whether token is the pairing token offered, unexpired.
func (s *Store) Valid(token string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.validLocked(token, now)
}

func (s *Store) validLocked(token string, now time.Time) bool {
	return s.token != "" && now.Before(s.expires) && subtle.ConstantTimeCompare([]byte(token), []byte(s.token)) == 1
}

// ErrToken refuses a pairing token that isn't offered, or went.
var ErrToken = errors.New("this pairing code has expired: pair again from hustleX on the computer")

// Pair pairs a phone named name with token, once: it returns the secret
// the phone keeps.
func (s *Store) Pair(token, name string, now time.Time) (string, Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.st.Enabled || !s.validLocked(token, now) {
		return "", Device{}, ErrToken
	}
	s.token = ""
	secret := random(32)
	if name == "" {
		name = "Phone"
	}
	d := Device{ID: random(6), Name: name, Hash: hash(secret), Added: now.UTC(), Seen: now.UTC()}
	s.st.Devices = append(s.st.Devices, d)
	if err := s.saveLocked(); err != nil {
		return "", Device{}, err
	}
	return secret, d, nil
}

// Check finds the paired phone holding secret, noting it was seen.
func (s *Store) Check(secret string, now time.Time) (Device, bool) {
	if secret == "" {
		return Device{}, false
	}
	h := hash(secret)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.st.Enabled {
		return Device{}, false
	}
	for i, d := range s.st.Devices {
		if subtle.ConstantTimeCompare([]byte(d.Hash), []byte(h)) == 1 {
			if now.Sub(d.Seen) > time.Minute {
				s.st.Devices[i].Seen = now.UTC()
				_ = s.saveLocked()
			}
			return s.st.Devices[i], true
		}
	}
	return Device{}, false
}

func hash(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func random(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
