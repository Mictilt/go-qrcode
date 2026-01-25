package standard

import (
	"image"
	"image/color"
	imagedraw "image/draw"
	"math/rand"
	"sort"
	"testing"
)

type rectInfo struct {
	x, y, width, height int
	r, g, b, a          uint8
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// -------------------------------------------------------------------------
// THE FASTEST METHOD (Grid-Aware + Sorted Merge)
// -------------------------------------------------------------------------
func optimizeRectanglesFast(img image.Image, bounds image.Rectangle) []rectInfo {
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

// -------------------------------------------------------------------------
// OLD LOGIC (For Comparison)
// -------------------------------------------------------------------------
func optimizeRectanglesOriginal(img image.Image, bounds image.Rectangle) []rectInfo {
	var allRects []rectInfo
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		x := bounds.Min.X
		for x < bounds.Max.X {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()
			r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)
			if a8 == 0 || (r8 == 255 && g8 == 255 && b8 == 255) {
				x++
				continue
			}
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
			allRects = append(allRects, rectInfo{
				x: startX, y: y, width: x - startX, height: 1,
				r: r8, g: g8, b: b8, a: a8,
			})
		}
	}
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
			for j := i + 1; j < len(allRects); j++ {
				if used[j] {
					continue
				}
				rect2 := allRects[j]
				if rect1.r != rect2.r || rect1.g != rect2.g || rect1.b != rect2.b || rect1.a != rect2.a {
					continue
				}
				if rect1.y == rect2.y && rect1.height == rect2.height {
					if rect1.x+rect1.width == rect2.x || rect2.x+rect2.width == rect1.x {
						newRects = append(newRects, rectInfo{
							x:      min(rect1.x, rect2.x),
							y:      rect1.y,
							width:  rect1.width + rect2.width,
							height: rect1.height,
							r:      rect1.r, g: rect1.g, b: rect1.b, a: rect1.a,
						})
						used[i] = true
						used[j] = true
						merged = true
						mergedThis = true
						break
					}
				}
				if rect1.x == rect2.x && rect1.width == rect2.width {
					if rect1.y+rect1.height == rect2.y || rect2.y+rect2.height == rect1.y {
						newRects = append(newRects, rectInfo{
							x:      rect1.x,
							y:      min(rect1.y, rect2.y),
							width:  rect1.width,
							height: rect1.height + rect2.height,
							r:      rect1.r, g: rect1.g, b: rect1.b, a: rect1.a,
						})
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
	return allRects
}

// -------------------------------------------------------------------------
// BENCHMARKS
// -------------------------------------------------------------------------
var (
	benchImg    *image.RGBA
	benchBounds image.Rectangle
	result      []rectInfo
)

func generateMockQRImage(width, height, blockSize int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	imagedraw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, imagedraw.Src)
	for y := 0; y < height; y += blockSize {
		for x := 0; x < width; x += blockSize {
			if rand.Float32() < 0.5 {
				rect := image.Rect(x, y, x+blockSize, y+blockSize)
				imagedraw.Draw(img, rect, &image.Uniform{color.Black}, image.Point{}, imagedraw.Src)
			}
		}
	}
	return img
}

func init() {
	benchImg = generateMockQRImage(300, 300, 3)
	benchBounds = benchImg.Bounds()
}

func BenchmarkOriginal(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		result = optimizeRectanglesOriginal(benchImg, benchBounds)
	}
}

func BenchmarkFast(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		result = optimizeRectanglesFast(benchImg, benchBounds)
	}
}

// go test -bench=. -benchmem

