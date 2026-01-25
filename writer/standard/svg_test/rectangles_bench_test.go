package standard

import (
	"image"
	"image/color"
	"image/draw"
	"math/rand"
	"sort"
	"testing"
)

type rectInfo struct {
	x, y, width, height int
	r, g, b, a          uint8
}

// optimizeRectangles uses Greedy Meshing to combine adjacent uniform pixels
// into the largest possible rectangles.
func optimizeRectangles(img image.Image, bounds image.Rectangle) []rectInfo {
	width := bounds.Dx()
	height := bounds.Dy()
	visited := make([]bool, width*height)
	var rects []rectInfo

	// Check if we can use the fast path for RGBA images
	rgbaImg, isRGBA := img.(*image.RGBA)

	// Helper to get pixel components without interface overhead if possible
	getPixel := func(x, y int) (uint8, uint8, uint8, uint8) {
		if isRGBA {
			offset := (y-rgbaImg.Rect.Min.Y)*rgbaImg.Stride + (x-rgbaImg.Rect.Min.X)*4
			return rgbaImg.Pix[offset], rgbaImg.Pix[offset+1], rgbaImg.Pix[offset+2], rgbaImg.Pix[offset+3]
		}
		r, g, b, a := img.At(x, y).RGBA()
		return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)
	}

	// Helper to check visited status
	isVisited := func(x, y int) bool {
		return visited[(y-bounds.Min.Y)*width+(x-bounds.Min.X)]
	}

	// Helper to mark a rectangle as visited
	markVisited := func(x, y, w, h int) {
		startY := y - bounds.Min.Y
		startX := x - bounds.Min.X
		for dy := 0; dy < h; dy++ {
			offset := (startY + dy) * width
			for dx := 0; dx < w; dx++ {
				visited[offset+startX+dx] = true
			}
		}
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if isVisited(x, y) {
				continue
			}

			r, g, b, a := getPixel(x, y)

			// Skip transparent or white pixels (background optimization)
			if a == 0 || (r == 255 && g == 255 && b == 255) {
				continue
			}

			// 1. Greedy Horizontal Expansion
			// Determine how wide this row of identical pixels is
			w := 1
			for x+w < bounds.Max.X {
				if isVisited(x+w, y) {
					break
				}
				nr, ng, nb, na := getPixel(x+w, y)
				if nr != r || ng != g || nb != b || na != a {
					break
				}
				w++
			}

			// 2. Greedy Vertical Expansion
			// Determine how many subsequent rows have this exact same horizontal segment
			h := 1
			for y+h < bounds.Max.Y {
				matchRow := true
				for k := 0; k < w; k++ {
					if isVisited(x+k, y+h) {
						matchRow = false
						break
					}
					nr, ng, nb, na := getPixel(x+k, y+h)
					if nr != r || ng != g || nb != b || na != a {
						matchRow = false
						break
					}
				}
				if !matchRow {
					break
				}
				h++
			}

			// 3. Add Rectangle and Mark Visited
			rects = append(rects, rectInfo{
				x: x, y: y, width: w, height: h,
				r: r, g: g, b: b, a: a,
			})
			markVisited(x, y, w, h)
			
			// Note: The outer X loop will naturally continue, hitting 'isVisited' true 
			// until it passes the width of this rect, or the Y loop increments.
		}
	}

	return rects
}
// --- OLD FUNCTION (Original 4-Pass Logic) ---
func oldOptimizeRectangles(img image.Image, bounds image.Rectangle) []rectInfo {
	width := bounds.Dx()
	height := bounds.Dy()
	visited := make([]bool, width*height)
	var rects []rectInfo

	getIndex := func(x, y int) int {
		return (y-bounds.Min.Y)*width + (x - bounds.Min.X)
	}

	// PHASE 1: Scan for 3x3 Blocks
	for y := bounds.Min.Y; y <= bounds.Max.Y-3; y += 3 {
		for x := bounds.Min.X; x <= bounds.Max.X-3; x += 3 {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()
			r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)
			if a8 == 0 || (r8 == 255 && g8 == 255 && b8 == 255) {
				continue
			}
			isUniform := true
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
				rects = append(rects, rectInfo{x: x, y: y, width: 3, height: 3, r: r8, g: g8, b: b8, a: a8})
				for dy := 0; dy < 3; dy++ {
					for dx := 0; dx < 3; dx++ {
						visited[getIndex(x+dx, y+dy)] = true
					}
				}
			}
		}
	}

	// PHASE 2: Clean up remaining
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
			rects = append(rects, rectInfo{x: startX, y: y, width: x - startX, height: 1, r: r8, g: g8, b: b8, a: a8})
		}
	}
	if len(rects) == 0 {
		return rects
	}

	// PHASE 3: Horizontal Merge
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
		if curr.y == next.y && curr.height == next.height && curr.x+curr.width == next.x &&
			curr.r == next.r && curr.g == next.g && curr.b == next.b && curr.a == next.a {
			curr.width += next.width
		} else {
			mergedHorz = append(mergedHorz, curr)
			curr = next
		}
	}
	mergedHorz = append(mergedHorz, curr)

	// PHASE 4: Vertical Merge
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
			if curr.x == next.x && curr.width == next.width && curr.y+curr.height == next.y &&
				curr.r == next.r && curr.g == next.g && curr.b == next.b && curr.a == next.a {
				curr.height += next.height
			} else {
				finalRects = append(finalRects, curr)
				curr = next
			}
		}
		finalRects = append(finalRects, curr)
	}
	return finalRects
}

// --- NEW FUNCTION (Greedy Meshing) ---
// This assumes 'optimizeRectangles' is defined in your main package file as per the corrected code above.

// --- TEST SETUP ---

func generateRandomImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Fill with random rects to simulate a QR code or blocky image
	for i := 0; i < 50; i++ {
		rw := rand.Intn(w/4) + 1
		rh := rand.Intn(h/4) + 1
		rx := rand.Intn(w - rw)
		ry := rand.Intn(h - rh)
		col := color.RGBA{uint8(rand.Intn(2) * 255), uint8(rand.Intn(2) * 255), uint8(rand.Intn(2) * 255), 255}
		draw.Draw(img, image.Rect(rx, ry, rx+rw, ry+rh), &image.Uniform{col}, image.Point{}, draw.Src)
	}
	return img
}

func BenchmarkOptimizeRectangles_Old(b *testing.B) {
	img := generateRandomImage(500, 500)
	bounds := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		oldOptimizeRectangles(img, bounds)
	}
}

func BenchmarkOptimizeRectangles_New(b *testing.B) {
	img := generateRandomImage(500, 500)
	bounds := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizeRectangles(img, bounds)
	}
}