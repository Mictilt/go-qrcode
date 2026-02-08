package standard

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"

	"github.com/yeqown/go-qrcode/writer/standard/imgkit"
	drawpkg "golang.org/x/image/draw"
)

// funcOption wraps a function that modifies outputImageOptions into an
// implementation of the ImageOption interface.
type funcOption struct {
	f func(oo *outputImageOptions)
}

func (fo *funcOption) apply(oo *outputImageOptions) {
	fo.f(oo)
}

func newFuncOption(f func(oo *outputImageOptions)) *funcOption {
	return &funcOption{
		f: f,
	}
}

// WithBgTransparent makes the background transparent.
func WithBgTransparent() ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		oo.bgTransparent = true
	})
}

// WithBgColor background color
func WithBgColor(c color.Color) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		if c == nil {
			return
		}

		oo.bgColor = parseFromColor(c)
	})
}

// WithBgColorRGBHex background color
func WithBgColorRGBHex(hex string) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		if hex == "" {
			return
		}

		oo.bgColor = parseFromHex(hex)
	})
}

// WithFgColor QR color
func WithFgColor(c color.Color) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		if c == nil {
			return
		}

		oo.qrColor = parseFromColor(c)
	})
}

// WithFgColorRGBHex Hex string to set QR Color
func WithFgColorRGBHex(hex string) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		oo.qrColor = parseFromHex(hex)
	})
}

// WithDataColor sets the color of every block except finder blocks
func WithDataColor(c color.Color) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		if c == nil {
			return
		}
		rgba := parseFromColor(c)
		if oo.qrColors == nil {
			oo.qrColors = &QRColors{}
		}
		oo.qrColors = oo.qrColors.withDataColor(rgba)
	})
}

// WithDataColorRGBHex sets the color of every block except finder blocks
func WithDataColorRGBHex(hex string) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		if oo.qrColors == nil {
			oo.qrColors = &QRColors{}
		}
		c := parseFromHex(hex)
		oo.qrColors = oo.qrColors.withDataColor(c)
	})
}

// WithFinderColor sets the color of finder blocks
func WithFinderColor(c color.Color) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		if c == nil {
			return
		}
		rgba := parseFromColor(c)
		if oo.qrColors == nil {
			oo.qrColors = &QRColors{}
		}
		oo.qrColors = oo.qrColors.withFinderColor(rgba)
	})
}

// WithFinderColorRGBHex sets the color of finder blocks
func WithFinderColorRGBHex(hex string) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		if oo.qrColors == nil {
			oo.qrColors = &QRColors{}
		}
		c := parseFromHex(hex)
		oo.qrColors = oo.qrColors.withFinderColor(c)
	})
}

// WithQRColors sets both data and finder colors using a QRColors struct.
// If qrColors is nil, both colors will use the qrColor.
// If only one field is set in QRColors, the other will use qrColor.
func WithQRColors(qrColors *QRColors) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		oo.qrColors = qrColors
	})
}

// WithQRColor sets both data and finder colors to the same color
func WithQRColor(c color.Color) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		if c == nil {
			return
		}
		rgba := parseFromColor(c)
		oo.qrColors = newQRColors(rgba)
	})
}

// WithQRColorRGBHex sets both data and finder colors to the same color using hex string
func WithQRColorRGBHex(hex string) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		rgba := parseFromHex(hex)
		oo.qrColors = newQRColors(rgba)
	})
}

// WithFgGradient QR gradient
func WithFgGradient(g *LinearGradient) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		if g == nil || len(g.Stops) == 0 {
			return
		}

		oo.qrGradient = g
	})
}

// WithLogoImage image should only has 1/5 width of QRCode at most
func WithLogoImage(img image.Image) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		if img == nil {
			return
		}

		oo.logo = img
	})
}

// WithLogoImageFileJPEG load image from file, jpeg is required.
// image should only have 1/5 width of QRCode at most
func WithLogoImageFileJPEG(f string) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		fd, err := os.Open(f)
		if err != nil {
			fmt.Printf("could not open file(%s), error=%v\n", f, err)
			return
		}
		defer fd.Close()

		img, err := jpeg.Decode(fd)
		if err != nil {
			fmt.Printf("could not open file(%s), error=%v\n", f, err)
			return
		}

		oo.logo = img
	})
}

