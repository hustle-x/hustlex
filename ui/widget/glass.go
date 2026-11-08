package widget

import (
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// UseGlass paints what floats over the app's content, as the title bar's
// chips, menus and pop-overs, as Liquid Glass; off, as plain surfaces. The
// app sets it from the user's choice, every frame (main thread).
var UseGlass = true

// Glass fills e with Liquid Glass, shaped by its Radius, or with bg while
// glass is off.
func Glass(e ui.Element, bg ui.Color) ui.Element {
	if UseGlass {
		return e.Material(glass.Glass{})
	}
	return e.Background(bg)
}
