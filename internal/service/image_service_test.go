package service

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// createTestImage creates a test image of the specified size and format
func createTestImage(width, height int, format string) ([]byte, string) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a solid color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	var filename string

	switch format {
	case "jpeg":
		jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
		filename = "test.jpg"
	case "png":
		png.Encode(&buf, img)
		filename = "test.png"
	default:
		jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
		filename = "test.jpg"
	}

	return buf.Bytes(), filename
}

func TestValidateImage_ValidJPEG(t *testing.T) {
	svc := NewImageService(nil)
	data, filename := createTestImage(100, 100, "jpeg")

	err := svc.ValidateImage(data, filename)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateImage_ValidPNG(t *testing.T) {
	svc := NewImageService(nil)
	data, filename := createTestImage(100, 100, "png")

	err := svc.ValidateImage(data, filename)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateImage_TooLarge(t *testing.T) {
	svc := NewImageService(nil)
	// Create data larger than 5MB
	data := make([]byte, MaxImageSize+1)

	err := svc.ValidateImage(data, "test.jpg")
	if err != ErrImageTooLarge {
		t.Errorf("expected ErrImageTooLarge, got %v", err)
	}
}

func TestValidateImage_InvalidFormat(t *testing.T) {
	svc := NewImageService(nil)
	data, _ := createTestImage(100, 100, "jpeg")

	err := svc.ValidateImage(data, "test.gif")
	if err != ErrInvalidFormat {
		t.Errorf("expected ErrInvalidFormat, got %v", err)
	}
}

func TestValidateImage_TooSmall(t *testing.T) {
	svc := NewImageService(nil)
	data, filename := createTestImage(30, 30, "jpeg")

	err := svc.ValidateImage(data, filename)
	if err != ErrImageTooSmall {
		t.Errorf("expected ErrImageTooSmall, got %v", err)
	}
}

func TestValidateImage_InvalidData(t *testing.T) {
	svc := NewImageService(nil)
	// Invalid image data
	data := []byte("not an image")

	err := svc.ValidateImage(data, "test.jpg")
	if err != ErrInvalidImageData {
		t.Errorf("expected ErrInvalidImageData, got %v", err)
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"test.jpg", "image/jpeg"},
		{"test.jpeg", "image/jpeg"},
		{"test.png", "image/png"},
		{"test.webp", "image/webp"},
		{"test.gif", "application/octet-stream"},
		{"test.txt", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			ct := GetContentType(tt.filename)
			if ct != tt.expected {
				t.Errorf("GetContentType(%s) = %s, expected %s", tt.filename, ct, tt.expected)
			}
		})
	}
}

func TestIsValidImageFormat(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/webp", true},
		{"image/gif", false},
		{"text/plain", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := IsValidImageFormat(tt.contentType)
			if result != tt.expected {
				t.Errorf("IsValidImageFormat(%s) = %v, expected %v", tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestExtractBasePath(t *testing.T) {
	svc := NewImageService(nil)

	tests := []struct {
		url      string
		expected string
	}{
		{"http://localhost:9000/bucket/1/notes/1/abc123_thumb.jpg", "http://localhost:9000/bucket/1/notes/1/abc123"},
		{"http://localhost:9000/bucket/1/notes/1/abc123_display.jpg", "http://localhost:9000/bucket/1/notes/1/abc123"},
		{"http://localhost:9000/bucket/1/notes/1/abc123_original.jpg", "http://localhost:9000/bucket/1/notes/1/abc123"},
		{"http://localhost:9000/bucket/1/notes/1/abc123.jpg", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := svc.extractBasePath(tt.url)
			if result != tt.expected {
				t.Errorf("extractBasePath(%s) = %s, expected %s", tt.url, result, tt.expected)
			}
		})
	}
}
