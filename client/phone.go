package client

import "github.com/hustlex/hustlex/proto"

// PhoneStatus says whether the phone web app is on, where, and the phones
// paired.
func (c *Client) PhoneStatus() (proto.PhoneStatus, error) {
	var out proto.PhoneStatus
	err := c.Call(proto.MethodPhoneStatus, nil, &out)
	return out, err
}

// SetPhone turns the phone web app on or off.
func (c *Client) SetPhone(on bool) (proto.PhoneStatus, error) {
	var out proto.PhoneStatus
	err := c.Call(proto.MethodPhoneSet, proto.PhoneSetParams{Enabled: on}, &out)
	return out, err
}

// PairPhone offers a pairing to show as a QR code.
func (c *Client) PairPhone() (proto.PhonePairing, error) {
	var out proto.PhonePairing
	err := c.Call(proto.MethodPhonePair, nil, &out)
	return out, err
}

// ForgetPhone unpairs phone id.
func (c *Client) ForgetPhone(id string) error {
	return c.Call(proto.MethodPhoneForget, proto.PhoneDeviceRef{ID: id}, nil)
}
