package standard

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"strings"

	svgo "github.com/ajstarks/svgo"
	"github.com/fogleman/gg"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard/imgkit"
)

type formatTyp uint8

const (
	// JPEG_FORMAT as default output file format.
	JPEG_FORMAT formatTyp = iota
	// PNG_FORMAT .
	PNG_FORMAT
	// SVG_FORMAT .
	SVG_FORMAT
)

// ImageEncoder is an interface which describes the rule how to encode image.Image into io.Writer
type ImageEncoder interface {
	// Encode specify which format to encode image into io.Writer.
	Encode(w io.Writer, img image.Image) error
}

// ImageEncoderWithOptions is an extended interface for encoders that can accept additional options
type ImageEncoderWithOptions interface {
	ImageEncoder
	// EncodeWithOptions encodes image with additional options
	EncodeWithOptions(w io.Writer, img image.Image, opts *outputImageOptions) error
}

// ImageEncoderWithMatrix is an interface for encoders that can work directly with QR matrix
type ImageEncoderWithMatrix interface {
	ImageEncoder
	// EncodeMatrix encodes QR matrix directly to SVG with options
	EncodeMatrix(w io.Writer, mat qrcode.Matrix, opts *outputImageOptions) error
}

type jpegEncoder struct{}

func (j jpegEncoder) Encode(w io.Writer, img image.Image) error {
	return jpeg.Encode(w, img, nil)
}

type pngEncoder struct{}

func (j pngEncoder) Encode(w io.Writer, img image.Image) error {
	return png.Encode(w, img)
}

// SVGShape interface for generating SVG path elements
type SVGShape interface {
	// GenerateSVGPath returns SVG path data for a block using DrawContext
	GenerateSVGPath(ctx *DrawContext, hasGradient bool) string

	// GenerateSVGFinder returns SVG path data for finder patterns using DrawContext
	GenerateSVGFinder(ctx *DrawContext, hasGradient bool) string
}

// svgRectangle generates SVG rectangle paths
type svgRectangle struct{}

func (s svgRectangle) GenerateSVGPath(ctx *DrawContext, hasGradient bool) string {
	x, y := ctx.UpperLeft()
	w, h := ctx.Edge()
	return fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`, x, y, float64(w), float64(h))
}

func (s svgRectangle) GenerateSVGFinder(ctx *DrawContext, hasGradient bool) string {
	return s.GenerateSVGPath(ctx, hasGradient)
}

// svgCircle generates SVG circle paths
type svgCircle struct{}

func (s svgCircle) GenerateSVGPath(ctx *DrawContext, hasGradient bool) string {
	x, y := ctx.UpperLeft()
	w, h := ctx.Edge()
	cx := x + float64(w)/2.0
	cy := y + float64(h)/2.0
	radius := float64(w) / 2
	if float64(h)/2 < radius {
		radius = float64(h) / 2
	}
	return fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="%.2f"/>`, cx, cy, radius)
}

func (s svgCircle) GenerateSVGFinder(ctx *DrawContext, hasGradient bool) string {
	return s.GenerateSVGPath(ctx, hasGradient)
}

// svgPathShape wraps any IShape and generates SVG path data by recording drawing operations
type svgPathShape struct {
	shape IShape
}

func (s svgPathShape) GenerateSVGPath(ctx *DrawContext, hasGradient bool) string {
	// Create a path recorder that implements GraphicsContext
	recorder := NewSVGPathRecorder(hasGradient)

	// Create a temporary DrawContext with the recorder as the GraphicsContext
	tempCtx := &DrawContext{
		GraphicsContext: recorder,
		x:               ctx.x,
		y:               ctx.y,
		w:               ctx.w,
		h:               ctx.h,
		color:           ctx.color,
		neighbours:      ctx.neighbours,
	}

	// Call the original shape's Draw method
	s.shape.Draw(tempCtx)

	// Return the recorded path data
	return recorder.toSVGPath()
}

func (s svgPathShape) GenerateSVGFinder(ctx *DrawContext, hasGradient bool) string {
	// Create a path recorder that implements GraphicsContext
	recorder := NewSVGPathRecorder(hasGradient)

	// Create a temporary DrawContext with the recorder as the GraphicsContext
	tempCtx := &DrawContext{
		GraphicsContext: recorder,
		x:               ctx.x,
		y:               ctx.y,
		w:               ctx.w,
		h:               ctx.h,
		color:           ctx.color,
		neighbours:      ctx.neighbours,
	}

	// Call the original shape's DrawFinder method
	s.shape.DrawFinder(tempCtx)

	// Return the recorded path data
	return recorder.toSVGPath()
}

// svgPathRecorder records drawing operations and converts them to SVG path commands
type svgPathRecorder struct {
	pathElements       []string // Store complete path elements
	currentCommands    []string // Current path commands being built
	currentX, currentY float64

	// Stroke attributes
	strokeDash    []float64
	strokeWidth   float64
	strokeLineCap string

	// Fill attributes
	fillRule string

	// Current color
	currentColor color.Color

	// Gradient support
	hasGradient bool // Whether gradients are enabled
}

// Implement GraphicsContext interface
func (r *svgPathRecorder) MoveTo(x, y float64) {
	r.currentCommands = append(r.currentCommands, fmt.Sprintf("M%.2f %.2f", x, y))
	r.currentX, r.currentY = x, y
}

func (r *svgPathRecorder) LineTo(x, y float64) {
	r.currentCommands = append(r.currentCommands, fmt.Sprintf("L%.2f %.2f", x, y))
	r.currentX, r.currentY = x, y
}

func (r *svgPathRecorder) QuadraticTo(cx, cy, x, y float64) {
	r.currentCommands = append(r.currentCommands, fmt.Sprintf("Q%.2f %.2f %.2f %.2f", cx, cy, x, y))
	r.currentX, r.currentY = x, y
}

func (r *svgPathRecorder) ClosePath() {
	r.currentCommands = append(r.currentCommands, "Z")
}

