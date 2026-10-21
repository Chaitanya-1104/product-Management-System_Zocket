package models

type Product struct {
	ID                      int      `json:"id"`
	UserID                  int      `json:"user_id"`
	ProductName             string   `json:"product_name"`
	ProductDescription      string   `json:"product_description"`
	ProductImages           []string `json:"product_images"`
	ProductPrice            float64  `json:"product_price"`
	CompressedProductImages []string `json:"compressed_product_images"`
}

type ProductInput struct {
	UserID             int      `json:"user_id" binding:"required"`
	ProductName        string   `json:"product_name" binding:"required"`
	ProductDescription string   `json:"product_description" binding:"required"`
	ProductImages      []string `json:"product_images"`
	ProductPrice       float64  `json:"product_price" binding:"required"`
}

type ImageProcessingMessage struct {
	ProductID int      `json:"product_id"`
	ImageURLs []string `json:"image_urls"`
}
