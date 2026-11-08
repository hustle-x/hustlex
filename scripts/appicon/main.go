// Command appicon draws the app's icons from the logo's mask
// (scripts/appicon/mark.png, the sidebar's mark at full resolution): the
// mark on a rounded square, white, laid out on Apple's icon grid, 1024
// pixels wide. The development
// app's (icon-dev.png) is blue, to tell the two apart in the Dock.
//
//	go run ./scripts/appicon
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
)

const (
	size  = 1024
	body  = 824 // the rounded square, as Apple's icon grid
	inset = (size - body) / 2
	ss    = 4 // samples per pixel side, for smooth edges
)

func main() {
	marks, err := readMark("scripts/appicon/mark.png")
	if err != nil {
		log.Fatal(err)
	}
	for name, colors := range map[string][3]color.RGBA{
		// White, the mark in the dark of the app's text.
		"icon.png": {{0xFF, 0xFF, 0xFF, 255}, {0xEC, 0xEC, 0xEE, 255}, {0x16, 0x15, 0x13, 255}},
		// The development app, blue.
		"icon-dev.png": {{0xA4, 0xC0, 0xFF, 255}, {0x7F, 0xA6, 0xFF, 255}, {0x0E, 0x0F, 0x12, 255}},
	} {
		img := draw(marks, colors[0], colors[1], colors[2])
		out := filepath.Join("cmd", "hustlex", "resources", name)
		f, err := os.Create(out)
		if err != nil {
			log.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			log.Fatal(err)
		}
		f.Close()
		fmt.Println(out)
	}
}

// readMark reads the mark's mask: a square of coverage, the logo as wide
// as the square.
func readMark(path string) (*image.Gray, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	g, ok := img.(*image.Gray)
	if !ok {
		return nil, fmt.Errorf("%s: want a grayscale mask", path)
	}
	return g, nil
}

func draw(mark *image.Gray, top, bottom, ink color.RGBA) *image.RGBA {
	// The mark's square, 28 points of the 36 the sidebar gives it,
	// centered.
	side := body * 28.0 / 36
	ox, oy := size/2-side/2, size/2-side/2
	cells := float64(mark.Bounds().Dx())
	// markCover is how much of (x, y) the mark covers, 0 to 1.
	markCover := func(x, y float64) float64 {
		mx, my := int((x-ox)/side*cells), int((y-oy)/side*cells)
		if !image.Pt(mx, my).In(mark.Bounds()) {
			return 0
		}
		return float64(mark.GrayAt(mx, my).Y) / 255
	}
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var r, g, b, a float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					x := float64(px) + (float64(sx)+0.5)/ss
					y := float64(py) + (float64(sy)+0.5)/ss
					// A soft shadow under the square.
					if !inSquircle(x, y) {
						if d := squircleDistance(x, y-12); d < 28 {
							a += 0.28 * (1 - d/28) * (1 - d/28)
						}
						continue
					}
					t := (y - inset) / body
					c := mix(top, bottom, t)
					if k := markCover(x, y); k > 0 {
						c = mix(c, ink, k)
					}
					r, g, b, a = r+float64(c.R), g+float64(c.G), b+float64(c.B), a+1
				}
			}
			n := float64(ss * ss)
			if a == 0 {
				continue
			}
			covered := math.Min(a, n)
			// Premultiplied: the shadow is black.
			img.SetRGBA(px, py, color.RGBA{
				R: uint8(r / n), G: uint8(g / n), B: uint8(b / n),
				A: uint8(math.Round(255 * covered / n)),
			})
		}
	}
	return img
}

// inSquircle reports whether (x, y) is inside the square, its corners
// rounded as the sidebar's mark's (8 points of 36).
func inSquircle(x, y float64) bool { return squircleDistance(x, y) == 0 }

// squircleDistance is how far (x, y) is outside the square, in pixels.
func squircleDistance(x, y float64) float64 {
	const half, radius = body / 2.0, body * 8.0 / 36
	dx := math.Max(math.Abs(x-size/2)-(half-radius), 0)
	dy := math.Max(math.Abs(y-size/2)-(half-radius), 0)
	return math.Max(math.Hypot(dx, dy)-radius, 0)
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Min(math.Max(t, 0), 1)
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}
