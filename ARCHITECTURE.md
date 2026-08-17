# Architecture Diagram

## New Service Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CLIENT (REST API)                           │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                    POST /products  │  GET /products/:id
                                    ▼
         ┌──────────────────────────────────────────────────┐
         │          API SERVICE (cmd/api/main.go)          │
         │                                                   │
         │  ┌──────────────────────────────────────────┐   │
         │  │   HTTP Handler (Gin Framework)           │   │
         │  │   - POST /products                        │   │
         │  │   - GET /products/:id                     │   │
         │  └──────────────────────────────────────────┘   │
         │                      │                            │
         │                      ▼                            │
         │  ┌──────────────────────────────────────────┐   │
         │  │  ProductService (internal/service/)      │   │
         │  │  - CreateProduct()                       │   │
         │  │  - GetProductByID()                      │   │
         │  └──────────────────────────────────────────┘   │
         │          │              │              │         │
         │          ▼              ▼              ▼         │
         │  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
         │  │Repository │  │  Redis   │  │  RabbitMQ    │  │
         │  │  (DAO)    │  │  Cache   │  │  Publisher   │  │
         │  └──────────┘  └──────────┘  └──────────────┘  │
         └──────────────────────────────────────────────────┘
              │                │                  │
              ▼                ▼                  ▼
    ┌─────────────────┐ ┌────────────┐  ┌──────────────────┐
    │  PostgreSQL     │ │   Redis    │  │    RabbitMQ      │
    │  (Products DB)  │ │  (Cache)   │  │  (Message Queue) │
    └─────────────────┘ └────────────┘  └──────────────────┘
                                             │
                                             ▼
                    ┌────────────────────────────────────────────────┐
                    │   WORKER SERVICE (cmd/worker/main.go)         │
                    │                                                 │
                    │  ┌────────────────────────────────────────┐   │
                    │  │   RabbitMQ Consumer                    │   │
                    │  │   - Consume ImageProcessingMessage    │   │
                    │  └────────────────────────────────────────┘   │
                    │                   │                            │
                    │                   ▼                            │
                    │  ┌────────────────────────────────────────┐   │
                    │  │   Image Processor                       │   │
                    │  │   - Download image                     │   │
                    │  │   - Compress (JPEG 75%)               │   │
                    │  │   - Save to disk                       │   │
                    │  └────────────────────────────────────────┘   │
                    │                   │                            │
                    │                   ▼                            │
                    │  ┌────────────────────────────────────────┐   │
                    │  │   Repository Update                     │   │
                    │  │   - Update compressed_product_images   │   │
                    │  └────────────────────────────────────────┘   │
                    │                   │                            │
                    │                   ▼                            │
                    │             PostgreSQL DB                      │
                    └────────────────────────────────────────────────┘
```

## Data Flow

```
1. CREATE PRODUCT
   ┌─────────────────────────────────────────────────────────┐
   │ Client sends:                                            │
   │ {                                                        │
   │   "user_id": 1,                                         │
   │   "product_name": "Laptop",                             │
   │   "product_images": ["http://...jpg", ...]              │
   │   "product_price": 999.99                               │
   │ }                                                        │
   └─────────────────────────────────────────────────────────┘
                          │
                          ▼
   ┌─────────────────────────────────────────────────────────┐
   │ API Handler:                                             │
   │ - Validate input                                        │
   │ - Call ProductService.CreateProduct()                   │
   └─────────────────────────────────────────────────────────┘
                          │
                          ▼
   ┌─────────────────────────────────────────────────────────┐
   │ ProductRepository.Create():                              │
   │ - INSERT product into PostgreSQL                        │
   │ - RETURNING id                                          │
   │ - Returns product_id = 123                              │
   └─────────────────────────────────────────────────────────┘
                          │
                          ▼
   ┌─────────────────────────────────────────────────────────┐
   │ RabbitMQ Publisher:                                      │
   │ - Publish message: {product_id: 123, image_urls: [...]} │
   │ - Queue: image_processing_queue                         │
   └─────────────────────────────────────────────────────────┘
                          │
                          ▼
   ┌─────────────────────────────────────────────────────────┐
   │ Return to Client:                                        │
   │ { "message": "product created", "product_id": 123 }     │
   └─────────────────────────────────────────────────────────┘


2. BACKGROUND IMAGE PROCESSING (Worker)
   ┌─────────────────────────────────────────────────────────┐
   │ Worker consumes from RabbitMQ:                           │
   │ { product_id: 123, image_urls: [...] }                  │
   └─────────────────────────────────────────────────────────┘
                          │
                          ▼
   ┌─────────────────────────────────────────────────────────┐
   │ For each image:                                          │
   │ 1. Download from URL                                    │
   │ 2. Compress using JPEG at 75% quality                   │
   │ 3. Save to ./compressed_images/product_123_*.jpg        │
   └─────────────────────────────────────────────────────────┘
                          │
                          ▼
   ┌─────────────────────────────────────────────────────────┐
   │ Update PostgreSQL:                                       │
   │ UPDATE products                                         │
   │ SET compressed_product_images = [paths]                 │
   │ WHERE id = 123                                          │
   └─────────────────────────────────────────────────────────┘
                          │
                          ▼
   ┌─────────────────────────────────────────────────────────┐
   │ Acknowledge message:                                     │
   │ Message removed from RabbitMQ queue                      │
   └─────────────────────────────────────────────────────────┘


3. FETCH PRODUCT (with caching)
   ┌─────────────────────────────────────────────────────────┐
   │ Client: GET /products/123                                │
   └─────────────────────────────────────────────────────────┘
                          │
                          ▼
   ┌─────────────────────────────────────────────────────────┐
   │ API Handler:                                             │
   │ - Call ProductService.GetProductByID("123")              │
   └─────────────────────────────────────────────────────────┘
                          │
                          ▼
   ┌─────────────────────────────────────────────────────────┐
   │ Try Redis Cache (10-min TTL):                            │
   │ - Found? Return cached product                          │
   │ - Not found? Query PostgreSQL                           │
   └─────────────────────────────────────────────────────────┘
                          │
                          ▼
   ┌─────────────────────────────────────────────────────────┐
   │ Return to Client:                                        │
   │ {                                                        │
   │   "product": {                                           │
   │     "id": 123,                                           │
   │     "product_name": "Laptop",                            │
   │     "compressed_product_images": [paths]                │
   │     ...                                                  │
   │   }                                                      │
   │ }                                                        │
   └─────────────────────────────────────────────────────────┘
```

## Package Dependencies

```
cmd/api/main.go
    ├── internal/config
    ├── internal/db
    ├── internal/cache
    ├── internal/queue
    └── internal/service
        ├── internal/repository
        ├── internal/models
        └── internal/queue

cmd/worker/main.go
    ├── internal/config
    ├── internal/db
    ├── internal/repository
    ├── internal/models
    └── external libs (image processing)
```

## Environment Variables

```
DATABASE_URL         → PostgreSQL connection string
REDIS_ADDR          → Redis server address
RABBITMQ_URL        → RabbitMQ connection URL
QUEUE_NAME          → Message queue name
```

All default to local development values for quick setup!
