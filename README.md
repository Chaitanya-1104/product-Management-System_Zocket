# Product Management System

A production-ready Go microservices application for managing product details and asynchronous image processing using RabbitMQ, PostgreSQL, and Redis caching.

## Architecture Overview

The system is divided into two main services:

1. **API Service** (`cmd/api/`): REST API for product CRUD operations and queue publishing
2. **Worker Service** (`cmd/worker/`): Background worker that consumes image processing tasks from RabbitMQ

### Technology Stack

- **Framework**: Gin (REST API)
- **Database**: PostgreSQL with pgx (connection pooling)
- **Cache**: Redis (10-minute TTL for product lookups)
- **Message Queue**: RabbitMQ (asynchronous image processing)
- **Image Processing**: Go image library with JPEG compression
- **Containerization**: Docker & Docker Compose

### Project Structure

```
.
├── cmd/
│   ├── api/              # API service entrypoint
│   │   └── main.go
│   └── worker/           # Image processor worker entrypoint
│       └── main.go
├── internal/
│   ├── config/           # Configuration management
│   │   └── config.go
│   ├── models/           # Domain models
│   │   └── product.go
│   ├── db/               # Database initialization
│   │   └── database.go
│   ├── cache/            # Redis client initialization
│   │   └── redis.go
│   ├── queue/            # RabbitMQ client wrapper
│   │   └── rabbitmq.go
│   ├── repository/       # Data access layer
│   │   └── product_repository.go
│   └── service/          # Business logic layer
│       └── product_service.go
├── compressed_images/    # Local storage for processed images
├── docker-compose.yml    # Multi-container orchestration
├── Dockerfile            # Multi-stage build for API and worker
├── go.mod               # Go module dependencies
├── go.sum
└── README.md
```

## API Endpoints

### Health Check
```
GET /health
```
Returns: `{ "status": "ok" }`

### Create Product
```
POST /products
Content-Type: application/json

{
  "user_id": 1,
  "product_name": "Laptop",
  "product_description": "High-performance laptop",
  "product_images": [
    "https://example.com/image1.jpg",
    "https://example.com/image2.jpg"
  ],
  "product_price": 999.99
}
```
Returns: `{ "message": "product created", "product_id": 123 }`

### Get Product Details
```
GET /products/:id
```
Returns:
```json
{
  "product": {
    "id": 123,
    "user_id": 1,
    "product_name": "Laptop",
    "product_description": "High-performance laptop",
    "product_images": ["https://example.com/image1.jpg", ...],
    "product_price": 999.99,
    "compressed_product_images": [
      "./compressed_images/product_123_image1.jpg",
      ...
    ]
  }
}
```

## Message Flow

1. **Product Creation**: Client sends POST request with product details and image URLs
2. **Product Persisted**: Product record saved to PostgreSQL with empty `compressed_product_images` array
3. **Queue Message Published**: Product ID and image URLs published to RabbitMQ queue with message type:
   ```json
   {
     "product_id": 123,
     "image_urls": ["https://example.com/image1.jpg", ...]
   }
   ```
4. **Worker Consumes**: Worker fetches message from queue
5. **Image Processing**: Worker downloads each image and compresses using JPEG at 75% quality
6. **Database Update**: Compressed image file paths stored in PostgreSQL `compressed_product_images`
7. **Acknowledgment**: Message acknowledged and removed from queue

## Getting Started

### Prerequisites
- Docker and Docker Compose installed
- Go 1.20+ (for local development)

### Quick Start with Docker Compose

```bash
# Build and start all services
docker-compose up --build

# Services will be available at:
# API: http://localhost:8080
# RabbitMQ Dashboard: http://localhost:15672 (guest/guest)
# PostgreSQL: localhost:5432 (postgres/post123)
# Redis: localhost:6379
```

### Local Development Setup

```bash
# Install dependencies
go mod download

# Set environment variables
export DATABASE_URL="postgres://postgres:post123@localhost:5432/product_management"
export REDIS_ADDR="localhost:6379"
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
export QUEUE_NAME="image_processing_queue"

# Start PostgreSQL, Redis, and RabbitMQ (using docker-compose without api/worker)
docker-compose up database redis rabbitmq

# Run API service
go run cmd/api/main.go

# In another terminal, run worker service
go run cmd/worker/main.go
```

### Database Schema

```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    product_description TEXT NOT NULL,
    product_images TEXT[] NOT NULL,
    product_price DECIMAL(10, 2) NOT NULL,
    compressed_product_images TEXT[] DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_products_user_id ON products(user_id);
```

## Configuration

All services use environment variables for configuration:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://postgres:post123@localhost:5432/product_management` | PostgreSQL connection string |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | RabbitMQ connection URL |
| `QUEUE_NAME` | `image_processing_queue` | RabbitMQ queue name |

## Features

✅ **RESTful API** - Gin framework with proper HTTP status codes  
✅ **Database Pooling** - pgx with connection pooling for performance  
✅ **Redis Caching** - 10-minute TTL cache for product lookups  
✅ **Async Processing** - RabbitMQ-based image processing queue  
✅ **Image Compression** - Automatic JPEG compression at 75% quality  
✅ **Error Handling** - Robust error handling with proper logging  
✅ **Containerization** - Multi-stage Docker builds for optimized images  
✅ **Configuration Management** - Environment-based configuration  
✅ **Message Acknowledgment** - Manual ACK/NACK for reliable message processing  

## Future Improvements

- Add authentication and authorization (JWT)
- Implement unit and integration tests
- Add API request validation middleware
- Integrate S3/MinIO for cloud image storage instead of local storage
- Add graceful shutdown handlers
- Implement request rate limiting
- Add structured logging (zap or logrus)
- Add API documentation (Swagger/OpenAPI)
- Implement database migrations tool
- Add monitoring and metrics (Prometheus)

## Running Tests

```bash
go test ./...
```

## Troubleshooting

### Worker not processing images
- Check RabbitMQ connection: `docker-compose logs worker`
- Verify queue name matches: `QUEUE_NAME=image_processing_queue`
- Check database connection string

### Images not cached in Redis
- Verify Redis is running: `docker-compose exec redis redis-cli ping`
- Check Redis connection in API logs

### Database connection issues
- Verify PostgreSQL is running: `docker-compose ps database`
- Check credentials in DATABASE_URL

## License

MIT
