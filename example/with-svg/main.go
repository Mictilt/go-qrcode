package main

import (
	"image/color"

	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
	"github.com/yeqown/go-qrcode/writer/standard/shapes"
)

func main() {
	qrc, err := qrcode.NewWith("https://github.com/yeqown/go-qrcode",
		qrcode.WithEncodingMode(qrcode.EncModeByte),
		qrcode.WithErrorCorrectionLevel(qrcode.ErrorCorrectionQuart),
	)
	if err != nil {
		panic(err)
	}

	// save QR code as SVG file
	w, err := standard.New("./qrcode.svg",
		standard.WithQRWidth(20),
		standard.WithBuiltinImageEncoder(standard.SVG_FORMAT),
	)
	if err != nil {
		panic(err)
	}
	if err = qrc.Save(w); err != nil {
		panic(err)
	}

	// You can also customize colors for SVG output
	w2, err := standard.New("./qrcode_colored.svg",
		standard.WithQRWidth(20),
		standard.WithFgColorRGBHex("#FF0000"), // Red QR code
		standard.WithBgColorRGBHex("#FFFFFF"), // White background
		standard.WithBuiltinImageEncoder(standard.SVG_FORMAT),
	)
	if err != nil {
		panic(err)
	}
	if err = qrc.Save(w2); err != nil {
		panic(err)
	}

	// Test SVG with rectangle shapes (default)
	w3, err := standard.New("./qrcode_rect.svg",
		standard.WithQRWidth(10),
		standard.WithBuiltinImageEncoder(standard.SVG_FORMAT),
	)
	if err != nil {
		panic(err)
	}
	if err = qrc.Save(w3); err != nil {
		panic(err)
	}

	// Test SVG with circle shapes
	w4, err := standard.New("./qrcode_circle.svg",
		standard.WithQRWidth(10),
		standard.WithCircleShape(),
		standard.WithBuiltinImageEncoder(standard.SVG_FORMAT),
	)
	if err != nil {
		panic(err)
	}
	if err = qrc.Save(w4); err != nil {
		panic(err)
	}

	// Test SVG with liquid block shapes (complex shape)
	w6, err := standard.New("./qrcode_liquid.svg",
		standard.WithQRWidth(10),
		standard.WithCustomShape(shapes.Assemble(shapes.SquareFinder(), shapes.LiquidBlock())),
		standard.WithBuiltinImageEncoder(standard.SVG_FORMAT),
	)
	if err != nil {
		panic(err)
	}
	if err = qrc.Save(w6); err != nil {
		panic(err)
	}

	// Test SVG with PlanetFinder (custom finder with complex stroke operations)
	w9, err := standard.New("./qrcode_planet.svg",
		standard.WithQRWidth(10),
		standard.WithCustomShape(shapes.Assemble(shapes.SquareFinder(), shapes.LiquidBlock())),
		standard.WithBuiltinImageEncoder(standard.SVG_FORMAT),
	)
	if err != nil {
		panic(err)
	}
	if err = qrc.Save(w9); err != nil {
		panic(err)
	}

	// Test SVG with gradient colors
	gradient := standard.NewGradient(45, standard.ColorStop{T: 0.0, Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}}, standard.ColorStop{T: 1.0, Color: color.RGBA{R: 0, G: 0, B: 255, A: 255}})
	w7, err := standard.New("./qrcode_gradient.svg",
		standard.WithQRWidth(10),
		standard.WithFgGradient(gradient),
		standard.WithBuiltinImageEncoder(standard.SVG_FORMAT),
	)
	if err != nil {
		panic(err)
	}
	if err = qrc.Save(w7); err != nil {
		panic(err)
	}

	// Test SVG with halftone
	w8, err := standard.New("./qrcode_halftone.svg",
		standard.WithQRWidth(10),
		standard.WithHalftone("../with-halftone/test.jpeg"),
		standard.WithBuiltinImageEncoder(standard.SVG_FORMAT),
	)
	if err != nil {
		panic(err)
	}
	if err = qrc.Save(w8); err != nil {
		panic(err)
	}

	// Test SVG with halftone and circle shapes
	w10, err := standard.New("./qrcode_halftone_circle.svg",
		standard.WithQRWidth(10),
		standard.WithCircleShape(),
		standard.WithHalftone("../with-halftone/test.jpeg"),
		standard.WithBuiltinImageEncoder(standard.SVG_FORMAT),
	)
	if err != nil {
		panic(err)
	}
	if err = qrc.Save(w10); err != nil {
		panic(err)
	}

	// Test SVG with halftone and liquid block shapes (complex shape)
	w11, err := standard.New("./qrcode_halftone_liquid.svg",
		standard.WithQRWidth(10),
		standard.WithCustomShape(shapes.Assemble(shapes.SquareFinder(), shapes.LiquidBlock())),
		standard.WithHalftone("../with-halftone/test.jpeg"),
		standard.WithBuiltinImageEncoder(standard.SVG_FORMAT),
	)
	if err != nil {
		panic(err)
	}
	if err = qrc.Save(w11); err != nil {
		panic(err)
	}

	// Test SVG with logo (logo will be embedded as image element)
	w5, err := standard.New("./qrcode_with_logo.svg",
		standard.WithQRWidth(20),
		standard.WithLogoImageFileJPEG("../with-halftone/test.jpeg"),
		standard.WithBuiltinImageEncoder(standard.SVG_FORMAT),
	)
	if err != nil {
		panic(err)
	}
	if err = qrc.Save(w5); err != nil {
		panic(err)
	}

	println("SVG files created successfully!")
}
