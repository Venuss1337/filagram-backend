package database

import (
	"context"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"os"
	"time"
)

func Connect()  (*mongo.Client, error) {
	err := godotenv.Load()
	if err != nil { return nil, err }

	clientOptions := options.Client()
	clientOptions.ApplyURI(os.Getenv("DATABASE_URL"))
	clientOptions.SetTimeout(10 * time.Second)

	client, err := mongo.Connect(clientOptions)
	if err != nil { return &mongo.Client{}, err }

	// test the connection with database
	if err := client.Ping(context.Background(), nil); err != nil { return &mongo.Client{}, err }

	return client, nil
}

