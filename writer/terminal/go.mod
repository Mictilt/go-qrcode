module github.com/Mictilt/go-qrcode/writer/terminal

go 1.17

require (
	github.com/mattn/go-runewidth v0.0.9
	github.com/nsf/termbox-go v1.1.1
	github.com/Mictilt/go-qrcode/v2 v2.2.5
)

require github.com/yeqown/reedsolomon v1.0.0 // indirect

//replace github.com/Mictilt/go-qrcode/v2 => ../../
