package main

import (
	"github.com/veandco/go-sdl2/sdl"
)

var (
	/// Texture containing a predefined font for debugging, etc.
	///
	Font *sdl.Texture
)

/// InitFont loads the bitmap surface with font on it.
///
func InitFont() {
	var surface *sdl.Surface
	var err error

	if surface, err = sdl.LoadBMP("font.bmp"); err != nil {
		panic(err)
	}

	// get the magenta color
	mask := sdl.MapRGB(surface.Format, 255, 0, 255)

	// set the mask color key
	surface.SetColorKey(1, mask)

	// create the texture
	if Font, err = Renderer.CreateTextureFromSurface(surface); err != nil {
		panic(err)
	}
}

/// DrawText using the loaded font.
///
func DrawText(s string, x, y int) {
	src := sdl.Rect{W: 5, H: 7}
	dst := sdl.Rect{
		X: int32(x),
		Y: int32(y),
		W: 5,
		H: 7,
	}

	// loop over all the characters in the string
	for _, c := range s {
		if c > 32 && c < 94 {
			src.X = (c - 33) * 6

			// draw the character to the renderer
			Renderer.Copy(Font, &src, &dst)
		}

		// advance
		dst.X += 7
	}
}
