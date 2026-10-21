# API Testing Guide

Complete guide for testing the Product Management System API endpoints.

## Setup

Make sure services are running:
```bash
docker-compose up --build
```

## API Endpoints

### 1. Health Check

Check if the API is running:

**Request:**
```bash
curl -X GET http://localhost:8080/health
```

**Response:**
```json
{
  "status": "ok"
}
```

**Status Code:** 200 OK

---

### 2. Create Product

Create a new product with images for processing.

**Request:**
```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "product_name": "MacBook Pro",
    "product_description": "High-performance laptop for professionals",
    "product_images": [
      "https://via.placeholder.com/400x300",
      "https://via.placeholder.com/400x300"
    ],
    "product_price": 1999.99
  }'
```

**Response:**
```json
{
  "message": "product created",
  "product_id": 1
}
```

**Status Code:** 201 Created

**Notes:**
- `user_id`: Required, user identifier
- `product_name`: Required, product name (max 255 chars)
- `product_description`: Required, detailed description
- `product_images`: Array of public image URLs
- `product_price`: Required, must be positive
- Images will be automatically queued for compression via RabbitMQ
- Worker will download, compress, and save images to database

---

### 3. Get Product Details

Fetch product information (with cached results).

**Request:**
```bash
curl -X GET http://localhost:8080/products/1
```

**Response (before image processing):**
```json
{
  "product": {
    "id": 1,
    "user_id": 1,
    "product_name": "MacBook Pro",
    "product_description": "High-performance laptop for professionals",
    "product_images": [
      "https://via.placeholder.com/400x300",
      "https://via.placeholder.com/400x300"
    ],
    "product_price": 1999.99,
    "compressed_product_images": []
  }
}
```

**Response (after worker processes images, ~5-10 seconds):**
```json
{
  "product": {
    "id": 1,
    "user_id": 1,
    "product_name": "MacBook Pro",
    "product_description": "High-performance laptop for professionals",
    "product_images": [
      "https://via.placeholder.com/400x300",
      "https://via.placeholder.com/400x300"
    ],
    "product_price": 1999.99,
    "compressed_product_images": [
      "./compressed_images/product_1_400x300.jpg",
      "./compressed_images/product_1_400x300.jpg"
    ]
  }
}
```

**Status Code:** 200 OK

**Notes:**
- `compressed_product_images` will be empty until worker processes the images
- Results are cached in Redis for 10 minutes
- Try again after a few seconds if `compressed_product_images` is empty

---

## Test Scenarios

### Scenario 1: Complete Flow

Test the entire product creation and image processing flow:

```bash
#!/bin/bash

echo "1️⃣  Creating product..."
RESPONSE=$(curl -s -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "product_name": "Test Product",
    "product_description": "Test Description",
    "product_images": ["https://via.placeholder.com/300"],
    "product_price": 49.99
  }')

PRODUCT_ID=$(echo $RESPONSE | grep -o '"product_id":[0-9]*' | grep -o '[0-9]*')
echo "Created product ID: $PRODUCT_ID"

echo ""
echo "2️⃣  Fetching product (images not yet compressed)..."
curl -s http://localhost:8080/products/$PRODUCT_ID | jq '.product.compressed_product_images'

echo ""
echo "3️⃣  Waiting for worker to process images (5 seconds)..."
sleep 5

echo "4️⃣  Fetching product (images should be compressed now)..."
curl -s http://localhost:8080/products/$PRODUCT_ID | jq '.product.compressed_product_images'

echo ""
echo "✅ Complete!"
```

### Scenario 2: Error Handling

Test various error conditions:

**Missing required field:**
```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "product_name": "Test"
    # Missing product_description and product_price
  }'
```

**Expected Response:**
```json
{
  "error": "Key: 'ProductInput.ProductDescription' Error:Field validation for 'ProductDescription' failed on the 'required' tag..."
}
```

**Status Code:** 400 Bad Request

**Invalid product ID:**
```bash
curl -X GET http://localhost:8080/products/99999
```

