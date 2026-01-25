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
	"sort"
	"strings"

	"github.com/fogleman/gg"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard/imgkit"
)

type formatTyp uint8

const (
	JPEG_FORMAT formatTyp = iota
	PNG_FORMAT
	SVG_FORMAT
)

// ImageEncoder is an interface which describes the rule how to encode image.Image into io.Writer
type ImageEncoder interface {
	Encode(w io.Writer, img image.Image) error
}

// ImageEncoderWithOptions is an extended interface for encoders that can accept additional options
type ImageEncoderWithOptions interface {
	ImageEncoder
	EncodeWithOptions(w io.Writer, img image.Image, opts *outputImageOptions) error
}

// ImageEncoderWithMatrix is an interface for encoders that can work directly with QR matrix
type ImageEncoderWithMatrix interface {
	ImageEncoder
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
	GenerateSVGPath(ctx *DrawContext, hasGradient bool) string
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
	recorder := NewSVGPathRecorder(hasGradient)
	tempCtx := &DrawContext{
		GraphicsContext: recorder,
		x:               ctx.x,
		y:               ctx.y,
		w:               ctx.w,
		h:               ctx.h,
		color:           ctx.color,
		neighbours:      ctx.neighbours,
	}
	s.shape.Draw(tempCtx)
	return recorder.toSVGPath()
}

func (s svgPathShape) GenerateSVGFinder(ctx *DrawContext, hasGradient bool) string {
	recorder := NewSVGPathRecorder(hasGradient)
	tempCtx := &DrawContext{
		GraphicsContext: recorder,
		x:               ctx.x,
		y:               ctx.y,
		w:               ctx.w,
		h:               ctx.h,
		color:           ctx.color,
		neighbours:      ctx.neighbours,
	}
	s.shape.DrawFinder(tempCtx)
	return recorder.toSVGPath()
}

// svgPathRecorder records drawing operations and converts them to SVG path commands
type svgPathRecorder struct {
	pathElements       []string
	currentCommands    []string
	currentX, currentY float64
	strokeDash         []float64
	strokeWidth        float64
	strokeLineCap      string
	fillRule           string
	currentColor       color.Color
	hasGradient        bool
}

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
	r.currentCommands = append(r.currentCommands,
		fmt.Sprintf("M%.2f %.2f A%.2f %.2f 0 1 1 %.2f %.2f A%.2f %.2f 0 1 1 %.2f %.2f Z",
			cx+radius, cy, radius, radius, cx-radius, cy, radius, radius, cx+radius, cy))
	r.currentX, r.currentY = cx+radius, cy
}

func (r *svgPathRecorder) DrawRectangle(x, y, w, h float64) {
	r.currentCommands = append(r.currentCommands,
		fmt.Sprintf("M%.2f %.2f L%.2f %.2f L%.2f %.2f L%.2f %.2f Z",
			x, y, x+w, y, x+w, y+h, x, y+h))
	r.currentX, r.currentY = x, y+h
}

func (r *svgPathRecorder) SetColor(c color.Color) {
	r.currentColor = c
}

func (r *svgPathRecorder) Fill() {
	if len(r.currentCommands) > 0 {
		pathData := strings.Join(r.currentCommands, " ")
		var colorStr string
		if !r.hasGradient {
			colorStr = colorToHex(r.currentColor)
		}
		r.pathElements = append(r.pathElements, r.generateSVGPathElement(pathData, "fill", colorStr))
		r.currentCommands = nil
	}
	r.strokeDash = nil
	r.strokeWidth = 0
	r.strokeLineCap = ""
}

func (r *svgPathRecorder) Stroke() {
	if len(r.currentCommands) > 0 {
		pathData := strings.Join(r.currentCommands, " ")
		colorStr := colorToHex(r.currentColor)
		r.pathElements = append(r.pathElements, r.generateSVGPathElement(pathData, "stroke", colorStr))
		r.currentCommands = nil
	}
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
	case 0:
		r.strokeLineCap = "butt"
	case 1:
		r.strokeLineCap = "round"
	case 2:
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
	// For SVG, we continue the path
}

