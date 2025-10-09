package main

import (
	"image/color"

	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

func main() {
	qrc, err := qrcode.NewWith("https://github.com/yeqown/go-qrcode",
		qrcode.WithEncodingMode(qrcode.EncModeByte),
		qrcode.WithErrorCorrectionLevel(qrcode.ErrorCorrectionQuart),
	)
	if err != nil {
		panic(err)
	}

	// Create a QR code with different colors for data and finder elements
	// - Data blocks (information points) and all other non-finder elements: Blue
	// - Finder patterns (anchor points): Red
	w, err := standard.New("element-colors-qr.png",
		standard.WithQRWidth(20),
		standard.WithDataColorRGBHex("#0066CC"),   // Blue for all non-finder elements
		standard.WithFinderColorRGBHex("#CC0000"), // Red for finder patterns
	)
	if err != nil {
		panic(err)
	}

	if err = qrc.Save(w); err != nil {
		panic(err)
	}

	// You can also use color.Color types instead of hex strings
	// standard.WithDataColor(color.RGBA{R: 0, G: 102, B: 204, A: 255}),
	// standard.WithFinderColor(color.RGBA{R: 204, G: 0, B: 0, A: 255}),

	// NEW: You can also use the QRColors struct to set both colors at once
	qrc2, err := qrcode.NewWith("https://github.com/yeqown/go-qrcode/new-feature",
		qrcode.WithEncodingMode(qrcode.EncModeByte),
		qrcode.WithErrorCorrectionLevel(qrcode.ErrorCorrectionQuart),
	)
	if err != nil {
		panic(err)
	}

	// Using QRColors struct - both data and finder will be green
	qrColors := &standard.QRColors{
		Data:   &color.RGBA{R: 0, G: 255, B: 0, A: 255}, // Green for data
		Finder: &color.RGBA{R: 255, G: 0, B: 0, A: 255}, // Red for finder
	}

	w2, err := standard.New("element-colors-qr-struct.png",
		standard.WithQRWidth(20),
		standard.WithQRColors(qrColors),
	)
	if err != nil {
		panic(err)
	}

	if err = qrc2.Save(w2); err != nil {
		panic(err)
	}

	// Or use WithQRColor to set both data and finder to the same color
	w3, err := standard.New("element-colors-same.png",
		standard.WithQRWidth(20),
		standard.WithQRColor(color.RGBA{R: 255, G: 0, B: 255, A: 255}), // Magenta for both
	)
	if err != nil {
		panic(err)
	}

	if err = qrc.Save(w3); err != nil {
		panic(err)
	}

	println("QR code with data and finder element colors saved as 'element-colors-qr.png'")
	println("QR code with QRColors struct saved as 'element-colors-qr-struct.png'")
	println("QR code with same color for both saved as 'element-colors-same.png'")
}