**Expected Response:**
```json
{
  "error": "fetch product by id: no rows in result set"
}
```

**Status Code:** 404 Not Found

---

## Monitoring & Debugging

### View API Logs

```bash
docker-compose logs -f api
```

### View Worker Logs

```bash
docker-compose logs -f worker
```

### Monitor RabbitMQ Queue

Access the RabbitMQ management interface:
- URL: http://localhost:15672
- Username: `guest`
- Password: `guest`

Navigate to:
- **Queues** → `image_processing_queue` to see pending/processing messages
- **Connections** to verify worker is connected
- **Channels** to check message throughput

### Check Compressed Images

View the compressed images stored locally:

```bash
ls -la compressed_images/
```

Example output:
```
-rw-r--r-- 1 user group  15234 Aug 14 10:30 product_1_300.jpg
-rw-r--r-- 1 user group  14567 Aug 14 10:30 product_1_300.jpg
```

### Database Query Examples

Connect to PostgreSQL:

```bash
docker-compose exec database psql -U postgres -d product_management
```

Sample queries:

```sql
-- List all products
SELECT id, product_name, product_price FROM products;

-- Get product with most images
SELECT id, product_name, array_length(product_images, 1) as image_count 
FROM products 
ORDER BY array_length(product_images, 1) DESC;

-- Get products with compressed images
SELECT id, product_name, compressed_product_images 
FROM products 
WHERE array_length(compressed_product_images, 1) > 0;

-- Count products by user
SELECT user_id, COUNT(*) as product_count 
FROM products 
GROUP BY user_id;
```

### Redis Cache Check

Check what's cached:

```bash
docker-compose exec redis redis-cli
```

Commands:

```bash
# List all keys
KEYS *

# Get specific product from cache
GET 1

# Clear cache
FLUSHALL

# Monitor cache hits in real-time
MONITOR
```

---

## Performance Testing

### Load Test with Multiple Products

```bash
#!/bin/bash

echo "Creating 10 products..."
for i in {1..10}; do
    curl -s -X POST http://localhost:8080/products \
      -H "Content-Type: application/json" \
      -d "{
        \"user_id\": $((i % 3 + 1)),
        \"product_name\": \"Product $i\",
        \"product_description\": \"Description for product $i\",
        \"product_images\": [\"https://via.placeholder.com/300\"],
        \"product_price\": $((i * 10)).99
      }" &
done

wait
echo "✅ All products created"
```

### Cache Hit Rate

The API automatically caches results in Redis. Monitor cache performance:

```bash
# First request (cache miss)
time curl -s http://localhost:8080/products/1 > /dev/null

# Second request (cache hit - should be faster)
time curl -s http://localhost:8080/products/1 > /dev/null
```

---

## Troubleshooting

### Product created but no compressed images after 30 seconds

1. **Check worker logs:**
   ```bash
   docker-compose logs worker
   ```

2. **Verify RabbitMQ connection:**
   - Access http://localhost:15672
   - Check if messages are in the queue

3. **Check database directly:**
   ```bash
   docker-compose exec database psql -U postgres -d product_management
   SELECT * FROM products;
   ```

4. **Verify image URLs are valid:**
   - Test if the image URLs are accessible
   - Use `wget` or `curl` to verify

### API returns 500 error

1. **Check database connection:**
   ```bash
   docker-compose exec database pg_isready
   ```

2. **Check PostgreSQL logs:**
   ```bash
   docker-compose logs database
   ```

3. **Verify environment variables:**
   ```bash
   docker-compose exec api env | grep DATABASE_URL
   ```

### High latency on first request

This is normal - Redis cache is empty. Subsequent requests to the same product will be faster (10 minutes TTL).

---

## Next Steps

- Implement authentication (JWT)
- Add API rate limiting
- Set up monitoring and alerts
- Integrate with S3/MinIO for image storage
- Add more complex product queries (filtering, sorting)
- Implement pagination for product lists
