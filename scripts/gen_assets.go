package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

func main() {
	b, err := os.ReadFile("assets/icon/mouse.png")
	if err != nil {
		panic(err)
	}
	src, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	darkLogo := image.NewRGBA(bounds)
	lightLogo := image.NewRGBA(bounds)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if a > 0 {
				alpha := uint8(a >> 8)
				darkLogo.Set(x, y, color.RGBA{245, 245, 247, alpha})
				lightLogo.Set(x, y, color.RGBA{29, 29, 31, alpha})
			}
		}
	}

	appBadge := image.NewRGBA(image.Rect(0, 0, 160, 160))
	for y := 0; y < 160; y++ {
		for x := 0; x < 160; x++ {
			dx := float64(x - 80)
			dy := float64(y - 80)
			if dx*dx+dy*dy <= 72*72 {
				t := float64(y) / 160.0
				r := uint8(10*(1-t) + 0*t)
				g := uint8(132*(1-t) + 85*t)
				bl := uint8(255*(1-t) + 212*t)
				appBadge.Set(x, y, color.RGBA{r, g, bl, 255})
			}
		}
	}

	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			srcX := x * w / 80
			srcY := y * h / 80
			_, _, _, a := src.At(srcX, srcY).RGBA()
			if a > 0 {
				alpha := uint8(a >> 8)
				if alpha > 128 {
					appBadge.Set(x+40, y+40, color.RGBA{255, 255, 255, 255})
				}
			}
		}
	}

	savePNG("resources/logo-dark.png", darkLogo)
	savePNG("resources/logo-light.png", lightLogo)
	savePNG("resources/app-icon.png", appBadge)
}

func savePNG(path string, img draw.Image) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		panic(err)
	}
}
