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

	// Test SVG with small logo (100x100 should be valid for 740x740 QR code)
	w, err := standard.New("./test_logo.svg",
		standard.WithQRWidth(20),
		standard.WithLogoImageFileJPEG("./test_small.jpeg"),
		standard.WithBuiltinImageEncoder(standard.SVG_FORMAT),
	)
	if err != nil {
		panic(err)
	}
	if err = qrc.Save(w); err != nil {
		panic(err)
	}

	println("Test logo SVG created successfully!")
}
