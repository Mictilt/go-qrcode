package standard

import (
	"image/color"

	"github.com/fogleman/gg"
)

var (
	_shapeRectangle IShape = rectangle{}
	_shapeCircle    IShape = circle{}
)

type IShape interface {
	// Draw the shape of QRCode block in IShape implemented way.
	Draw(ctx *DrawContext)

	// DrawFinder to fill the finder pattern of QRCode, what's finder? google it for more information.
	DrawFinder(ctx *DrawContext)
}

// GraphicsContext defines the interface for graphics operations
type GraphicsContext interface {
	MoveTo(x, y float64)
	LineTo(x, y float64)
	QuadraticTo(cx, cy, x, y float64)
	ClosePath()
	DrawCircle(cx, cy, radius float64)
	DrawRectangle(x, y, w, h float64)
	SetColor(c color.Color)
	Fill()
	SetDash(dashes ...float64)
	SetLineWidth(lineWidth float64)
	SetLineCap(lineCap gg.LineCap)
	SetLineCapSquare()
	Stroke()
	SetFillRuleEvenOdd()
	NewSubPath()
	SetFillRuleWinding()
}

// GGContextWrapper wraps gg.Context to implement GraphicsContext
type GGContextWrapper struct {
	*gg.Context
}

func (wrapper *GGContextWrapper) MoveTo(x, y float64) {
	wrapper.Context.MoveTo(x, y)
}

func (wrapper *GGContextWrapper) LineTo(x, y float64) {
	wrapper.Context.LineTo(x, y)
}

func (wrapper *GGContextWrapper) QuadraticTo(cx, cy, x, y float64) {
	wrapper.Context.QuadraticTo(cx, cy, x, y)
}

func (wrapper *GGContextWrapper) ClosePath() {
	wrapper.Context.ClosePath()
}

func (wrapper *GGContextWrapper) DrawCircle(cx, cy, radius float64) {
	wrapper.Context.DrawCircle(cx, cy, radius)
}

func (wrapper *GGContextWrapper) DrawRectangle(x, y, width, height float64) {
	wrapper.Context.DrawRectangle(x, y, width, height)
}

func (wrapper *GGContextWrapper) SetColor(c color.Color) {
	wrapper.Context.SetColor(c)
}

func (wrapper *GGContextWrapper) Fill() {
	wrapper.Context.Fill()
}

func (wrapper *GGContextWrapper) SetDash(dashes ...float64) {
	wrapper.Context.SetDash(dashes...)
}

func (wrapper *GGContextWrapper) SetLineWidth(lineWidth float64) {
	wrapper.Context.SetLineWidth(lineWidth)
}

func (wrapper *GGContextWrapper) SetLineCap(lineCap gg.LineCap) {
	wrapper.Context.SetLineCap(lineCap)
}

func (wrapper *GGContextWrapper) SetLineCapSquare() {
	wrapper.Context.SetLineCapSquare()
}

func (wrapper *GGContextWrapper) Stroke() {
	wrapper.Context.Stroke()
}

func (wrapper *GGContextWrapper) SetFillRuleEvenOdd() {
	wrapper.Context.SetFillRuleEvenOdd()
}

func (wrapper *GGContextWrapper) NewSubPath() {
	wrapper.Context.NewSubPath()
}

func (wrapper *GGContextWrapper) SetFillRuleWinding() {
	wrapper.Context.SetFillRuleWinding()
}

// DrawContext is a rectangle area
type DrawContext struct {
	GraphicsContext

	x, y float64
	w, h int

	color      color.Color
	neighbours uint16
}

// UpperLeft returns the point which indicates the upper left position.
func (dc *DrawContext) UpperLeft() (dx, dy float64) {
	return dc.x, dc.y
}

// Edge returns width and height of each shape could take at most.
func (dc *DrawContext) Edge() (width, height int) {
	return dc.w, dc.h
}

// Bit flags for the 8 surrounding cells in a 3x3 grid around the center (x, y).
// Layout:
// NTopLeft		NTop 	NTopRight
// NLeft  		NSelf	NRight
// NBotLeft 	NBot 	NBotRight
const (
	NTopLeft  uint16 = 1 << iota // top-left
	NTop                         // top
	NTopRight                    // top-right
	NLeft                        // left
	NSelf                        // center (self)
	NRight                       // right
	NBotLeft                     // bottom-left
	NBot                         // bottom
	NBotRight                    // bottom-right
)

// Neighbours returns a bitmask representing the neighboring blocks of the current block
func (dc *DrawContext) Neighbours() uint16 {
	return dc.neighbours
}

// Color returns the color which should be fill into the shape. Note that if you're not
// using this color but your coded color.Color, some ImageOption functions those set foreground color
// would take no effect.
func (dc *DrawContext) Color() color.Color {
	return dc.color
}

// rectangle IShape
type rectangle struct{}

func (r rectangle) Draw(c *DrawContext) {
	// FIXED(@yeqown): miss parameter of DrawRectangle
	c.DrawRectangle(c.x, c.y, float64(c.w), float64(c.h))
	c.SetColor(c.color)
	c.Fill()
}

func (r rectangle) DrawFinder(ctx *DrawContext) {
	r.Draw(ctx)
}

// circle IShape
type circle struct{}

// Draw
// FIXED: Draw could not draw circle
func (r circle) Draw(c *DrawContext) {
	// choose a proper radius values
	radius := c.w / 2
	r2 := c.h / 2
	if r2 <= radius {
		radius = r2
	}

	cx, cy := c.x+float64(c.w)/2.0, c.y+float64(c.h)/2.0 // get center point
	c.DrawCircle(cx, cy, float64(radius))
	c.SetColor(c.color)
	c.Fill()
}

func (r circle) DrawFinder(ctx *DrawContext) {
	r.Draw(ctx)
}
