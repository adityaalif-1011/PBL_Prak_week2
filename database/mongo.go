package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"api-students/config"
)

// NewMongo membuat koneksi ke MongoDB dan memastikan server bisa di-ping.
func NewMongo(ctx context.Context) (*mongo.Client, error) {
	uri := config.GetEnv("MONGO_URI", "mongodb://localhost:27017")

	opts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("connect mongodb: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}
	return client, nil
}

// MongoDB mengembalikan handle database yang dipakai aplikasi.
func MongoDB(client *mongo.Client) *mongo.Database {
	return client.Database(config.GetEnv("MONGO_DB", "api_students"))
}
