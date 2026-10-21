package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"

	"product-management-system/internal/models"
	"product-management-system/internal/repository"
)

type ProductService struct {
	repository *repository.ProductRepository
	cache      *redis.Client
	queue      QueuePublisher
}

type QueuePublisher interface {
	PublishJSON(payload interface{}) error
}

func NewProductService(db *pgxpool.Pool, cache *redis.Client, queue QueuePublisher) *ProductService {
	return &ProductService{
		repository: repository.NewProductRepository(db),
		cache:      cache,
		queue:      queue,
	}
}

func (s *ProductService) CreateProduct(ctx context.Context, input models.ProductInput) (int, error) {
	id, err := s.repository.Create(ctx, input)
	if err != nil {
		return 0, err
	}

	msg := models.ImageProcessingMessage{
		ProductID: id,
		ImageURLs: input.ProductImages,
	}
	if err := s.queue.PublishJSON(msg); err != nil {
		return 0, fmt.Errorf("publish image processing request: %w", err)
	}

	return id, nil
}

func (s *ProductService) GetProductByID(ctx context.Context, id string) (models.Product, error) {
	cachedData, err := s.cache.Get(ctx, id).Result()
	if err == nil {
		var product models.Product
		if err := json.Unmarshal([]byte(cachedData), &product); err == nil {
			return product, nil
		}
	}

	product, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return models.Product{}, err
	}

	productJSON, _ := json.Marshal(product)
	if err := s.cache.Set(ctx, id, productJSON, 10*time.Minute).Err(); err != nil {
		// Log but don't fail if cache set fails
		_ = err
	}
	return product, nil
}
