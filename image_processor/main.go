package main

import (
    "encoding/json"
    "log"
    "github.com/streadway/amqp"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "github.com/nfnt/resize"
    "image/jpeg"
    "image/png"
    "strings"
)

func main() {
    // Connect to RabbitMQ
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    if err != nil {
        log.Fatalf("Failed to connect to RabbitMQ: %v", err)
    }
    defer conn.Close()

    ch, err := conn.Channel()
    if err != nil {
        log.Fatalf("Failed to open channel: %v", err)
    }
    defer ch.Close()

    // Declare the same queue as in the main service
    q, err := ch.QueueDeclare(
        "image_processing_queue",
        true,
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        log.Fatalf("Failed to declare queue: %v", err)
    }

    msgs, err := ch.Consume(
        q.Name,
        "",
        false,  // auto-ack set to false for manual acknowledgment
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        log.Fatalf("Failed to register a consumer: %v", err)
    }

    forever := make(chan bool)
    go func() {
        for d := range msgs {
            var urls []string
            if err := json.Unmarshal(d.Body, &urls); err != nil {
                log.Printf("Error unmarshaling message: %v", err)
                d.Nack(false, true) // Negative acknowledgment, requeue the message
                continue
            }

            processedUrls, err := processImages(urls)
            if err != nil {
                log.Printf("Error processing images: %v", err)
                d.Nack(false, true)
                continue
            }

            // TODO: Update the database with processed image URLs
            
            d.Ack(false) // Acknowledge the message
            log.Printf("Successfully processed images: %v", processedUrls)
        }
    }()

    log.Println("Image processor is running. To exit press CTRL+C")
    <-forever
}

func processImages(urls []string) ([]string, error) {
    processedUrls := make([]string, 0)
    for _, url := range urls {
        processedUrl, err := processImage(url)
        if err != nil {
            return nil, err
        }
        processedUrls = append(processedUrls, processedUrl)
    }
    return processedUrls, nil
}

func processImage(url string) (string, error) {
    // Download image
    resp, err := http.Get(url)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    // Create temp file
    tempFile, err := os.CreateTemp("", "image-*"+filepath.Ext(url))
    if err != nil {
        return "", err
    }
    defer tempFile.Close()

    // Copy downloaded image to temp file
    _, err = io.Copy(tempFile, resp.Body)
    if err != nil {
        return "", err
    }

    // Process image (compress)
    // TODO: Implement actual image compression logic
    // For now, we'll just return the original URL
    return url, nil
} 