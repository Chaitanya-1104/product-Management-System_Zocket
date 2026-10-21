package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"product-management-system/internal/cache"
	"product-management-system/internal/config"
	"product-management-system/internal/db"
	"product-management-system/internal/models"
	"product-management-system/internal/queue"
	"product-management-system/internal/service"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	dbPool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer dbPool.Close()

	redisClient := cache.NewClient(cfg.RedisAddr)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("redis ping failed: %v", err)
	}
	defer redisClient.Close()

	mq, err := queue.New(cfg.RabbitURL, cfg.QueueName)
	if err != nil {
		log.Fatalf("connect to rabbitmq: %v", err)
	}
	defer mq.Close()

	productService := service.NewProductService(dbPool, redisClient, mq)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/products", func(c *gin.Context) {
		var input models.ProductInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		productID, err := productService.CreateProduct(c, input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "product created", "product_id": productID})
	})

	r.GET("/products/:id", func(c *gin.Context) {
		product, err := productService.GetProductByID(c, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"product": product})
	})

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("start server: %v", err)
	}
}
