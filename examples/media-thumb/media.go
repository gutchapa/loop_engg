// Package media provides image/video processing primitives.
// Loop Engineering target: minimize processing time per item (µs).
package media

import (
	"image"
	"image/color"
	"math"
)

// MediaType represents the type of media being processed.
type MediaType int

const (
	TypeImage MediaType = iota
	TypeVideo
	TypeAudio
)

// MediaItem represents a media file to process.
type MediaItem struct {
	ID        int
	Type      MediaType
	Width     int
	Height    int
	SizeBytes int64
	Duration  float64 // seconds
	Pixels    []uint8 // simulated pixel data (grayscale, W*H)
}

// ThumbnailSpec describes the desired thumbnail output.
type ThumbnailSpec struct {
	MaxWidth  int
	MaxHeight int
	Quality   int // 1-100
}

// ProcessResult holds the outcome of processing.
type ProcessResult struct {
	Item      MediaItem
	Thumbnail *image.Gray
	Duration  float64 // ms
}

// GenerateItem creates a randomized media item for benchmarking.
func GenerateItem(id int, width, height int) MediaItem {
	pixels := make([]uint8, width*height)
	// Fill with a simple pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixels[y*width+x] = uint8((x + y) * 7 % 256)
		}
	}
	return MediaItem{
		ID:        id,
		Type:      TypeImage,
		Width:     width,
		Height:    height,
		SizeBytes: int64(width * height),
		Pixels:    pixels,
	}
}

// GenerateItems creates n media items for benchmarking.
func GenerateItems(n int) []MediaItem {
	items := make([]MediaItem, n)
	sizes := [][2]int{
		{640, 480},
		{1280, 720},
		{1920, 1080},
		{3840, 2160},
	}
	for i := range items {
		s := sizes[i%len(sizes)]
		items[i] = GenerateItem(i+1, s[0], s[1])
	}
	return items
}

// NearestNeighbor resize — fastest, lowest quality.
func ResizeNearest(src *image.Gray, width, height int) *image.Gray {
	dst := image.NewGray(image.Rect(0, 0, width, height))
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	xRatio := float64(srcW) / float64(width)
	yRatio := float64(srcH) / float64(height)

	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			sx := int(math.Floor(float64(dx) * xRatio))
			sy := int(math.Floor(float64(dy) * yRatio))
			dst.SetGray(dx, dy, src.GrayAt(sx, sy))
		}
	}
	return dst
}

// Bilinear resize — smoother, slower.
func ResizeBilinear(src *image.Gray, width, height int) *image.Gray {
	dst := image.NewGray(image.Rect(0, 0, width, height))
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	xRatio := float64(srcW) / float64(width)
	yRatio := float64(srcH) / float64(height)

	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			gx := float64(dx) * xRatio
			gy := float64(dy) * yRatio
			x1, y1 := int(math.Floor(gx)), int(math.Floor(gy))
			x2, y2 := x1+1, y1+1
			if x2 >= srcW {
				x2 = srcW - 1
			}
			if y2 >= srcH {
				y2 = srcH - 1
			}

			xf := gx - float64(x1)
			yf := gy - float64(y1)

			c00 := float64(src.GrayAt(x1, y1).Y)
			c10 := float64(src.GrayAt(x2, y1).Y)
			c01 := float64(src.GrayAt(x1, y2).Y)
			c11 := float64(src.GrayAt(x2, y2).Y)

			top := c00*(1-xf) + c10*xf
			bot := c01*(1-xf) + c11*xf
			val := uint8(top*(1-yf) + bot*yf)
			dst.SetGray(dx, dy, color.Gray{Y: val})
		}
	}
	return dst
}

// ToGray converts pixel data to a grayscale image.
func ToGray(pixels []uint8, width, height int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, width, height))
	copy(img.Pix, pixels)
	return img
}

// GenerateThumbnail creates a thumbnail using the specified method.
func GenerateThumbnail(item MediaItem, spec ThumbnailSpec, useBilinear bool) *image.Gray {
	src := ToGray(item.Pixels, item.Width, item.Height)

	// Calculate output dimensions maintaining aspect ratio
	ratio := math.Min(
		float64(spec.MaxWidth)/float64(item.Width),
		float64(spec.MaxHeight)/float64(item.Height),
	)
	outW := int(math.Max(1, float64(item.Width)*ratio))
	outH := int(math.Max(1, float64(item.Height)*ratio))

	if useBilinear {
		return ResizeBilinear(src, outW, outH)
	}
	return ResizeNearest(src, outW, outH)
}
