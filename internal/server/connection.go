package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Client *mongo.Client
)
func InitializeConnection() error {
	// Try to load from current directory first
	if err := godotenv.Load(".env.local"); err != nil {
		// If that fails, try loading from the project root
		workDir, err := os.Getwd()
		if err == nil {
			// Try to find .env.local relative to the working directory
			envPath := filepath.Join(workDir, ".env.local")
			godotenv.Load(envPath)
		}
		fmt.Printf("Warning: Could not load .env.local file. Using environment variables directly.\n")
	}
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		return fmt.Errorf("MONGODB_URI environment variable is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	Client, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	fmt.Println("Connected to MongoDB")

	return nil
}

func Disconnect() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := Client.Disconnect(ctx); err != nil {
		fmt.Printf("failed to disconnect from MongoDB: %v\n", err)
	} else {
		fmt.Println("Disconnected from MongoDB")
	}
}