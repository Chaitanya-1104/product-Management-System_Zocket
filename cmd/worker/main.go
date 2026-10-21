package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"image/jpeg"
	"log"
	"net/http"

	"github.com/streadway/amqp"

	"product-management-system/internal/config"
	"product-management-system/internal/db"
	"product-management-system/internal/models"
	"product-management-system/internal/repository"
	"product-management-system/internal/storage"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	dbPool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer dbPool.Close()

	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		log.Fatalf("connect to rabbitmq: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("open rabbitmq channel: %v", err)
	}
	defer ch.Close()

	if _, err := ch.QueueDeclare(cfg.QueueName, true, false, false, false, nil); err != nil {
		log.Fatalf("declare queue: %v", err)
	}

	msgs, err := ch.Consume(cfg.QueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("register consumer: %v", err)
	}

	// Create storage backend based on config
	var st storage.Storage
	switch cfg.StorageType {
	case "s3":
		// Future: S3Storage implementation
		log.Println("using S3 storage (not yet implemented, falling back to database)")
		st = storage.NewDatabaseStorage()
	default:
		log.Println("using database storage (Base64)")
		st = storage.NewDatabaseStorage()
	}

	log.Println("Image processor worker started, waiting for messages...")

	repo := repository.NewProductRepository(dbPool)

	for msg := range msgs {
		var payload models.ImageProcessingMessage
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			log.Printf("error unmarshaling message: %v", err)
			msg.Nack(false, true)
			continue
		}

		log.Printf("processing images for product %d", payload.ProductID)

		compressedReferences := []string{}
		for index, imageURL := range payload.ImageURLs {
			compressedImage, err := downloadAndCompressImage(imageURL)
			if err != nil {
				log.Printf("error processing image %s: %v", imageURL, err)
				continue
			}

			// Store via the storage backend and get a reference (Base64 data URL)
			ref, err := st.Store(ctx, payload.ProductID, compressedImage, index)
			if err != nil {
				log.Printf("error storing image for product %d: %v", payload.ProductID, err)
				continue
			}

			compressedReferences = append(compressedReferences, ref)
		}

		if len(compressedReferences) > 0 {
			if err := repo.UpdateCompressedImages(ctx, payload.ProductID, compressedReferences); err != nil {
				log.Printf("error updating database: %v", err)
				msg.Nack(false, true)
				continue
			}
		}

		msg.Ack(false)
		log.Printf("successfully processed images for product %d", payload.ProductID)
	}
}

func downloadAndCompressImage(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 75}); err != nil {
		return nil, fmt.Errorf("compress image: %w", err)
	}

	return buf.Bytes(), nil
}