func (r *svgPathRecorder) DrawCircle(cx, cy, radius float64) {
	// Always add to current commands so Fill() can work properly
	r.currentCommands = append(r.currentCommands,
		fmt.Sprintf("M%.2f %.2f A%.2f %.2f 0 1 1 %.2f %.2f A%.2f %.2f 0 1 1 %.2f %.2f Z",
			cx+radius, cy, radius, radius, cx-radius, cy, radius, radius, cx+radius, cy))
	r.currentX, r.currentY = cx+radius, cy
}

func (r *svgPathRecorder) DrawRectangle(x, y, w, h float64) {
	// Always add to current commands so Fill() can work properly
	r.currentCommands = append(r.currentCommands,
		fmt.Sprintf("M%.2f %.2f L%.2f %.2f L%.2f %.2f L%.2f %.2f Z",
			x, y, x+w, y, x+w, y+h, x, y+h))
	r.currentX, r.currentY = x, y+h
}

func (r *svgPathRecorder) SetColor(c color.Color) {
	r.currentColor = c
}

func (r *svgPathRecorder) Fill() {
	// Complete the current path when Fill() is called
	if len(r.currentCommands) > 0 {
		pathData := strings.Join(r.currentCommands, " ")
		var colorStr string
		if !r.hasGradient {
			// Only set fill color if gradients are not enabled
			colorStr = r.colorToHex(r.currentColor)
		}
		r.pathElements = append(r.pathElements, r.generateSVGPathElement(pathData, "fill", colorStr))
		r.currentCommands = nil // Reset for next path
	}
	// Reset stroke attributes after fill operation
	r.strokeDash = nil
	r.strokeWidth = 0
	r.strokeLineCap = ""
}

func (r *svgPathRecorder) Stroke() {
	// Complete the current path when Stroke() is called
	if len(r.currentCommands) > 0 {
		pathData := strings.Join(r.currentCommands, " ")
		colorStr := r.colorToHex(r.currentColor)
		r.pathElements = append(r.pathElements, r.generateSVGPathElement(pathData, "stroke", colorStr))
		r.currentCommands = nil // Reset for next path
	}
	// Reset stroke attributes after stroke operation
	r.strokeDash = nil
	r.strokeWidth = 0
	r.strokeLineCap = ""
}

func (r *svgPathRecorder) SetDash(dashes ...float64) {
	r.strokeDash = dashes
}

func (r *svgPathRecorder) SetLineWidth(lineWidth float64) {
	r.strokeWidth = lineWidth
}

func (r *svgPathRecorder) SetLineCap(lineCap gg.LineCap) {
	switch lineCap {
	case 0: // gg.LineCapButt
		r.strokeLineCap = "butt"
	case 1: // gg.LineCapRound
		r.strokeLineCap = "round"
	case 2: // gg.LineCapSquare
		r.strokeLineCap = "square"
	default:
		r.strokeLineCap = "butt"
	}
}

func (r *svgPathRecorder) SetLineCapSquare() {
	r.strokeLineCap = "square"
}

func (r *svgPathRecorder) SetFillRuleEvenOdd() {
	r.fillRule = "evenodd"
}

func (r *svgPathRecorder) SetFillRuleWinding() {
	r.fillRule = "nonzero"
}

func (r *svgPathRecorder) NewSubPath() {
	// For SVG, we might need to handle subpaths differently
	// For now, we'll just continue the path
}

func (r *svgPathRecorder) colorToHex(c color.Color) string {
	if c == nil {
		return ""
	}
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	return fmt.Sprintf("#%02x%02x%02x", rgba.R, rgba.G, rgba.B)
}

func (r *svgPathRecorder) generateSVGPathElement(pathData, operation, colorStr string) string {
	var attrs []string

	if operation == "fill" {
		// For fill operations, we might have fill-rule
		if r.fillRule != "" {
			attrs = append(attrs, fmt.Sprintf(`fill-rule="%s"`, r.fillRule))
		}
		if colorStr != "" {
			attrs = append(attrs, fmt.Sprintf(`fill="%s"`, colorStr))
		}
	} else if operation == "stroke" {
		// For stroke operations, add stroke attributes
		if colorStr != "" {
			attrs = append(attrs, fmt.Sprintf(`stroke="%s"`, colorStr))
		}
		if r.strokeWidth > 0 {
			attrs = append(attrs, fmt.Sprintf(`stroke-width="%.2f"`, r.strokeWidth))
		}
		if r.strokeLineCap != "" {
			attrs = append(attrs, fmt.Sprintf(`stroke-linecap="%s"`, r.strokeLineCap))
		}
		if len(r.strokeDash) > 0 {
			dashStr := make([]string, len(r.strokeDash))
			for i, d := range r.strokeDash {
				dashStr[i] = fmt.Sprintf("%.2f", d)
			}
			attrs = append(attrs, fmt.Sprintf(`stroke-dasharray="%s"`, strings.Join(dashStr, " ")))
		}
		// For stroked paths, we typically don't want fill
		attrs = append(attrs, `fill="none"`)
	}

	if len(attrs) > 0 {
		return fmt.Sprintf(`<path d="%s" %s/>`, pathData, strings.Join(attrs, " "))
	}
	return fmt.Sprintf(`<path d="%s"/>`, pathData)
}

// NewSVGPathRecorder creates a new svgPathRecorder
func NewSVGPathRecorder(hasGradient bool) *svgPathRecorder {
	return &svgPathRecorder{
		hasGradient: hasGradient,
	}
}

func (r *svgPathRecorder) toSVGPath() string {
	// Finalize any remaining path
	r.Fill()

	if len(r.pathElements) == 0 {
		return ""
	}
	return strings.Join(r.pathElements, "")
}

