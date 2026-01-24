package imgkit

import (
	"golang.org/x/image/draw"
	"image"
	"image/color"
)

// Binaryzation process image with threshold value (0-255) and return new image.
// If useOriginalColors is true, pixels are kept in their original color or set to black,
// otherwise standard black/white binarization is applied.
func Binaryzation(src image.Image, threshold uint8, useOriginalColors bool) image.Image {
	if threshold < 0 || threshold > 255 {
		threshold = 128
	}

	bounds := src.Bounds()
	height, width := bounds.Max.Y-bounds.Min.Y, bounds.Max.X-bounds.Min.X

	if useOriginalColors {
		// Create new image with same type as source to preserve colors
		dst := image.NewRGBA(bounds)
		draw.Draw(dst, bounds, src, bounds.Min, draw.Src)

		return dst
	} else {
		// Original black/white binarization
		gray := Gray(src)
		for i := 0; i < height; i++ {
			for j := 0; j < width; j++ {
				if gray.At(j, i).(color.Gray).Y > threshold {
					gray.Set(j, i, color.White)
				} else {
					gray.Set(j, i, color.Black)
				}
			}
		}
		return gray
	}
}

func Gray(src image.Image) *image.Gray {
	bounds := src.Bounds()
	height, width := bounds.Max.Y-bounds.Min.Y, bounds.Max.X-bounds.Min.X
	gray := image.NewGray(bounds)

	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			c := color.GrayModel.Convert(src.At(j, i))
			gray.SetGray(j, i, c.(color.Gray))
		}
	}

	return gray
}

func Scale(src image.Image, rect image.Rectangle, scale draw.Scaler) image.Image {
	if scale == nil {
		scale = draw.ApproxBiLinear
	}

	dst := image.NewRGBA(rect)
	scale.Scale(dst, rect, src, src.Bounds(), draw.Over, nil)
	return dst
}
