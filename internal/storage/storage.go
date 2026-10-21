package storage

import (
	"context"
	"encoding/base64"
	"fmt"
)

// Storage is the interface for storing processed images.
// Implementations can store as Base64 in the database,
// in S3, or any other backend.
type Storage interface {
	// Store saves the image data and returns a reference (Base64 string or URL).
	Store(ctx context.Context, productID int, imageData []byte, index int) (string, error)
}

// DatabaseStorage stores images as Base64-encoded strings.
// The returned value is a Base64 data URL that can be stored in the DB.
type DatabaseStorage struct{}

func NewDatabaseStorage() *DatabaseStorage {
	return &DatabaseStorage{}
}

func (s *DatabaseStorage) Store(ctx context.Context, productID int, imageData []byte, index int) (string, error) {
	if len(imageData) == 0 {
		return "", fmt.Errorf("empty image data for product %d index %d", productID, index)
	}

	// Detect content type from the data (JPEG or PNG)
	contentType := detectContentType(imageData)

	// Encode as Base64 and return as a data URL
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, encoded)

	return dataURL, nil
}

// detectContentType checks magic bytes to determine image type.
func detectContentType(data []byte) string {
	if len(data) >= 2 {
		switch {
		case data[0] == 0xFF && data[1] == 0xD8:
			return "image/jpeg"
		case data[0] == 0x89 && data[1] == 0x50: // PNG starts with \x89PNG
			return "image/png"
		}
	}
	return "image/jpeg" // default fallback
}