package imgkit_test

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yeqown/go-qrcode/writer/standard/imgkit"
)

func Test_Gray(t *testing.T) {
	t.Skipf("need human to check")

	img, err := imgkit.Read("testdata/test.png")
	assert.NoError(t, err)

	out := imgkit.Gray(img)
	assert.Equal(t, out.Bounds(), img.Bounds())
	imgkit.Save(out, "testdata/test_gray.png")
}

func TestBinaryzation(t *testing.T) {
	t.Skipf("need human to check")

	img, err := imgkit.Read("testdata/test.png")
	assert.NoError(t, err)

	out := imgkit.Binaryzation(img, 60, false)
	assert.Equal(t, out.Bounds(), img.Bounds())
	err = imgkit.Save(out, "testdata/test_binaryzation.png")
	assert.NoError(t, err)
}

func TestBinaryzationWithColors(t *testing.T) {
	img, err := imgkit.Read("../testdata/test.jpeg")
	assert.NoError(t, err)

	// Test with custom colors: pixels below threshold become black, others keep original colors
	out := imgkit.Binaryzation(img, 60, true)
	assert.Equal(t, out.Bounds(), img.Bounds())

	// Verify that the output image has the same dimensions
	assert.Equal(t, img.Bounds().Dx(), out.Bounds().Dx())
	assert.Equal(t, img.Bounds().Dy(), out.Bounds().Dy())

	// Verify that some pixels are black (below threshold) and some retain original colors (above threshold)
	blackCount := 0
	colorCount := 0
	bounds := out.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := out.At(x, y).RGBA()
			if r == 0 && g == 0 && b == 0 && a == 0xffff {
				blackCount++
			} else {
				colorCount++
			}
		}
	}

	// Ensure we have both black and colored pixels
	assert.True(t, blackCount > 0, "Should have some black pixels from binaryzation")
	assert.True(t, colorCount > 0, "Should have some colored pixels preserved")

	err = imgkit.Save(out, "../testdata/test_binaryzation_colors.png")
	assert.NoError(t, err)
}

func TestScale(t *testing.T) {
	t.Skipf("need human to check")

	img, err := imgkit.Read("testdata/test_binaryzation.png")
	assert.NoError(t, err)

	out := imgkit.Scale(img, image.Rect(0, 0, 100, 100), nil)
	assert.Equal(t, out.Bounds(), image.Rect(0, 0, 100, 100))
	err = imgkit.Save(out, "testdata/test_binaryzation_scale.png")
	assert.NoError(t, err)
}
