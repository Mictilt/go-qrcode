# SVG QR Code Example

This example demonstrates how to generate QR codes in SVG format using the go-qrcode library.

## Features

- Generate QR codes as scalable SVG files
- Support for custom colors and styling
- Vector graphics that scale perfectly at any size

## Usage

```go
package main

import (
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

	// Generate black QR code on white background
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

	// Generate colored QR code
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
}
```

## Running the Example

```bash
go run main.go
```

This will generate two SVG files:
- `qrcode.svg` - A standard black and white QR code
- `qrcode_colored.svg` - A red QR code on white background

## Benefits of SVG Format

- **Scalable**: SVG graphics can be resized without loss of quality
- **Small file size**: Especially for simple QR codes
- **Web-friendly**: Can be embedded directly in HTML
- **Editable**: Can be modified with CSS or JavaScript
- **Print-ready**: Perfect for high-resolution printing
