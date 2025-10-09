package main

import (
	"os"

	"github.com/Mictilt/go-qrcode/v2"
	"github.com/Mictilt/go-qrcode/writer/file"
)

func main() {
	qrc, err := qrcode.New("with_file_writer")
	if err != nil {
		panic(err)
	}

	w := file.New(os.Stdout)
	if err = qrc.Save(w); err != nil {
		panic(err)
	}
}