// writeSVGGradient writes an SVG linear gradient definition using user space units (pixels)
func (s svgEncoder) writeSVGGradient(w io.Writer, gradient *LinearGradient, id string, width, height int) error {
	// Convert angle to SVG gradient coordinates (user space)
	angle := gradient.Angle
	angleRad := angle * math.Pi / 180.0

	// Direction vector
	dx := math.Cos(angleRad)
	dy := -math.Sin(angleRad) // Negative because SVG Y increases downward

	// Calculate gradient coordinates based on actual QR code bounds (like PNG)
	// The QR code is centered in the canvas with borders
	bounds := image.Rect(0, 0, width, height)
	xmin, xmax := float64(bounds.Min.X), float64(bounds.Max.X)
	ymin, ymax := float64(bounds.Min.Y), float64(bounds.Max.Y)

	// Get all 4 corners of the image
	corners := [4][2]float64{
		{xmin, ymin},
		{xmin, ymax},
		{xmax, ymin},
		{xmax, ymax},
	}

	// Compute min and max projection of corners on the gradient axis (same as PNG)
	minProj, maxProj := math.Inf(1), math.Inf(-1)
	for _, p := range corners {
		proj := p[0]*dx + p[1]*dy
		if proj < minProj {
			minProj = proj
		}
		if proj > maxProj {
			maxProj = proj
		}
	}

	// Calculate gradient line endpoints based on projections
	centerX := (xmin + xmax) / 2
	centerY := (ymin + ymax) / 2
	halfRange := (maxProj - minProj) / 2

	x1 := centerX - halfRange*dx
	y1 := centerY - halfRange*dy
	x2 := centerX + halfRange*dx
	y2 := centerY + halfRange*dy

	_, err := fmt.Fprintf(w, `<defs><linearGradient id="%s" gradientUnits="userSpaceOnUse" x1="%.3f" y1="%.3f" x2="%.3f" y2="%.3f">\n`, id, x1, y1, x2, y2)
	if err != nil {
		return err
	}

	// Write color stops
	for _, stop := range gradient.Stops {
		r, g, b, _ := stop.Color.RGBA()
		colorStr := fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		_, err = fmt.Fprintf(w, `  <stop offset="%.3f" stop-color="%s"/>\n`, stop.T, colorStr)
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(w, `</linearGradient></defs>\n`)
	return err
}

// getSVGShape returns the appropriate SVG shape based on the shape option
func getSVGShape(shape IShape) SVGShape {
	if shape == _shapeCircle {
		return svgCircle{}
	}
	if shape == _shapeRectangle {
		return svgRectangle{}
	}
	// For any other shape (including ComposableShape), use the generic path recorder
	return svgPathShape{shape: shape}
}

// SvgoEncoder encodes QR code directly to SVG using svgo library
type SvgoEncoder struct{}

// Encode encodes an image to SVG by embedding it as a base64 PNG (fallback method)
func (s SvgoEncoder) Encode(w io.Writer, img image.Image) error {
	return s.EncodeWithOptions(w, img, nil)
}

// EncodeWithOptions encodes an image to SVG with options by embedding it as a base64 PNG (fallback method)
func (s SvgoEncoder) EncodeWithOptions(w io.Writer, img image.Image, opts *outputImageOptions) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create SVG canvas
	canvas := svgo.New(w)
	canvas.Startview(width, height, 0, 0, width, height)

	// Encode image to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}

	// Embed as base64 data URL
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(buf.Bytes()))
	canvas.Image(0, 0, width, height, dataURL)

	canvas.End()
	return nil
}

// EncodeMatrix encodes QR code matrix directly to SVG using svgo
func (s SvgoEncoder) EncodeMatrix(w io.Writer, mat qrcode.Matrix, opts *outputImageOptions) error {
	rows := mat.Height()
	cols := mat.Width()

	// Calculate SVG dimensions
	borderWidths := opts.borderWidths
	width := cols*opts.qrBlockWidth() + borderWidths[3] + borderWidths[1]
	height := rows*opts.qrBlockWidth() + borderWidths[0] + borderWidths[2]

	// Create SVG canvas
	canvas := svgo.New(w)
	canvas.Startview(width, height, 0, 0, width, height)

	// Set background if specified
	if opts.bgColor.A > 0 {
		canvas.Rect(0, 0, width, height, fmt.Sprintf("fill:#%02x%02x%02x", opts.bgColor.R, opts.bgColor.G, opts.bgColor.B))
	}

	// Define gradients if present
	if opts.qrGradient != nil {
		s.writeSvgoGradient(canvas, opts.qrGradient, "qrGradient", width, height)
	}

	// Create bitmap for neighbor calculation
	bitmap := make([][]bool, rows)
	for i := range bitmap {
		bitmap[i] = make([]bool, cols)
	}
	mat.Iterate(qrcode.IterDirection_ROW, func(x, y int, v qrcode.QRValue) {
		bitmap[y][x] = v.IsSet()
	})

	// Prepare halftone if requested
	hasHalftone := opts.halftoneImg != nil
	var halftoneImg image.Image
	halftoneW := float64(opts.qrBlockWidth()) / 3.0
	if hasHalftone {
		halftoneImg = imgkit.Binaryzation(
			imgkit.Scale(opts.halftoneImg, image.Rect(0, 0, mat.Width()*3, mat.Width()*3), nil),
			60,
		)
	}

	// Draw QR code blocks
	mat.Iterate(qrcode.IterDirection_ROW, func(x int, y int, v qrcode.QRValue) {
		if !v.IsSet() {
			return
		}

		// Skip if overlaps with logo
		blockX := x*opts.qrBlockWidth() + borderWidths[3]
		blockY := y*opts.qrBlockWidth() + borderWidths[0]
		if opts.logo != nil && opts.logoSafeZone {
			logoWidth := opts.logo.Bounds().Dx()
			logoHeight := opts.logo.Bounds().Dy()
			logoX := (width - logoWidth) / 2
			logoY := (height - logoHeight) / 2
			logoSafeZone := opts.qrBlockWidth() * 2

			if float64(blockX) >= float64(logoX)-float64(logoSafeZone) &&
				float64(blockX) < float64(logoX+logoWidth)+float64(logoSafeZone) &&
				float64(blockY) >= float64(logoY)-float64(logoSafeZone) &&
				float64(blockY) < float64(logoY+logoHeight)+float64(logoSafeZone) {
				return
			}
		}

		// Halftone mode for data modules
		if hasHalftone && v.Type() != qrcode.QRType_FINDER {
			for i := 0; i < 3; i++ {
				for j := 0; j < 3; j++ {
					subX := blockX + int(float64(i)*halftoneW)
					subY := blockY + int(float64(j)*halftoneW)

					var fillStr string
					if i == 1 && j == 1 {
						if opts.qrGradient != nil {
							fillStr = "url(#qrGradient)"
						} else {
							blockColor := opts.translateToRGBA(v)
							r, g, b, _ := blockColor.RGBA()
							fillStr = fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
						}
					} else {
						c := halftoneColor(halftoneImg, opts.bgTransparent, x*3+i, y*3+j)
						r, g, b, a := c.RGBA()
						if a == 0 {
							continue
						}
						fillStr = fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
					}

					subCtx := &DrawContext{
						x:     float64(subX),
						y:     float64(subY),
						w:     int(halftoneW),
						h:     int(halftoneW),
						color: color.Black,
						// No neighbors for sub-blocks
					}

					s.drawBlockWithSvgo(canvas, subCtx, opts.getShape(), qrcode.QRType_DATA, fillStr)
				}
			}
			return
		}

		// Get color for normal rendering
		var fillColor string
		if opts.qrGradient != nil {
			fillColor = "url(#qrGradient)"
		} else {
			blockColor := opts.translateToRGBA(v)
			r, g, b, _ := blockColor.RGBA()
			fillColor = fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		}

		// Calculate neighbors for complex shapes
		neighbours := getNeighbours(bitmap, x, y)

		// Create drawing context
		blockSize := opts.qrBlockWidth()
		drawCtx := &DrawContext{
			x:          float64(blockX),
			y:          float64(blockY),
			w:          blockSize,
			h:          blockSize,
			color:      color.Black,
			neighbours: neighbours,
		}

		// Draw the block using svgo
		s.drawBlockWithSvgo(canvas, drawCtx, opts.getShape(), v.Type(), fillColor)
	})

	// Embed logo if present
	if opts.logo != nil {
		s.embedLogoWithSvgo(canvas, opts.logo, width, height)
	}

	canvas.End()
	return nil
}

