package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
)

// Define the payload expected by RabbitMQ consumer
type RabbitMQPayload struct {
	ProductID int      `json:"product_id"`
	ImageURLs []string `json:"image_urls"`
}

func CreateProductHandler(ch *amqp.Channel) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			ProductID   int      `json:"product_id"`
			ProductImages []string `json:"product_images"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Map the request data to RabbitMQ Payload
		rabbitPayload := RabbitMQPayload{
			ProductID: request.ProductID, // Generate a mock ID or pull this dynamically
			ImageURLs: request.ProductImages,
		}

		// Marshal payload to JSON
		messageBody, err := json.Marshal(rabbitPayload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not marshal payload"})
			return
		}

		// Publish the payload to RabbitMQ
		err = ch.Publish(
			"",
			"image_processing_queue",
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        messageBody,
			},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish message"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status":"messa sent to rabbitmq"})
	}
}
