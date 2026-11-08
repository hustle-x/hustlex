package theme

import (
	"math"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// Palette is one theme's colors: Dark or Light.
type Palette struct {
	ID, Name string
	Light    bool // a light ground: shadows, overlays and terminals follow
	// Ground is what the panels sit on; zero derives it from Background.
	Ground ui.Color

	Background, Panel, Raised, Sunken ui.Color
	Hover, Selected, Dialog           ui.Color
	Border, BorderStrong              ui.Color
	Text, TextStrong, TextSoft, Muted ui.Color
	Accent, OnAccent                  ui.Color
	Working, Done, Warning, Error     ui.Color
	Idle                              ui.Color
}

// The built-in palettes. Status colors keep their meaning in every one:
// blue works, yellow waits, green is done, red failed.
var (
	// Obsidian is the dark mode: flat near-black panels on a darker ground,
	// with a coral accent.
	Obsidian = Palette{
		ID: "obsidian", Name: "Dark",
		Ground:     ui.Hex("#000000"),
		Background: ui.Hex("#080808"), Panel: ui.Hex("#0B0B0B"), Raised: ui.Hex("#141414"), Sunken: ui.Hex("#050505"),
		Hover: ui.Hex("#121212"), Selected: ui.Hex("#171717"), Dialog: ui.Hex("#0E0E0E"),
		Border: ui.Hex("#171717"), BorderStrong: ui.Hex("#242424"),
		Text: ui.Hex("#F0F0F0"), TextStrong: ui.Hex("#FFFFFF"), TextSoft: ui.Hex("#C4C4C4"), Muted: ui.Hex("#878787"),
		Accent: ui.Hex("#FF6B5E"), OnAccent: ui.Hex("#2A0905"),
		Working: ui.Hex("#9DB6F5"), Done: ui.Hex("#86D0A8"), Warning: ui.Hex("#E6C47E"), Error: ui.Hex("#EE8C8C"), Idle: ui.Hex("#555555"),
	}
)

// Daylight is the light mode: plain white and neutral grays, no warm tint.
var Daylight = Palette{
	ID: "daylight", Name: "Light", Light: true,
	Background: ui.Hex("#F2F2F3"), Panel: ui.Hex("#FAFAFA"), Raised: ui.Hex("#FFFFFF"), Sunken: ui.Hex("#F7F7F8"),
	Hover: ui.Hex("#EBEBED"), Selected: ui.Hex("#E3E3E6"), Dialog: ui.Hex("#FFFFFF"),
	Border: ui.Hex("#E4E4E7"), BorderStrong: ui.Hex("#D2D2D7"),
	Text: ui.Hex("#18181B"), TextStrong: ui.Hex("#000000"), TextSoft: ui.Hex("#47474D"), Muted: ui.Hex("#85858D"),
	Accent: ui.Hex("#18181B"), OnAccent: ui.Hex("#FFFFFF"),
	Working: ui.Hex("#3B66D8"), Done: ui.Hex("#2E9C68"), Warning: ui.Hex("#B7791F"), Error: ui.Hex("#C43D3D"), Idle: ui.Hex("#A1A1AA"),
}

// Palettes is every built-in palette, the default first.
var Palettes = []Palette{Obsidian, Daylight}

// Accents are the colors a user can give any palette instead of its own.
var Accents = []ui.Color{
	ui.Hex("#FF6B5E"), // coral
	ui.Hex("#D8C3A0"), // champagne
	ui.Hex("#E2E4E9"), // silver
	ui.Hex("#9DB6F5"), // blue
	ui.Hex("#B4A4F2"), // violet
	ui.Hex("#E9A6B8"), // rose
	ui.Hex("#D49A6A"), // copper
	ui.Hex("#E3C07A"), // amber
	ui.Hex("#8FCFA5"), // sage
	ui.Hex("#7CCBD3"), // teal
}

// Lookup is the palette with the given ID, else the default; a non-zero
// accent replaces the palette's own.
func Lookup(id string, accent ui.Color) Palette {
	p := Palettes[0]
	for _, q := range Palettes {
		if q.ID == id {
			p = q
		}
	}
	if accent.A != 0 {
		p.Accent = accent
		p.OnAccent = onColor(accent)
	}
	return p
}

// Terminal is the terminals' colors for the current palette: its ground
// and text, with ANSI colors tuned to it.
func Terminal() *terminal.Theme {
	t := terminal.DarkTheme()
	if current.Light {
		t = terminal.LightTheme()
	}
	fg := Mix(Text, Background, 0.08)
	t.Foreground, t.Background = fg, Background
	t.Cursor = Accent
	t.Selection = Fade(Accent, 0.25)
	t.Palette = [16]ui.Color{
		BorderStrong, Error, Done, Warning,
		Working, ui.Hex("#C7A4E8"), ui.Hex("#7CCBD3"), fg,
		Idle, Mix(Error, TextStrong, 0.2), Mix(Done, TextStrong, 0.2), Mix(Warning, TextStrong, 0.2),
		Mix(Working, TextStrong, 0.2), ui.Hex("#DAB8F0"), ui.Hex("#9DDCE3"), TextStrong,
	}
	return t
}

// onColor is the text color that reads on bg: near-black on a light
// color, white on a dark one.
func onColor(bg ui.Color) ui.Color {
	lin := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	if 0.2126*lin(bg.R)+0.7152*lin(bg.G)+0.0722*lin(bg.B) > 0.18 {
		return ui.Hex("#0B0C0E")
	}
	return ui.Hex("#FFFFFF")
}
