package config

import "os"

type Config struct {
	DatabaseURL  string
	RedisAddr    string
	RabbitURL    string
	QueueName    string
	StorageType  string // "database" (Base64 in DB) or "s3" (object storage)
	StorageURL   string // Base URL for serving stored images (e.g. "http://localhost:8080/images/")
}

func Load() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgresql://neondb_owner:npg_CR36HPBfXQjy@ep-wandering-voice-ax79in50-pooler.c-4.us-east-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		RabbitURL:   getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		QueueName:   getEnv("QUEUE_NAME", "image_processing_queue"),
		StorageType: getEnv("STORAGE_TYPE", "database"),
		StorageURL:  getEnv("STORAGE_URL", "http://localhost:8080/images/"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
