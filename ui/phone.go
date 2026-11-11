package ui

import (
	"image"
	"image/color"
	"time"

	"github.com/egoist/mygo/ui"
	"rsc.io/qr"

	"github.com/hustlex/hustlex/client"
	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/textfmt"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// hustleX on the user's phone: the host's daemon serves a web app over
// Tailscale (daemon/phone.go); the app turns it on, pairs phones with a QR
// code and forgets them.

// phoneState belongs to the main thread.
type phoneState struct {
	host    string // the host it was read for
	loaded  bool
	status  proto.PhoneStatus
	err     string
	busy    bool
	open    bool // the Pair a phone dialog shows
	pairing proto.PhonePairing
	qr      *ui.Bitmap
	devices int // phones paired when the dialog opened, to see a new one
	polling *int
}

// phoneOf is the connected host's phone state, read anew for a new host.
func (a *App) phoneOf() *phoneState {
	if a.phone.host != a.host.ID {
		a.phone = phoneState{host: a.host.ID}
		a.loadPhone()
	}
	return &a.phone
}

func (a *App) loadPhone() {
	cl, host := a.client, a.host.ID
	if cl == nil {
		return
	}
	go func() {
		st, err := cl.PhoneStatus()
		a.onMain(func() {
			if a.phone.host != host {
				return
			}
			a.phone.loaded = true
			if err != nil {
				a.phone.err = client.Explain(err)
				return
			}
			a.phone.status, a.phone.err = st, ""
		})
	}()
}

// phoneCard is Settings → App's Your phone: on or off, where phones open
// it, Pair a phone, and the phones paired.
func (a *App) phoneCard(c *ui.Context) {
	ps := a.phoneOf()
	widget.Card(c).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			ui.Row(c).FillWidth().Gap(12).Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
					ui.Text(c, "Your phone").FontSize(12.5)
					ui.Text(c, "Open hustleX on your phone over Tailscale: scan with the camera, then Add to Home Screen. Answer your agents, read and write to sessions, see your accounts.").
						FontSize(11.5).TextColor(theme.Muted)
				})
				on := ps.status.Enabled
				if ps.loaded && ps.err == "" && widget.Switch(c, &on, "Your phone") {
					a.setPhone(on)
				}
			})
			switch {
			case ps.err != "":
				ui.Text(c, ps.err).FontSize(11.5).TextColor(theme.Error)
			case !ps.status.Enabled:
				return
			case ps.status.Problem != "":
				ui.Text(c, ps.status.Problem).FontSize(11.5).TextColor(theme.Warning)
			case ps.status.URL != "":
				ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, ps.status.URL).Font(theme.MonoFont).FontSize(11.5).TextColor(theme.TextSoft).Grow(1).SingleLine()
					if widget.Button(c, widget.Primary, icon.Phone, "Pair a phone…", "").Clicked() {
						a.pairPhone()
					}
				})
			default:
				ui.Text(c, "Starting…").FontSize(11.5).TextColor(theme.Muted)
			}
			for _, d := range ps.status.Devices {
				ui.Row(c.Key(d.ID)).FillWidth().Gap(10).Padding(7, 10).Radius(8).Background(theme.Sunken).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, icon.Phone).FontSize(13).TextColor(theme.Muted)
					ui.Column(c).Grow(1).MinWidth(0).Children(func() {
						ui.Text(c, d.Name).FontSize(12.5)
						seen := "paired " + textfmt.Age(d.Added, time.Now()) + " ago"
						if !d.Seen.IsZero() {
							seen = "last used " + textfmt.Age(d.Seen, time.Now()) + " ago"
						}
						ui.Text(c, seen).FontSize(11).TextColor(theme.Muted)
					})
					if widget.GhostButton(c.Key("forget"), "Forget").Clicked() {
						a.forgetPhone(d)
					}
				})
			}
		})
	})
}

func (a *App) setPhone(on bool) {
	cl := a.client
	if cl == nil {
		return
	}
	a.phone.status.Enabled = on
	go func() {
		st, err := cl.SetPhone(on)
		a.onMain(func() {
			if err != nil {
				a.phone.err = client.Explain(err)
				return
			}
			a.phone.status, a.phone.err = st, ""
		})
	}()
}

func (a *App) forgetPhone(d proto.PhoneDevice) {
	cl := a.client
	if cl == nil {
		return
	}
	go func() {
		err := cl.ForgetPhone(d.ID)
		a.onMain(func() {
			if err != nil {
				a.status = client.Explain(err)
				return
			}
			a.notify(d.Name + " can't open hustleX anymore")
			a.loadPhone()
		})
	}()
}