// WithLogoImageFilePNG load image from file, PNG is required.
// image should only have 1/5 width of QRCode at most
func WithLogoImageFilePNG(f string) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		fd, err := os.Open(f)
		if err != nil {
			fmt.Printf("Open file(%s) failed: %v\n", f, err)
			return
		}
		defer fd.Close()

		img, err := png.Decode(fd)
		if err != nil {
			fmt.Printf("Decode file(%s) as PNG failed: %v\n", f, err)
			return
		}

		oo.logo = img
	})
}

// New function not from standard package
// WithLogoImageAdaptiveFileJPEG loads a JPEG image and scales it using high-quality
// CatmullRom (bicubic) interpolation for the best visual result.
// qrModules is the QR code dimension in modules (use qrc.Dimension() to get this value).
func WithLogoImageAdaptiveFileJPEG(f string, logoSizeMultiplier int, qrWidth uint8, qrModules int) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		fd, err := os.Open(f)
		if err != nil {
			fmt.Printf("could not open file(%s), error=%v\n", f, err)
			return
		}
		defer fd.Close()
		img, err := jpeg.Decode(fd)
		if err != nil {
			fmt.Printf("could not decode JPEG file(%s), error=%v\n", f, err)
			return
		}

		logoBounds := img.Bounds()
		logoWidthOriginal := float64(logoBounds.Dx())
		logoHeightOriginal := float64(logoBounds.Dy())

		// Calculate target size using actual QR module count
		// QR total size in pixels = qrModules * qrWidth
		// Logo should fit within 1/logoSizeMultiplier of the QR
		qrTotalSize := float64(qrModules) * float64(qrWidth)
		targetSize := (qrTotalSize / float64(logoSizeMultiplier))

		var logoWidth, logoHeight int
		if logoWidthOriginal > logoHeightOriginal {
			logoWidth = int(targetSize)
			logoHeight = int(targetSize * logoHeightOriginal / logoWidthOriginal)
		} else {
			logoHeight = int(targetSize)
			logoWidth = int(targetSize * logoWidthOriginal / logoHeightOriginal)
		}

		// Ensure minimum size of 1 pixel
		if logoWidth < 1 {
			logoWidth = 1
		}
		if logoHeight < 1 {
			logoHeight = 1
		}

		// Use CatmullRom (bicubic) for highest quality scaling
		// This preserves edges and details much better than BiLinear
		resized := image.NewRGBA(image.Rect(0, 0, logoWidth, logoHeight))
		drawpkg.CatmullRom.Scale(resized, resized.Bounds(), img, img.Bounds(), drawpkg.Over, nil)
		oo.logo = resized
	})
}

// New function not from standard package
// WithLogoImageAdaptiveFilePNG loads a PNG image and scales it using high-quality
// CatmullRom (bicubic) interpolation for the best visual result.
// Properly preserves transparency/alpha channel.
// qrModules is the QR code dimension in modules (use qrc.Dimension() to get this value).
func WithLogoImageAdaptiveFilePNG(f string, logoSizeMultiplier int, qrWidth uint8, qrModules int) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		fd, err := os.Open(f)
		if err != nil {
			fmt.Printf("could not open file(%s), error=%v\n", f, err)
			return
		}
		defer fd.Close()
		img, err := png.Decode(fd)
		if err != nil {
			fmt.Printf("could not decode PNG file(%s), error=%v\n", f, err)
			return
		}

		logoBounds := img.Bounds()
		logoWidthOriginal := float64(logoBounds.Dx())
		logoHeightOriginal := float64(logoBounds.Dy())

		// Calculate target size using actual QR module count
		// QR total size in pixels = qrModules * qrWidth
		// Logo should fit within 1/logoSizeMultiplier of the QR
		qrTotalSize := float64(qrModules) * float64(qrWidth)
		targetSize := (qrTotalSize / float64(logoSizeMultiplier))

		var logoWidth, logoHeight int
		if logoWidthOriginal > logoHeightOriginal {
			logoWidth = int(targetSize)
			logoHeight = int(targetSize * logoHeightOriginal / logoWidthOriginal)
		} else {
			logoHeight = int(targetSize)
			logoWidth = int(targetSize * logoWidthOriginal / logoHeightOriginal)
		}

		// Ensure minimum size of 1 pixel
		if logoWidth < 1 {
			logoWidth = 1
		}
		if logoHeight < 1 {
			logoHeight = 1
		}

		// Use CatmullRom (bicubic) for highest quality scaling
		// This preserves edges, details, and transparency much better than BiLinear
		resized := image.NewNRGBA(image.Rect(0, 0, logoWidth, logoHeight))
		drawpkg.CatmullRom.Scale(resized, resized.Bounds(), img, img.Bounds(), drawpkg.Over, nil)
		oo.logo = resized
	})
}