// writeSvgoGradient writes a gradient definition using svgo
func (s SvgoEncoder) writeSvgoGradient(canvas *svgo.SVG, gradient *LinearGradient, id string, width, height int) {
	// Convert angle to actual coordinates based on QR code dimensions
	angle := gradient.Angle
	angleRad := angle * math.Pi / 180.0
	dx := math.Cos(angleRad)
	dy := -math.Sin(angleRad)

	// Calculate gradient coordinates based on actual QR code bounds (like PNG)
	bounds := image.Rect(0, 0, width, height)
	xmin, xmax := float64(bounds.Min.X), float64(bounds.Max.X)
	ymin, ymax := float64(bounds.Min.Y), float64(bounds.Max.Y)

	// Get all 4 corners of the image
	corners := [4][2]float64{
		{xmin, ymin},
		{xmin, ymax},
		{xmax, ymin},
		{xmax, ymax},
	}

	// Compute min and max projection of corners on the gradient axis (same as PNG)
	minProj, maxProj := math.Inf(1), math.Inf(-1)
	for _, p := range corners {
		proj := p[0]*dx + p[1]*dy
		if proj < minProj {
			minProj = proj
		}
		if proj > maxProj {
			maxProj = proj
		}
	}

	// Calculate gradient line endpoints based on projections
	centerX := (xmin + xmax) / 2
	centerY := (ymin + ymax) / 2
	halfRange := (maxProj - minProj) / 2

	x1 := int(centerX - halfRange*dx)
	y1 := int(centerY - halfRange*dy)
	x2 := int(centerX + halfRange*dx)
	y2 := int(centerY + halfRange*dy)

	// Create color stops
	var stops []svgo.Offcolor
	for _, stop := range gradient.Stops {
		r, g, b, a := stop.Color.RGBA()
		colorStr := fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		opacity := float64(a>>8) / 255.0
		stops = append(stops, svgo.Offcolor{
			Offset:  uint8(stop.T * 100),
			Color:   colorStr,
			Opacity: opacity,
		})
	}

	// Use full integer coordinates to avoid truncation; svgo uses user-space units
	// svgo API expects uint8 for coordinates; clamp to [0,255]
	clamp := func(v int) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	canvas.LinearGradient(id, clamp(x1), clamp(y1), clamp(x2), clamp(y2), stops)
}

// drawBlockWithSvgo draws a QR block using svgo
func (s SvgoEncoder) drawBlockWithSvgo(canvas *svgo.SVG, ctx *DrawContext, shape IShape, qrType qrcode.QRType, fillColor string) {
	x, y := int(ctx.x), int(ctx.y)
	w, h := ctx.w, ctx.h

	// Use appropriate shape based on type
	switch qrType {
	case qrcode.QRType_FINDER:
		if shape != nil {
			// Use custom finder shape
			s.drawShapeWithSvgo(canvas, ctx, shape, true, fillColor)
		} else {
			// Default square finder
			canvas.Rect(x, y, w, h, fmt.Sprintf("fill:%s", fillColor))
		}
	default:
		if shape != nil {
			// Use custom shape
			s.drawShapeWithSvgo(canvas, ctx, shape, false, fillColor)
		} else {
			// Default square block
			canvas.Rect(x, y, w, h, fmt.Sprintf("fill:%s", fillColor))
		}
	}
}

