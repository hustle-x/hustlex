package phonepair

import (
	"path/filepath"
	"testing"
	"time"
)

func TestPairOnceThenCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "phone.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	token, code, _ := s.Offer(now)
	if _, _, err := s.Pair(token, "iPhone", now); err == nil {
		t.Fatal("paired while off")
	}
	if err := s.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	token, code2, _ := s.Offer(now)
	if code == code2 || len(code2) != 7 {
		t.Fatalf("codes %q %q", code, code2)
	}
	if _, _, err := s.Pair("wrong", "iPhone", now); err == nil {
		t.Fatal("paired with a wrong token")
	}
	secret, d, err := s.Pair(token, "iPhone", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Pair(token, "Other", now); err == nil {
		t.Fatal("a token paired twice")
	}
	again, _ := Open(path)
	if got, ok := again.Check(secret, now); !ok || got.ID != d.ID {
		t.Fatal("the paired phone isn't known after a reopen")
	}
	if _, ok := again.Check("nope", now); ok {
		t.Fatal("a wrong secret opened it")
	}
	late, _, _ := again.Offer(now)
	if _, _, err := again.Pair(late, "Late", now.Add(pairFor+time.Second)); err == nil {
		t.Fatal("an expired token paired")
	}
	if err := again.Forget(d.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := again.Check(secret, now); ok {
		t.Fatal("a forgotten phone still opens it")
	}
}