// WithQRWidth specify width of each qr block
func WithQRWidth(width uint8) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		oo.qrWidth = int(width)
	})
}

// WithCircleShape use circle shape as rectangle(default)
func WithCircleShape() ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		oo.shape = _shapeCircle
	})
}

// WithCustomShape use custom shape as rectangle(default)
func WithCustomShape(shape IShape) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		oo.shape = shape
	})
}

// WithBuiltinImageEncoder option includes: JPEG_FORMAT as default, PNG_FORMAT, SVG_FORMAT.
// This works like WithBuiltinImageEncoder, the different between them is
// formatTyp is enumerated in (JPEG_FORMAT, PNG_FORMAT, SVG_FORMAT)
func WithBuiltinImageEncoder(format formatTyp) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		var encoder ImageEncoder
		switch format {
		case JPEG_FORMAT:
			encoder = jpegEncoder{}
		case PNG_FORMAT:
			encoder = pngEncoder{}
		case SVG_FORMAT:
			encoder = svgEncoder{}
		default:
			panic("Not supported file format")
		}

		oo.imageEncoder = encoder
	})
}

// WithCustomImageEncoder to use custom image encoder to encode image.Image into
// io.Writer
func WithCustomImageEncoder(encoder ImageEncoder) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		if encoder == nil {
			return
		}

		oo.imageEncoder = encoder
	})
}

// WithBorderWidth specify the both 4 sides' border width. Notice that
// WithBorderWidth(a) means all border width use this variable `a`,
// WithBorderWidth(a, b) mean top/bottom equal to `a`, left/right equal to `b`.
// WithBorderWidth(a, b, c, d) mean top, right, bottom, left.
func WithBorderWidth(widths ...int) ImageOption {
	apply := func(arr *[4]int, top, right, bottom, left int) {
		arr[0] = top
		arr[1] = right
		arr[2] = bottom
		arr[3] = left
	}

	return newFuncOption(func(oo *outputImageOptions) {
		n := len(widths)
		switch n {
		case 0:
			apply(&oo.borderWidths, _defaultPadding, _defaultPadding, _defaultPadding, _defaultPadding)
		case 1:
			apply(&oo.borderWidths, widths[0], widths[0], widths[0], widths[0])
		case 2, 3:
			apply(&oo.borderWidths, widths[0], widths[1], widths[0], widths[1])
		default:
			// 4+
			apply(&oo.borderWidths, widths[0], widths[1], widths[2], widths[3])
		}
	})
}

// WithHalftone ...
func WithHalftone(path string) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		srcImg, err := imgkit.Read(path)
		if err != nil {
			fmt.Println("Read halftone image failed: ", err)
			return
		}

		oo.halftoneImg = srcImg
	})
}

// WithLogoSizeMultiplier used in Writer in validLogoImage method to validate logo size
func WithLogoSizeMultiplier(multiplier int) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		oo.logoSizeMultiplier = multiplier
	})
}

// WithLogoSafeZone enables the safe zone logic around the logo area.
func WithLogoSafeZone() ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		oo.logoSafeZone = true
	})
}

// WithResolution sets the output image size to resolution×resolution. For PNG/JPEG the QR is drawn at that size natively (sharp). For SVG the element is res×res with a viewBox.
func WithResolution(resolution *int) ImageOption {
	return newFuncOption(func(oo *outputImageOptions) {
		if resolution != nil {
			oo.resolution = resolution
		}
	})
}