// drawShapeWithSvgo draws a custom shape using svgo by simulating the drawing operations
func (s SvgoEncoder) drawShapeWithSvgo(canvas *svgo.SVG, ctx *DrawContext, shape IShape, isFinder bool, fillColor string) {
	// Create a recorder that collects drawing operations
	recorder := &svgoShapeRecorder{
		canvas:    canvas,
		ctx:       ctx,
		fillColor: fillColor,
	}

	// Create a DrawContext with the recorder
	tempCtx := &DrawContext{
		GraphicsContext: recorder,
		x:               ctx.x,
		y:               ctx.y,
		w:               ctx.w,
		h:               ctx.h,
		color:           ctx.color,
		neighbours:      ctx.neighbours,
	}

	// Call the shape's draw method
	if isFinder {
		shape.DrawFinder(tempCtx)
	} else {
		shape.Draw(tempCtx)
	}
}

// embedLogoWithSvgo embeds a logo image using svgo
func (s SvgoEncoder) embedLogoWithSvgo(canvas *svgo.SVG, logo image.Image, totalWidth, totalHeight int) {
	logoWidth := logo.Bounds().Dx()
	logoHeight := logo.Bounds().Dy()
	logoX := (totalWidth - logoWidth) / 2
	logoY := (totalHeight - logoHeight) / 2

	// Encode logo to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, logo); err != nil {
		return // Skip logo if encoding fails
	}

	// Embed as base64 data URL
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(buf.Bytes()))
	canvas.Image(logoX, logoY, logoWidth, logoHeight, dataURL)
}

// svgoShapeRecorder records drawing operations for svgo
type svgoShapeRecorder struct {
	canvas    *svgo.SVG
	ctx       *DrawContext
	fillColor string

	// Drawing state
	currentPath   []string
	strokeWidth   float64
	strokeColor   string
	strokeDash    []float64
	strokeLineCap string
	fillRule      string
	hasStroke     bool
	hasFill       bool
}

func (r *svgoShapeRecorder) MoveTo(x, y float64) {
	r.currentPath = append(r.currentPath, fmt.Sprintf("M%.2f %.2f", x, y))
}

func (r *svgoShapeRecorder) LineTo(x, y float64) {
	r.currentPath = append(r.currentPath, fmt.Sprintf("L%.2f %.2f", x, y))
}

func (r *svgoShapeRecorder) QuadraticTo(cx, cy, x, y float64) {
	r.currentPath = append(r.currentPath, fmt.Sprintf("Q%.2f %.2f %.2f %.2f", cx, cy, x, y))
}

func (r *svgoShapeRecorder) ClosePath() {
	r.currentPath = append(r.currentPath, "Z")
}

func (r *svgoShapeRecorder) DrawCircle(cx, cy, radius float64) {
	// If using even-odd or winding rules, append to current path to allow compound shapes
	if r.fillRule != "" {
		r.currentPath = append(r.currentPath,
			fmt.Sprintf("M%.2f %.2f A%.2f %.2f 0 1 1 %.2f %.2f A%.2f %.2f 0 1 1 %.2f %.2f Z",
				cx+radius, cy, radius, radius, cx-radius, cy, radius, radius, cx+radius, cy))
		return
	}

	if r.strokeWidth > 0 && len(r.strokeDash) == 0 {
		// Simple stroke without dash - can use circle element
		attrs := fmt.Sprintf("stroke:%s;stroke-width:%.2f;fill:none", r.strokeColor, r.strokeWidth)
		if r.strokeLineCap != "" {
			attrs += fmt.Sprintf(";stroke-linecap:%s", r.strokeLineCap)
		}
		r.canvas.Circle(int(cx), int(cy), int(radius), attrs)
	} else if r.strokeWidth > 0 && len(r.strokeDash) > 0 {
		// Dashed stroke - need to convert to path
		// Create a circular path
		pathData := fmt.Sprintf("M%.2f %.2f A%.2f %.2f 0 1 1 %.2f %.2f A%.2f %.2f 0 1 1 %.2f %.2f",
			cx+radius, cy, radius, radius, cx-radius, cy, radius, radius, cx+radius, cy)
		attrs := fmt.Sprintf("stroke:%s;stroke-width:%.2f;fill:none", r.strokeColor, r.strokeWidth)
		if len(r.strokeDash) > 0 {
			dashStr := make([]string, len(r.strokeDash))
			for i, d := range r.strokeDash {
				dashStr[i] = fmt.Sprintf("%.2f", d)
			}
			attrs += fmt.Sprintf(";stroke-dasharray:%s", strings.Join(dashStr, " "))
		}
		if r.strokeLineCap != "" {
			attrs += fmt.Sprintf(";stroke-linecap:%s", r.strokeLineCap)
		}
		r.canvas.Path(pathData, attrs)
	} else {
		// Fill the circle
		r.canvas.Circle(int(cx), int(cy), int(radius), fmt.Sprintf("fill:%s", r.fillColor))
	}
}

func (r *svgoShapeRecorder) DrawRectangle(x, y, w, h float64) {
	if r.fillRule != "" {
		r.currentPath = append(r.currentPath,
			fmt.Sprintf("M%.2f %.2f L%.2f %.2f L%.2f %.2f L%.2f %.2f Z",
				x, y, x+w, y, x+w, y+h, x, y+h))
		return
	}

	if r.strokeWidth > 0 && len(r.strokeDash) == 0 {
		// Simple stroke without dash - can use rect element
		attrs := fmt.Sprintf("stroke:%s;stroke-width:%.2f;fill:none", r.strokeColor, r.strokeWidth)
		if r.strokeLineCap != "" {
			attrs += fmt.Sprintf(";stroke-linecap:%s", r.strokeLineCap)
		}
		r.canvas.Rect(int(x), int(y), int(w), int(h), attrs)
	} else if r.strokeWidth > 0 && len(r.strokeDash) > 0 {
		// Dashed stroke - need to convert to path
		pathData := fmt.Sprintf("M%.2f %.2f L%.2f %.2f L%.2f %.2f L%.2f %.2f Z",
			x, y, x+w, y, x+w, y+h, x, y+h)
		attrs := fmt.Sprintf("stroke:%s;stroke-width:%.2f;fill:none", r.strokeColor, r.strokeWidth)
		if len(r.strokeDash) > 0 {
			dashStr := make([]string, len(r.strokeDash))
			for i, d := range r.strokeDash {
				dashStr[i] = fmt.Sprintf("%.2f", d)
			}
			attrs += fmt.Sprintf(";stroke-dasharray:%s", strings.Join(dashStr, " "))
		}
		if r.strokeLineCap != "" {
			attrs += fmt.Sprintf(";stroke-linecap:%s", r.strokeLineCap)
		}
		r.canvas.Path(pathData, attrs)
	} else {
		// Fill the rectangle
		r.canvas.Rect(int(x), int(y), int(w), int(h), fmt.Sprintf("fill:%s", r.fillColor))
	}
}