func (r *svgPathRecorder) generateSVGPathElement(pathData, operation, colorStr string) string {
	var attrs []string

	if operation == "fill" {
		if r.fillRule != "" {
			attrs = append(attrs, fmt.Sprintf(`fill-rule="%s"`, r.fillRule))
		}
		if colorStr != "" {
			attrs = append(attrs, fmt.Sprintf(`fill="%s"`, colorStr))
		}
	} else if operation == "stroke" {
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
		attrs = append(attrs, `fill="none"`)
	}

	if len(attrs) > 0 {
		return fmt.Sprintf(`<path d="%s" %s/>`, pathData, strings.Join(attrs, " "))
	}
	return fmt.Sprintf(`<path d="%s"/>`, pathData)
}

func (r *svgPathRecorder) toSVGPath() string {
	r.Fill()
	if len(r.pathElements) == 0 {
		return ""
	}
	return strings.Join(r.pathElements, "")
}

func NewSVGPathRecorder(hasGradient bool) *svgPathRecorder {
	return &svgPathRecorder{
		hasGradient: hasGradient,
	}
}

// Shared utility functions
func colorToHex(c color.Color) string {
	if c == nil {
		return ""
	}
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	return fmt.Sprintf("#%02x%02x%02x", rgba.R, rgba.G, rgba.B)
}

func embedLogoAsPNG(w io.Writer, logo image.Image, width, height, logoWidth, logoHeight int) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, logo); err != nil {
		return err
	}
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(buf.Bytes()))
	logoX := (width - logoWidth) / 2
	logoY := (height - logoHeight) / 2
	_, err := fmt.Fprintf(w, `<image x="%d" y="%d" width="%d" height="%d" href="%s"/>\n`,
		logoX, logoY, logoWidth, logoHeight, dataURL)
	return err
}

