package media

import (
	"image"
	"image/color"
	"testing"
)

func TestGenerateItem(t *testing.T) {
	item := GenerateItem(1, 100, 50)
	if item.Width != 100 || item.Height != 50 {
		t.Errorf("expected 100x50, got %dx%d", item.Width, item.Height)
	}
	if len(item.Pixels) != 5000 {
		t.Errorf("expected 5000 pixels, got %d", len(item.Pixels))
	}
}

func TestToGray(t *testing.T) {
	pixels := make([]uint8, 100)
	for i := range pixels {
		pixels[i] = uint8(i)
	}
	img := ToGray(pixels, 10, 10)
	if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 10 {
		t.Errorf("expected 10x10, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
	if img.Pix[5] != 5 {
		t.Errorf("pixel 5 expected 5, got %d", img.Pix[5])
	}
}

func TestNearestNeighbor(t *testing.T) {
	src := image.NewGray(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			src.SetGray(x, y, color.Gray{Y: uint8((x + y) * 40)})
		}
	}
	dst := ResizeNearest(src, 2, 2)
	if dst.Bounds().Dx() != 2 || dst.Bounds().Dy() != 2 {
		t.Errorf("expected 2x2, got %dx%d", dst.Bounds().Dx(), dst.Bounds().Dy())
	}
}

func TestBilinear(t *testing.T) {
	src := image.NewGray(image.Rect(0, 0, 4, 4))
	src.SetGray(0, 0, color.Gray{Y: 0})
	src.SetGray(3, 3, color.Gray{Y: 255})
	dst := ResizeBilinear(src, 2, 2)
	if dst.Bounds().Dx() != 2 || dst.Bounds().Dy() != 2 {
		t.Errorf("expected 2x2, got %dx%d", dst.Bounds().Dx(), dst.Bounds().Dy())
	}
}

func TestGenerateThumbnail(t *testing.T) {
	item := GenerateItem(1, 1920, 1080)
	spec := ThumbnailSpec{MaxWidth: 320, MaxHeight: 240, Quality: 80}
	thumb := GenerateThumbnail(item, spec, false)
	if thumb.Bounds().Dx() > 320 || thumb.Bounds().Dy() > 240 {
		t.Errorf("thumb too large: %dx%d", thumb.Bounds().Dx(), thumb.Bounds().Dy())
	}
}

func TestGenerateThumbnailBilinear(t *testing.T) {
	item := GenerateItem(2, 640, 480)
	spec := ThumbnailSpec{MaxWidth: 160, MaxHeight: 120, Quality: 90}
	thumb := GenerateThumbnail(item, spec, true)
	if thumb == nil {
		t.Fatal("bilinear thumb should not be nil")
	}
}

func TestGenerateItems(t *testing.T) {
	items := GenerateItems(10)
	if len(items) != 10 {
		t.Errorf("expected 10 items, got %d", len(items))
	}
}

func BenchmarkNearestResize(b *testing.B) {
	item := GenerateItem(1, 1920, 1080)
	src := ToGray(item.Pixels, item.Width, item.Height)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResizeNearest(src, 320, 240)
	}
}

func BenchmarkBilinearResize(b *testing.B) {
	item := GenerateItem(1, 1920, 1080)
	src := ToGray(item.Pixels, item.Width, item.Height)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResizeBilinear(src, 320, 240)
	}
}

func BenchmarkGenerateThumbnail(b *testing.B) {
	item := GenerateItem(1, 1920, 1080)
	spec := ThumbnailSpec{MaxWidth: 320, MaxHeight: 240}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateThumbnail(item, spec, false)
	}
}

func BenchmarkBatchThumbnails(b *testing.B) {
	items := GenerateItems(100)
	spec := ThumbnailSpec{MaxWidth: 320, MaxHeight: 240}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			GenerateThumbnail(item, spec, false)
		}
	}
}
