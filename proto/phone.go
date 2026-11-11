package proto

import "time"

// A host's daemon can serve hustleX's web app to the user's phones over
// Tailscale: what waits on the user, the sessions, the accounts. Phones
// pair with a code the app shows (docs/phone.md).
const (
	MethodPhoneStatus = "phone.status" // → PhoneStatus
	MethodPhoneSet    = "phone.set"    // PhoneSetParams → PhoneStatus
	MethodPhonePair   = "phone.pair"   // → PhonePairing
	MethodPhoneForget = "phone.forget" // PhoneDeviceRef
)

// PhoneStatus is whether the web app is on, where phones open it, and
// the phones paired.
type PhoneStatus struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url,omitempty"` // while served
	// Problem says why it isn't served while on: no Tailscale, say.
	Problem string        `json:"problem,omitempty"`
	Devices []PhoneDevice `json:"devices,omitempty"`
}

// PhoneDevice is a paired phone.
type PhoneDevice struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Added time.Time `json:"added"`
	Seen  time.Time `json:"seen,omitzero"`
}

// PhoneSetParams turn the web app on or off.
type PhoneSetParams struct {
	Enabled bool `json:"enabled"`
}

// PhonePairing is a pairing offered: the address the phone opens (a QR
// code of it), the code both screens show, and when it goes.
type PhonePairing struct {
	URL     string    `json:"url"`
	Code    string    `json:"code"`
	Expires time.Time `json:"expires"`
}

// PhoneDeviceRef names a paired phone.
type PhoneDeviceRef struct {
	ID string `json:"id"`
}