// pairPhone offers a pairing and shows its QR code, turning the web app
// on first when it is off.
func (a *App) pairPhone() {
	cl := a.client
	if cl == nil {
		return
	}
	ps := a.phoneOf()
	ps.busy, ps.err = true, ""
	go func() {
		st, err := cl.PhoneStatus()
		if err == nil && !st.Enabled {
			st, err = cl.SetPhone(true)
		}
		var p proto.PhonePairing
		for i := 0; err == nil && i < 10; i++ {
			// A web app just turned on starts in a moment.
			if p, err = cl.PairPhone(); err == nil || st.Problem != "" {
				break
			}
			time.Sleep(500 * time.Millisecond)
			st, _ = cl.PhoneStatus()
		}
		var bmp *ui.Bitmap
		if err == nil {
			bmp, err = qrBitmap(p.URL)
		}
		a.onMain(func() {
			ps := &a.phone
			ps.busy, ps.status = false, st
			if err != nil {
				a.status = "Couldn't pair a phone: " + client.Explain(err)
				return
			}
			ps.pairing, ps.qr, ps.open, ps.devices = p, bmp, true, len(st.Devices)
			a.followPairing(cl)
		})
	}()
}

// qrBitmap is text as a QR code, with its quiet zone, a few pixels a
// module, drawn crisp at any size.
func qrBitmap(text string) (*ui.Bitmap, error) {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return nil, err
	}
	const scale, quiet = 6, 4
	n := (code.Size + 2*quiet) * scale
	img := image.NewGray(image.Rect(0, 0, n, n))
	for y := range n {
		for x := range n {
			c := color.Gray{Y: 255}
			if code.Black(x/scale-quiet, y/scale-quiet) {
				c.Y = 0
			}
			img.SetGray(x, y, c)
		}
	}
	return ui.NewBitmap(img).Pixelated(), nil
}

// followPairing reads the phones paired while the dialog shows, to say
// when the new one did.
func (a *App) followPairing(cl *client.Client) {
	tok := new(int)
	a.phone.polling = tok
	go func() {
		for {
			time.Sleep(2 * time.Second)
			st, err := cl.PhoneStatus()
			stop := make(chan bool, 1)
			a.onMain(func() {
				ps := &a.phone
				if !ps.open || ps.polling != tok {
					stop <- true
					return
				}
				if err == nil {
					ps.status = st
					if len(st.Devices) > ps.devices {
						ps.open = false
						a.notify("Paired " + st.Devices[0].Name + ": add hustleX to its Home Screen")
						stop <- true
						return
					}
				}
				stop <- false
			})
			if <-stop {
				return
			}
		}
	}()
}

// pairPhoneDialog shows the QR code a phone scans, and the code both
// screens show.
func (a *App) pairPhoneDialog(c *ui.Context) {
	ps := &a.phone
	if !ps.open {
		return
	}
	widget.Dialog(c, &ps.open, 420, 14, func() {
		ui.Row(c).FillWidth().Children(func() {
			ui.Column(c).Grow(1).Gap(3).Children(func() {
				ui.Text(c, "Pair a phone").FontSize(16)
				ui.Text(c, "Scan with your phone's camera. It opens hustleX on "+a.host.Name+" over Tailscale.").FontSize(12).TextColor(theme.Muted)
			})
			if widget.IconButton(c, icon.Close, "Close", false).Clicked() {
				ps.open = false
			}
		})
		ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
			if ps.qr != nil {
				ui.Box(c).Padding(10).Radius(12).Background(ui.Color{R: 255, G: 255, B: 255, A: 255}).Children(func() {
					ui.Image(c, ps.qr).Size(200, 200)
				})
			}
		})
		ui.Column(c).FillWidth().Gap(2).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Check your phone shows").FontSize(12).TextColor(theme.Muted)
			ui.Text(c, ps.pairing.Code).Font(theme.MonoFont).FontSize(22).TextColor(theme.Text)
		})
		for i, step := range []string{
			"Your phone needs Tailscale, signed in to the same account.",
			"Tap Pair on the phone, then Share → Add to Home Screen.",
			"The code works once, for " + textfmt.Until(ps.pairing.Expires, time.Now()) + ".",
		} {
			ui.Row(c.Key(i)).FillWidth().Gap(8).Children(func() {
				ui.Text(c, "•").FontSize(12).TextColor(theme.Muted)
				ui.Text(c, step).FontSize(12).TextColor(theme.TextSoft).Grow(1)
			})
		}
	})
}
