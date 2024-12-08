package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/streadway/amqp"
)

type RabbitMQPayload struct {
	ProductID int      `json:"product_id"`
	ImageURLs []string `json:"image_urls"`
}

// StartConsumer initializes the RabbitMQ consumer and begins processing messages.
func StartConsumer(db *pgxpool.Pool) {
	// Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Open a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open RabbitMQ channel: %v", err)
	}
	defer ch.Close()

	// Declare the queue
	_, err = ch.QueueDeclare(
		"image_processing_queue", // Queue name
		true,                     // Durable
		false,                    // Delete when unused
		false,                    // Exclusive
		false,                    // No-wait
		nil,                      // Arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// Consume messages
	msgs, err := ch.Consume(
		"image_processing_queue", // Queue name
		"",
		false, // Auto Ack
		false, // Exclusive
		false,
		false,
		nil,
	)

	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	// Start processing messages
	log.Println("Waiting for messages...")
	go func() {
		for msg := range msgs {
			processMessage(msg.Body, db)
		}
	}()
	select {} // Block indefinitely
}

// processMessage handles each RabbitMQ message
func processMessage(body []byte, db *pgxpool.Pool) {
	// Parse the message body
	var payload struct {
		ProductID int      `json:"product_id"`
		ImageURLs []string `json:"image_urls"`
	}
	err := json.Unmarshal(body, &payload)
	if err != nil {
		log.Printf("Error decoding message: %v", err)
		return
	}

	log.Printf("Processing images for Product ID: %d", payload.ProductID)

	// Process each image
	var compressedImagePaths []string
	for _, url := range payload.ImageURLs {
		compressedImage, err := downloadAndCompressImage(url)
		if err != nil {
			log.Printf("Failed to process image %s: %v", url, err)
			continue
		}

		// Save the compressed image locally
		filePath := fmt.Sprintf("./compressed_images/product_%d_%s.jpg", payload.ProductID, getFileNameFromURL(url))
		err = os.WriteFile(filePath, compressedImage, 0644)
		if err != nil {
			log.Printf("Failed to save compressed image: %v", err)
			continue
		}
		compressedImagePaths = append(compressedImagePaths, filePath)
	}

	// Update the database with compressed image paths
	err = updateDatabaseWithCompressedImages(payload.ProductID, compressedImagePaths, db)
	if err != nil {
		log.Printf("Failed to update database: %v", err)
		return
	}

	log.Printf("Successfully processed images for Product ID: %d", payload.ProductID)
}

// downloadAndCompressImage downloads and compresses an image from the given URL
func downloadAndCompressImage(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	var compressedBuffer bytes.Buffer
	err = jpeg.Encode(&compressedBuffer, img, &jpeg.Options{Quality: 75})
	if err != nil {
		return nil, fmt.Errorf("failed to compress image: %w", err)
	}

	return compressedBuffer.Bytes(), nil
}

// updateDatabaseWithCompressedImages updates the database with compressed image paths
func updateDatabaseWithCompressedImages(productID int, compressedImages []string, db *pgxpool.Pool) error {
	query := `UPDATE products SET compressed_product_images = $1 WHERE id = $2`
	_, err := db.Exec(context.Background(), query, compressedImages, productID)
	if err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}
	return nil
}

// getFileNameFromURL extracts the file name from an image URL
func getFileNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}
