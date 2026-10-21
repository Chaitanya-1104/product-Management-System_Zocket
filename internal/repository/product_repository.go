package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lib/pq"

	"product-management-system/internal/models"
)

type ProductRepository struct {
	db *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(ctx context.Context, product models.ProductInput) (int, error) {
	var id int
	query := `
		INSERT INTO products (user_id, product_name, product_description, product_images, product_price, compressed_product_images)
		VALUES ($1, $2, $3, $4, $5, '{}')
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		product.UserID,
		product.ProductName,
		product.ProductDescription,
		pq.Array(product.ProductImages),
		product.ProductPrice,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert product: %w", err)
	}
	return id, nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id string) (models.Product, error) {
	var product models.Product
	query := `
		SELECT id, user_id, product_name, product_description, product_images, product_price, compressed_product_images
		FROM products
		WHERE id = $1
	`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&product.ID,
		&product.UserID,
		&product.ProductName,
		&product.ProductDescription,
		pq.Array(&product.ProductImages),
		&product.ProductPrice,
		pq.Array(&product.CompressedProductImages),
	)
	if err != nil {
		return models.Product{}, fmt.Errorf("fetch product by id: %w", err)
	}
	return product, nil
}

func (r *ProductRepository) UpdateCompressedImages(ctx context.Context, productID int, images []string) error {
	query := `UPDATE products SET compressed_product_images = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, images, productID)
	if err != nil {
		return fmt.Errorf("update compressed images: %w", err)
	}
	return nil
}
