package main

import (
	"context"

	"encoding/json"
	"log"
	"time"

	"product-management-system/consumer"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lib/pq"
	"github.com/streadway/amqp"
)

var ctx = context.Background()

// db connection
var db *pgxpool.Pool
var redisClient *redis.Client

var rabbitConn *amqp.Connection
var rabbitChannel *amqp.Channel

func init() {
	//db conn.
	var err error
	db, err = pgxpool.Connect(context.Background(), "postgres://postgres:post123@localhost:5432/product_management")
	if err != nil {
		log.Fatalf("Unable to connect database: %v", err)
	}
	log.Println("Connected successfully to database")

	//redis connn.
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	log.Println("Connected to Redis.")
	//rabbitmq conn.
	rabbitConn, err = amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("failes to conn to rabbitmq-%v", err)
	}
	log.Println("Connnected to rabbitmq successfully")

	rabbitChannel, err = rabbitConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel to RabbitMQ: %v", err)
	}
	//queue
	_, err = rabbitChannel.QueueDeclare(
		"image_processing_queue", // Queue name
		true,                     // Durable
		false,                    // Delete when unused
		false,                    // Exclusive
		false,                    // No-wait
		nil,                      // Arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare queue-%v", err)
	}
	log.Println("rabitmq queue declared.")
}

type Product struct {
	ID                      int      `json:"id"`
	UserID                  int      `json:"user_id" binding:"required"`
	ProductName             string   `json:"product_name" binding:"required"`
	ProductDescription      string   `json:"product_description" binding:"required"`
	ProductImages           []string `json:"product_images"`
	ProductPrice            float64  `json:"product_price" binding:"required"`
	CompressedProductImages []string `json:"compressed_product_images"`
}

func main() {
	//rabbitmq consumer

	log.Println("strting rabbitmq consumer...")
	go consumer.StartConsumer(db)

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Welcome to the Product Management System!"})
	})

	//post add products
	r.POST("/products", func(c *gin.Context) {
		var newProduct Product
		if err := c.ShouldBindJSON(&newProduct); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(
			context.Background(),
			"INSERT INTO products (user_id,product_name,product_description,product_images,product_price,compressed_product_images) VALUES ($1,$2,$3,$4,$5,'{}')",
			newProduct.UserID,
			newProduct.ProductName,
			newProduct.ProductDescription,
			pq.Array(newProduct.ProductImages),
			newProduct.ProductPrice,
		)

		if err != nil {
			log.Printf("Error saving to databse: %v", err)
			c.JSON(500, gin.H{"error": "unable to save product to database"})
			return
		}

		msg, _ := json.Marshal(newProduct.ProductImages)
		err = rabbitChannel.Publish(
			"",
			"image_processing_queue",
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        msg,
			},
		)
		if err != nil {
			c.JSON(500, gin.H{"error": "unable to send mess to rabit"})
			log.Println("Error sending to rabbtmq", err)
			return
		}

		c.JSON(201, gin.H{"message": "Product added", "product": newProduct})

		//products = append(products, newProduct)
		//c.JSON(201, gin.H{"message": "Product added", "product": newProduct})
	})

	//fetch product details using redis
	r.GET("products/:id", func(c *gin.Context) {
		id := c.Param("id")
		cachedProduct, err := redisClient.Get(ctx, id).Result()
		if err == nil {
			//if found in cache
			log.Println("Cache hit for id:", id)
			c.JSON(200, gin.H{"product": json.RawMessage(cachedProduct)})
			return
		}
		//if not in cache
		if err != redis.Nil {
			log.Printf("Error while fetching Redis data: %v", err)
			c.JSON(500, gin.H{"error": "Redis fetch error"})
			return
		}
		//if not in cache fetch from db
		var product Product
		err = db.QueryRow(
			ctx,
			"SELECT id,user_id, product_name, product_description, product_images, product_price, compressed_product_images FROM products WHERE id=$1",
			id,
		).Scan(
			&product.ID,
			&product.UserID,
			&product.ProductName,
			&product.ProductDescription,
			pq.Array(&product.ProductImages),
			&product.ProductPrice,
			pq.Array(&product.CompressedProductImages),
		)

		if err != nil {
			log.Printf("Error fetching from db: %v", err)
			c.JSON(500, gin.H{"error": "unable to fetch product from database"})
			return
		}

		// Cache result in Redis for faster future lookups
		productJSON, _ := json.Marshal(product)

		err = redisClient.Set(ctx, id, productJSON, 10*time.Minute).Err() // Set cache for 10 minutes
		if err != nil {
			log.Printf("failed to cache product in redis:%v", err)
		}
		c.JSON(200, gin.H{"product": product})

	})
	//fetch products by useer id
	r.GET("/products", func(c *gin.Context) {
		userID := c.Query("user_id")
		priceRange := c.Query("price_range")
		productName := c.Query("product_name")

		query := "SELECT id, user_id, product_name, product_description, product_images, product_price, compressed_product_images FROM products WHERE user_id=$1"
		args := []interface{}{userID}

		if priceRange != "" {
			query += " AND product_price BETWEEN $2 AND $3"
			args = append(args, priceRange)
		}

		if productName != "" {
			query += " AND product_name ILIKE $4"
			args = append(args, "%"+productName+"%")
		}

		log.Printf("Query: %s", query)
		log.Printf("Args: %v", args)

		rows, err := db.Query(ctx, query, args...)
		if err != nil {
			log.Printf("err fetching prods -%v", err)
			c.JSON(500, gin.H{"error": "unable to fetch products"})
			return
		}
		defer rows.Close()

		var products []Product
		for rows.Next() {
			var p Product
			if err := rows.Scan(
				&p.ID,
				&p.UserID,
				&p.ProductName,
				&p.ProductDescription,
				pq.Array(&p.ProductImages),
				&p.ProductPrice,
				pq.Array(&p.CompressedProductImages),
			); err != nil {
				log.Printf("Error parsing row: %v", err)
				c.JSON(500, gin.H{"error": "unable to parse database row"})
				return
			}
			products = append(products, p)
		}

		c.JSON(200, gin.H{"products": products})
	})

	r.Run(":8080") // Start the server on port 8080
}