func (r *svgoShapeRecorder) SetColor(c color.Color) {
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	r.strokeColor = fmt.Sprintf("#%02x%02x%02x", rgba.R, rgba.G, rgba.B)
}

func (r *svgoShapeRecorder) Fill() {
	if len(r.currentPath) > 0 {
		pathData := strings.Join(r.currentPath, " ")
		attrs := fmt.Sprintf("fill:%s", r.fillColor)
		if r.fillRule != "" {
			attrs += fmt.Sprintf(";fill-rule:%s", r.fillRule)
		}
		r.canvas.Path(pathData, attrs)
		r.currentPath = nil
	}
	r.hasFill = true
}

func (r *svgoShapeRecorder) Stroke() {
	if len(r.currentPath) > 0 {
		pathData := strings.Join(r.currentPath, " ")
		attrs := fmt.Sprintf("stroke:%s;stroke-width:%.2f;fill:none", r.strokeColor, r.strokeWidth)
		if len(r.strokeDash) > 0 {
			dashStr := make([]string, len(r.strokeDash))
			for i, d := range r.strokeDash {
				dashStr[i] = fmt.Sprintf("%.2f", d)
			}
			attrs += fmt.Sprintf(";stroke-dasharray:%s", strings.Join(dashStr, " "))
		}
		if r.strokeLineCap != "" {
			attrs += fmt.Sprintf(";stroke-linecap:%s", r.strokeLineCap)
		}
		r.canvas.Path(pathData, attrs)
		r.currentPath = nil
	}
	r.hasStroke = true
}

func (r *svgoShapeRecorder) SetDash(dashes ...float64) {
	r.strokeDash = dashes
}

func (r *svgoShapeRecorder) SetLineWidth(lineWidth float64) {
	r.strokeWidth = lineWidth
}

func (r *svgoShapeRecorder) SetLineCap(lineCap gg.LineCap) {
	// Line cap is handled in SVG path attributes
}

func (r *svgoShapeRecorder) SetLineCapSquare() {
	r.strokeLineCap = "square"
}

func (r *svgoShapeRecorder) SetFillRuleEvenOdd() {
	r.fillRule = "evenodd"
}

func (r *svgoShapeRecorder) SetFillRuleWinding() {
	r.fillRule = "nonzero"
}

func (r *svgoShapeRecorder) NewSubPath() {
	// For complex shapes, we might need to handle subpaths differently
	// For now, this is a simplified implementation
}

// svgEncoder encodes QR code to SVG format (legacy implementation)
// It implements ImageEncoder, ImageEncoderWithOptions, and ImageEncoderWithMatrix interfaces
type svgEncoder struct{}

func (s svgEncoder) Encode(w io.Writer, img image.Image) error {
	return s.EncodeWithOptions(w, img, nil)
}

func (s svgEncoder) EncodeWithOptions(w io.Writer, img image.Image, opts *outputImageOptions) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Check if logo is present and valid (same logic as PNG)
	hasLogo := opts != nil && opts.logo != nil
	var logoValid bool
	var logoWidth, logoHeight int
	if hasLogo {
		bound := opts.logo.Bounds()
		logoWidth = bound.Max.X - bound.Min.X
		logoHeight = bound.Max.Y - bound.Min.Y
		logoValid = validLogoImage(width, height, logoWidth, logoHeight, opts.logoSizeMultiplier)
	}

	// SVG header
	_, err := fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