func writeSVGGradient(w io.Writer, gradient *LinearGradient, id string, width, height int) error {
	angle := gradient.Angle
	angleRad := angle * math.Pi / 180.0
	dx := math.Cos(angleRad)
	dy := -math.Sin(angleRad)

	bounds := image.Rect(0, 0, width, height)
	xmin, xmax := float64(bounds.Min.X), float64(bounds.Max.X)
	ymin, ymax := float64(bounds.Min.Y), float64(bounds.Max.Y)

	corners := [4][2]float64{
		{xmin, ymin},
		{xmin, ymax},
		{xmax, ymin},
		{xmax, ymax},
	}

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

func getSVGShape(shape IShape) SVGShape {
	if shape == _shapeCircle {
		return svgCircle{}
	}
	if shape == _shapeRectangle {
		return svgRectangle{}
	}
	return svgPathShape{shape: shape}
}

// svgEncoder encodes QR code to SVG format
type svgEncoder struct{}

func (s svgEncoder) Encode(w io.Writer, img image.Image) error {
	return s.EncodeWithOptions(w, img, nil)
}

func (s svgEncoder) EncodeWithOptions(w io.Writer, img image.Image, opts *outputImageOptions) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	hasLogo := opts != nil && opts.logo != nil
	var logoValid bool
	var logoWidth, logoHeight int
	if hasLogo {
		bound := opts.logo.Bounds()
		logoWidth = bound.Max.X - bound.Min.X
		logoHeight = bound.Max.Y - bound.Min.Y
		logoValid = validLogoImage(width, height, logoWidth, logoHeight, opts.logoSizeMultiplier)
	}

	_, err := fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<svg width="%d" height="%d" shape-rendering="crispEdges" xmlns="http://www.w3.org/2000/svg">
`, width, height)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, `<rect width="%d" height="%d" fill="white"/>\n`, width, height)
	if err != nil {
		return err
	}

	// Optimize rectangles
	rects := optimizeRectangles(img, bounds)
	for _, rect := range rects {
		if rect.a == 0 || (rect.r == 255 && rect.g == 255 && rect.b == 255) {
			continue
		}

		if rect.r == 0 && rect.g == 0 && rect.b == 0 {
			_, err = fmt.Fprintf(w, `<rect x="%d" y="%d" width="%d" height="%d" fill="black"/>\n`,
				rect.x, rect.y, rect.width, rect.height)
		} else {
			_, err = fmt.Fprintf(w, `<rect x="%d" y="%d" width="%d" height="%d" fill="rgb(%d,%d,%d)"/>\n`,
				rect.x, rect.y, rect.width, rect.height, rect.r, rect.g, rect.b)
		}
		if err != nil {
			return err
		}
	}

	if logoValid {
		if err := embedLogoAsPNG(w, opts.logo, width, height, logoWidth, logoHeight); err != nil {
			// Skip logo if encoding fails
		}
	}

	_, err = fmt.Fprintf(w, `</svg>`)
	return err
}

type rectInfo struct {
	x, y, width, height int
	r, g, b, a          uint8
}

func optimizeRectangles(img image.Image, bounds image.Rectangle) []rectInfo {
	width := bounds.Dx()
	height := bounds.Dy()
	visited := make([]bool, width*height)
	var rects []rectInfo

	// Helper to get linear index
	getIndex := func(x, y int) int {
		return (y-bounds.Min.Y)*width + (x - bounds.Min.X)
	}

	// PHASE 1: Scan for 3x3 Blocks (Initial Discovery)
	for y := bounds.Min.Y; y <= bounds.Max.Y-3; y += 3 {
		for x := bounds.Min.X; x <= bounds.Max.X-3; x += 3 {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()
			r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)

			if a8 == 0 || (r8 == 255 && g8 == 255 && b8 == 255) {
				continue
			}

			isUniform := true
			// Check the 3x3 grid
			for dy := 0; dy < 3; dy++ {
				for dx := 0; dx < 3; dx++ {
					if visited[getIndex(x+dx, y+dy)] {
						isUniform = false
						break
					}
					nc := img.At(x+dx, y+dy)
					nr, ng, nb, na := nc.RGBA()
					if uint8(nr>>8) != r8 || uint8(ng>>8) != g8 || uint8(nb>>8) != b8 || uint8(na>>8) != a8 {
						isUniform = false
						break
					}
				}
				if !isUniform {
					break
				}
			}

			if isUniform {
				rects = append(rects, rectInfo{
					x: x, y: y, width: 3, height: 3,
					r: r8, g: g8, b: b8, a: a8,
				})
				for dy := 0; dy < 3; dy++ {
					for dx := 0; dx < 3; dx++ {
						visited[getIndex(x+dx, y+dy)] = true
					}
				}
			}
		}
	}

	// PHASE 2: Clean up remaining pixels (Standard RLE)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		x := bounds.Min.X
		for x < bounds.Max.X {
			idx := getIndex(x, y)
			if visited[idx] {
				x++
				continue
			}

			c := img.At(x, y)
			r, g, b, a := c.RGBA()
			r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)

			if a8 == 0 || (r8 == 255 && g8 == 255 && b8 == 255) {
				x++
				continue
			}

			startX := x
			x++
			for x < bounds.Max.X {
				if visited[getIndex(x, y)] {
					break
				}
				nextC := img.At(x, y)
				nr, ng, nb, na := nextC.RGBA()
				nr8, ng8, nb8, na8 := uint8(nr>>8), uint8(ng>>8), uint8(nb>>8), uint8(na>>8)
				if na8 != a8 || nr8 != r8 || ng8 != g8 || nb8 != b8 {
					break
				}
				x++
			}

			rects = append(rects, rectInfo{
				x: startX, y: y, width: x - startX, height: 1,
				r: r8, g: g8, b: b8, a: a8,
			})
		}
	}

	if len(rects) == 0 {
		return rects
	}

	// PHASE 3: Horizontal Merge (Sort + Linear Scan)
	// Sort by Y first, then X
	sort.Slice(rects, func(i, j int) bool {
		if rects[i].y != rects[j].y {
			return rects[i].y < rects[j].y
		}
		return rects[i].x < rects[j].x
	})

	var mergedHorz []rectInfo
	curr := rects[0]

	for i := 1; i < len(rects); i++ {
		next := rects[i]
		// Try to merge horizontally
		if curr.y == next.y &&
			curr.height == next.height &&
			curr.x+curr.width == next.x &&
			curr.r == next.r && curr.g == next.g && curr.b == next.b && curr.a == next.a {
			// Merge
			curr.width += next.width
		} else {
			// Push current and start new
			mergedHorz = append(mergedHorz, curr)
			curr = next
		}
	}
	mergedHorz = append(mergedHorz, curr)

	// PHASE 4: Vertical Merge (Sort + Linear Scan)
	// Sort by X first, then Y
	sort.Slice(mergedHorz, func(i, j int) bool {
		if mergedHorz[i].x != mergedHorz[j].x {
			return mergedHorz[i].x < mergedHorz[j].x
		}
		return mergedHorz[i].y < mergedHorz[j].y
	})

	var finalRects []rectInfo
	if len(mergedHorz) > 0 {
		curr = mergedHorz[0]
		for i := 1; i < len(mergedHorz); i++ {
			next := mergedHorz[i]
			// Try to merge vertically
			if curr.x == next.x &&
				curr.width == next.width &&
				curr.y+curr.height == next.y &&
				curr.r == next.r && curr.g == next.g && curr.b == next.b && curr.a == next.a {
				// Merge
				curr.height += next.height
			} else {
				// Push current and start new
				finalRects = append(finalRects, curr)
				curr = next
			}
		}
		finalRects = append(finalRects, curr)
	}

	return finalRects
}

func (s svgEncoder) EncodeMatrix(w io.Writer, mat qrcode.Matrix, opts *outputImageOptions) error {
	if opts == nil {
		opts = defaultOutputImageOption()
	}

	width := mat.Width()*opts.qrBlockWidth() + opts.borderWidths[0] + opts.borderWidths[1]
	height := mat.Height()*opts.qrBlockWidth() + opts.borderWidths[2] + opts.borderWidths[3]

	hasLogo := opts.logo != nil
	var logoValid bool
	var logoWidth, logoHeight int
	if hasLogo {
		bound := opts.logo.Bounds()
		logoWidth = bound.Max.X - bound.Min.X
		logoHeight = bound.Max.Y - bound.Min.Y
		logoValid = validLogoImage(width, height, logoWidth, logoHeight, opts.logoSizeMultiplier)
	}

	svgShape := getSVGShape(opts.getShape())

	_, err := fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<svg width="%d" height="%d" shape-rendering="crispEdges" xmlns="http://www.w3.org/2000/svg">
`, width, height)
	if err != nil {
		return err
	}

	if opts.qrGradient != nil {
		if err := writeSVGGradient(w, opts.qrGradient, "qrGradient", width, height); err != nil {
			return err
		}
	}

	backgroundColor := opts.backgroundColor()
	r, g, b, a := backgroundColor.RGBA()
	if a != 0 {
		hexColor := fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		_, err = fmt.Fprintf(w, `<rect width="%d" height="%d" fill="%s"/>\n`, width, height, hexColor)
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(w, `<g>\n`)
	if err != nil {
		return err
	}

	rows := mat.Height()
	cols := mat.Width()
	bitmap := make([][]bool, rows)
	for i := range bitmap {
		bitmap[i] = make([]bool, cols)
	}
	mat.Iterate(qrcode.IterDirection_ROW, func(x, y int, v qrcode.QRValue) {
		bitmap[y][x] = v.IsSet()
	})

	if logoValid && opts.logoSafeZone {
		mat.Iterate(qrcode.IterDirection_ROW, func(x int, y int, v qrcode.QRValue) {
			if blockOverlapsLogo(x, y, opts.qrBlockWidth(), opts.borderWidths[3], opts.borderWidths[0], width, height, logoWidth, logoHeight) {
				bitmap[y][x] = false
			}
		})
	}

	hasHalftone := opts.halftoneImg != nil
	var halftoneImg image.Image
	halftoneW := float64(opts.qrBlockWidth()) / 3.0

	if hasHalftone {
		halftoneImg = imgkit.Binaryzation(
			imgkit.Scale(opts.halftoneImg, image.Rect(0, 0, mat.Width()*3, mat.Height()*3), nil),
			60,
			false,
		)
	}

	mat.Iterate(qrcode.IterDirection_ROW, func(x int, y int, v qrcode.QRValue) {
		

		if logoValid && opts.logoSafeZone &&
			blockOverlapsLogo(x, y, opts.qrBlockWidth(), opts.borderWidths[3], opts.borderWidths[0],
				width, height, logoWidth, logoHeight) {
			return
		}

		blockX := x*opts.qrBlockWidth() + opts.borderWidths[3]
		blockY := y*opts.qrBlockWidth() + opts.borderWidths[0]
		neighbours := getNeighbours(bitmap, x, y)
		drawCtx := &DrawContext{
			x:          float64(blockX),
			y:          float64(blockY),
			w:          opts.qrBlockWidth(),
			h:          opts.qrBlockWidth(),
			color:      opts.translateToRGBA(v),
			neighbours: neighbours,
		}
		// Handle halftone for data modules
		if hasHalftone && v.Type() == qrcode.QRType_DATA {
			for i := 0; i < 3; i++ {
				for j := 0; j < 3; j++ {
					subX := float64(blockX) + float64(i)*halftoneW
					subY := float64(blockY) + float64(j)*halftoneW
						
					var subColor color.Color
					if i == 1 && j == 1 {
						subColor = drawCtx.color
					} else {
						subColor = halftoneColor(halftoneImg, opts.bgTransparent, x*3+i, y*3+j)
					}
					
					// Get the fill string for this sub-block
					r, g, b, a := subColor.RGBA()
					r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)
					
					// Skip fully transparent pixels
					if a8 == 0 || (r8 == 255 && g8 == 255 && b8 == 255) {
						continue
					}
					
					subFillStr := fmt.Sprintf("#%02x%02x%02x", r8, g8, b8)
					
					// Create a DrawContext for this sub-block
					ctx2 := &DrawContext{
						GraphicsContext: drawCtx.GraphicsContext,
						x:               subX,
						y:               subY,
						w:               int(halftoneW),
						h:               int(halftoneW),
						color:           subColor,
						neighbours:      drawCtx.neighbours,
					}
					
					// Generate the SVG path for this sub-block using the shape
					pathData := svgShape.GenerateSVGPath(ctx2, false)
					
					// Write the sub-block
					fmt.Fprintf(w, `<g fill="%s">%s</g>\n`, subFillStr, pathData)
				}
			}
			return
		}

		if !v.IsSet() {
			return
		}

		// Normal block rendering
		var fillStr string
		if opts.qrGradient != nil {
			fillStr = "url(#qrGradient)"
		} else {
			blockColor := opts.translateToRGBA(v)
			r, g, b, _ := blockColor.RGBA()
			fillStr = fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		}

		var pathData string
		switch v.Type() {
		case qrcode.QRType_FINDER:
			pathData = svgShape.GenerateSVGFinder(drawCtx, opts.qrGradient != nil)
		default:
			pathData = svgShape.GenerateSVGPath(drawCtx, opts.qrGradient != nil)
		}

		isComplexShape := strings.Contains(pathData, `stroke="`) || strings.Contains(pathData, `fill="`)

		if isComplexShape {
			if opts.qrGradient != nil {
				fmt.Fprintf(w, `<g fill="url(#qrGradient)">%s</g>\n`, pathData)
			} else {
				fmt.Fprintf(w, `<g>%s</g>\n`, pathData)
			}
		} else {
			fmt.Fprintf(w, `<g fill="%s">%s</g>\n`, fillStr, pathData)
		}
	})

	_, err = fmt.Fprintf(w, `</g>\n`)
	if err != nil {
		return err
	}

	if logoValid {
		if err := embedLogoAsPNG(w, opts.logo, width, height, logoWidth, logoHeight); err != nil {
			// Skip logo if encoding fails
		}
	}

	_, err = fmt.Fprintf(w, `</svg>`)
	return err
}
