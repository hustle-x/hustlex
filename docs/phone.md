# hustleX on your phone

A host's daemon can serve hustleX's web app to your phones: the Inbox (what your agents wait on, answered from the phone), your sessions (their last messages, and a box to write to them) and your accounts' limits. No App Store and no cloud: the phone reaches the host over [Tailscale](https://tailscale.com), and you add the page to its Home Screen to open it like an app.

- **Settings → App → Your phone** turns it on for the connected host and lists the phones paired, each to forget. **Pair a phone…** (also in ⌘K) shows a QR code: scan it with the phone's camera, check the phone shows the same six digits as the computer, tap Pair, then Share → Add to Home Screen.
- The daemon listens on the host's Tailscale address only, port 7651 (`daemon/phone.go`), so only devices on your tailnet reach it. Requests naming another host are refused, writes must come from the page's own script, and every read or write needs a paired phone's secret, kept in an HttpOnly cookie. The host keeps only a hash of each secret (`core/phonepair`, `phone.json` in the daemon's folder).
- A pairing code works once, for ten minutes. Forgetting a phone, or turning the web app off, locks it out at once.
- It needs Tailscale on the host and the phone, signed in to the same tailnet; without it, Settings says so. Traffic between them goes over Tailscale's WireGuard tunnel.