`, width, height)
	if err != nil {
		return err
	}

	// Background rectangle (white)
	_, err = fmt.Fprintf(w, `<rect width="%d" height="%d" fill="white"/>\n`, width, height)
	if err != nil {
		return err
	}

	// Draw pixels as optimized rectangles using comprehensive rectangle merging
	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

	type rectInfo struct {
		x, y, width, height int
		r, g, b, a          uint8
	}

	var allRects []rectInfo

	// First pass: create initial rectangles (horizontal runs)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		x := bounds.Min.X
		for x < bounds.Max.X {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()
			r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)

			// Skip transparent or white pixels
			if a8 == 0 || (r8 == 255 && g8 == 255 && b8 == 255) {
				x++
				continue
			}

			// Find horizontal run
			startX := x
			for x < bounds.Max.X {
				nextC := img.At(x, y)
				nr, ng, nb, na := nextC.RGBA()
				nr8, ng8, nb8, na8 := uint8(nr>>8), uint8(ng>>8), uint8(nb>>8), uint8(na>>8)
				if na8 != a8 || nr8 != r8 || ng8 != g8 || nb8 != b8 {
					break
				}
				x++
			}
			width := x - startX

			allRects = append(allRects, rectInfo{
				x: startX, y: y, width: width, height: 1,
				r: r8, g: g8, b: b8, a: a8,
			})
		}
	}

	// Second pass: merge touching rectangles of the same color
	merged := true
	for merged {
		merged = false
		var newRects []rectInfo

		used := make([]bool, len(allRects))
		for i := 0; i < len(allRects); i++ {
			if used[i] {
				continue
			}

			rect1 := allRects[i]
			mergedThis := false

			// Look for rectangles that can be merged with rect1
			for j := i + 1; j < len(allRects); j++ {
				if used[j] {
					continue
				}

				rect2 := allRects[j]

				// Only merge if same color
				if rect1.r != rect2.r || rect1.g != rect2.g || rect1.b != rect2.b || rect1.a != rect2.a {
					continue
				}

				// Check if rectangles touch (adjacent edges)

				// Horizontal adjacency (same y, touching x edges)
				if rect1.y == rect2.y && rect1.height == rect2.height {
					if rect1.x+rect1.width == rect2.x || rect2.x+rect2.width == rect1.x {
						// Merge horizontally
						newRect := rectInfo{
							x:      min(rect1.x, rect2.x),
							y:      rect1.y,
							width:  rect1.width + rect2.width,
							height: rect1.height,
							r:      rect1.r, g: rect1.g, b: rect1.b, a: rect1.a,
						}
						newRects = append(newRects, newRect)
						used[i] = true
						used[j] = true
						merged = true
						mergedThis = true
						break
					}
				}

				// Vertical adjacency (same x, touching y edges)
				if rect1.x == rect2.x && rect1.width == rect2.width {
					if rect1.y+rect1.height == rect2.y || rect2.y+rect2.height == rect1.y {
						// Merge vertically
						newRect := rectInfo{
							x:      rect1.x,
							y:      min(rect1.y, rect2.y),
							width:  rect1.width,
							height: rect1.height + rect2.height,
							r:      rect1.r, g: rect1.g, b: rect1.b, a: rect1.a,
						}
						newRects = append(newRects, newRect)
						used[i] = true
						used[j] = true
						merged = true
						mergedThis = true
						break
					}
				}
			}

			if !mergedThis {
				newRects = append(newRects, rect1)
			}
		}

		allRects = newRects
	}

	// Output all merged rectangles
	for _, rect := range allRects {
		if rect.a == 0 || (rect.r == 255 && rect.g == 255 && rect.b == 255) {
			continue // Skip transparent or white rectangles
		}

		if rect.r == 0 && rect.g == 0 && rect.b == 0 {
			// Black rectangle
			_, err = fmt.Fprintf(w, `<rect x="%d" y="%d" width="%d" height="%d" fill="black"/>\n`,
				rect.x, rect.y, rect.width, rect.height)
		} else {
			// Colored rectangle
			_, err = fmt.Fprintf(w, `<rect x="%d" y="%d" width="%d" height="%d" fill="rgb(%d,%d,%d)"/>\n`,
				rect.x, rect.y, rect.width, rect.height, rect.r, rect.g, rect.b)
		}
		if err != nil {
			return err
		}
	}

	// Embed logo if present and valid (same as PNG)
	if logoValid {
		// Encode logo to PNG
		var buf bytes.Buffer
		if err := png.Encode(&buf, opts.logo); err != nil {
			// Skip logo if encoding fails
		} else {
			// Embed as base64 data URL
			dataURL := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(buf.Bytes()))
			logoX := (width - logoWidth) / 2
			logoY := (height - logoHeight) / 2
			_, err = fmt.Fprintf(w, `<image x="%d" y="%d" width="%d" height="%d" href="%s"/>\n`,
				logoX, logoY, logoWidth, logoHeight, dataURL)
			if err != nil {
				return err
			}
		}
	}

	// SVG footer
	_, err = fmt.Fprintf(w, `</svg>`)
	return err
}

func (s svgEncoder) EncodeMatrix(w io.Writer, mat qrcode.Matrix, opts *outputImageOptions) error {
	if opts == nil {
		opts = defaultOutputImageOption()
	}

	width := mat.Width()*opts.qrBlockWidth() + opts.borderWidths[0] + opts.borderWidths[1]
	height := mat.Height()*opts.qrBlockWidth() + opts.borderWidths[2] + opts.borderWidths[3]

	// Check if logo is present and valid (same logic as PNG)
	hasLogo := opts.logo != nil
	var logoValid bool
	var logoWidth, logoHeight int
	if hasLogo {
		bound := opts.logo.Bounds()
		logoWidth = bound.Max.X - bound.Min.X
		logoHeight = bound.Max.Y - bound.Min.Y
		logoValid = validLogoImage(width, height, logoWidth, logoHeight, opts.logoSizeMultiplier)
	}

	// Get the SVG shape generator
	svgShape := getSVGShape(opts.getShape())

	// SVG header
	_, err := fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
`, width, height)
	if err != nil {
		return err
	}

	// Define gradients if present
	if opts.qrGradient != nil {
		if err := s.writeSVGGradient(w, opts.qrGradient, "qrGradient", width, height); err != nil {
			return err
		}
	}

	// Background rectangle
	backgroundColor := opts.backgroundColor()
	r, g, b, a := backgroundColor.RGBA()
	if a != 0 { // not transparent
		hexColor := fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		_, err = fmt.Fprintf(w, `<rect width="%d" height="%d" fill="%s"/>\n`, width, height, hexColor)
		if err != nil {
			return err
		}
	}

	// Group for QR code blocks
	_, err = fmt.Fprintf(w, `<g>\n`)
	if err != nil {
		return err
	}

	// Logo validation is already done above

	// Create a bitmap for neighbor calculation
	rows := mat.Height()
	cols := mat.Width()
	bitmap := make([][]bool, rows)
	for i := range bitmap {
		bitmap[i] = make([]bool, cols)
	}
	mat.Iterate(qrcode.IterDirection_ROW, func(x, y int, v qrcode.QRValue) {
		bitmap[y][x] = v.IsSet()
	})

	// If logo safe zone is enabled, clear the corresponding area in bitmap (same as PNG)
	if logoValid && opts.logoSafeZone {
		mat.Iterate(qrcode.IterDirection_ROW, func(x int, y int, v qrcode.QRValue) {
			if blockOverlapsLogo(x, y, opts.qrBlockWidth(), opts.borderWidths[3], opts.borderWidths[0], width, height, logoWidth, logoHeight) {
				bitmap[y][x] = false
			}
		})
	}

	// Check if halftone is enabled
	hasHalftone := opts.halftoneImg != nil
	var halftoneImg image.Image
	halftoneW := float64(opts.qrBlockWidth()) / 3.0

	if hasHalftone {
		halftoneImg = imgkit.Binaryzation(
			imgkit.Scale(opts.halftoneImg, image.Rect(0, 0, mat.Width()*3, mat.Width()*3), nil),
			60,
		)
	}

	// Iterate through the matrix and generate SVG paths
	mat.Iterate(qrcode.IterDirection_ROW, func(x int, y int, v qrcode.QRValue) {
		// Skip if not set
		if !v.IsSet() {
			return
		}

		// Skip drawing this block if it overlaps with the logo area and safe zone is enabled (same as PNG)
		if logoValid && opts.logoSafeZone &&
			blockOverlapsLogo(x, y, opts.qrBlockWidth(), opts.borderWidths[3], opts.borderWidths[0],
				width, height, logoWidth, logoHeight) {
			return
		}

		// Calculate position
		blockX := x*opts.qrBlockWidth() + opts.borderWidths[3]
		blockY := y*opts.qrBlockWidth() + opts.borderWidths[0]

		// Handle halftone: create 3x3 grid like PNG version
		if hasHalftone && v.Type() == qrcode.QRType_DATA {
			// Use the exact same approach as PNG: create 3x3 sub-blocks
			// Each sub-block gets its color from the halftone image
			halftoneImgWidth := mat.Width() * 3
			halftoneImgHeight := mat.Width() * 3

			// Calculate the region in the halftone image that corresponds to this QR block
			startX := x * 3
			startY := y * 3

			// Create 3x3 grid of sub-blocks (like PNG does)
			for i := 0; i < 3; i++ {
				for j := 0; j < 3; j++ {
					halftoneX := startX + i
					halftoneY := startY + j

					// Get color for this sub-block
					var fillStr string
					if opts.qrGradient != nil {
						fillStr = "url(#qrGradient)"
					} else {
						// Center sub-block always uses QR color
						if i == 1 && j == 1 {
							// Center sub-block uses QR block color
							blockColor := opts.translateToRGBA(v)
							r, g, b, _ := blockColor.RGBA()
							fillStr = fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
						} else {
							// Edge sub-blocks use halftone pattern - only draw if black pixel
							if halftoneX < halftoneImgWidth && halftoneY < halftoneImgHeight {
								halftonePixel := halftoneImg.At(halftoneX, halftoneY)
								if gray, ok := halftonePixel.(color.Gray); ok && gray.Y == 0 {
									// Black pixel - use QR color
									blockColor := opts.translateToRGBA(v)
									r, g, b, _ := blockColor.RGBA()
									fillStr = fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
								} else {
									// White pixel - skip drawing (transparent)
									continue
								}
							} else {
								// Out of bounds - skip drawing
								continue
							}
						}
					}

					// Calculate position for this sub-block
					subX := blockX + int(float64(i)*halftoneW)
					subY := blockY + int(float64(j)*halftoneW)

					// Create a rectangle that fills most of the sub-block (like PNG)
					rectSize := halftoneW * 0.9 // 90% of sub-block size
					cx := float64(subX) + (halftoneW-rectSize)/2.0
					cy := float64(subY) + (halftoneW-rectSize)/2.0

					// Write the rectangle
					fmt.Fprintf(w, `<g fill="%s"><rect x="%.2f" y="%.2f" width="%.2f" height="%.2f"/></g>`,
						fillStr, cx, cy, rectSize, rectSize)
				}
			}
			return
		}

		// Normal block rendering (no halftone)
		// Get color/gradient for this block
		var fillStr string
		if opts.qrGradient != nil {
			fillStr = "url(#qrGradient)"
		} else {
			blockColor := opts.translateToRGBA(v)
			r, g, b, _ := blockColor.RGBA()
			fillStr = fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		}

		// Calculate neighbors for complex shapes
		neighbours := getNeighbours(bitmap, x, y)

		// Create DrawContext for SVG shape generation
		drawCtx := &DrawContext{
			x:          float64(blockX),
			y:          float64(blockY),
			w:          opts.qrBlockWidth(),
			h:          opts.qrBlockWidth(),
			color:      opts.translateToRGBA(v), // Use actual QR code color
			neighbours: neighbours,
		}

		// Generate SVG path based on block type
		var pathData string
		var isComplexShape bool
		switch v.Type() {
		case qrcode.QRType_FINDER:
			pathData = svgShape.GenerateSVGFinder(drawCtx, opts.qrGradient != nil)
		default:
			pathData = svgShape.GenerateSVGPath(drawCtx, opts.qrGradient != nil)
		}
		// Check if this is a complex shape that handles its own colors (for all block types)
		isComplexShape = strings.Contains(pathData, `stroke="`) || strings.Contains(pathData, `fill="`)

		// Write the path element
		if isComplexShape {
			// Complex shapes handle their own colors, but we still need to apply gradients
			if opts.qrGradient != nil {
				// Apply gradient to complex shapes by wrapping with gradient fill
				fmt.Fprintf(w, `<g fill="url(#qrGradient)">%s</g>\n`, pathData)
			} else {
				// No gradient, just wrap in a group
				fmt.Fprintf(w, `<g>%s</g>\n`, pathData)
			}
		} else {
			// Simple shapes use the group fill
			fmt.Fprintf(w, `<g fill="%s">%s</g>\n`, fillStr, pathData)
		}
	})

	// Close the QR code group
	_, err = fmt.Fprintf(w, `</g>\n`)
	if err != nil {
		return err
	}

	// Embed logo if present and valid (same as PNG)
	if logoValid {
		// Encode logo to PNG
		var buf bytes.Buffer
		if err := png.Encode(&buf, opts.logo); err != nil {
			// Skip logo if encoding fails
		} else {
			// Embed as base64 data URL
			dataURL := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(buf.Bytes()))
			logoX := (width - logoWidth) / 2
			logoY := (height - logoHeight) / 2
			_, err = fmt.Fprintf(w, `<image x="%d" y="%d" width="%d" height="%d" href="%s"/>\n`,
				logoX, logoY, logoWidth, logoHeight, dataURL)
			if err != nil {
				return err
			}
		}
	}

	// Close SVG
	_, err = fmt.Fprintf(w, `</svg>`)
	return err
}
