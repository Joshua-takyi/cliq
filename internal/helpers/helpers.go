package helpers

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader" // Import the correct v2 uploader package for Cloudinary
	"github.com/joho/godotenv"
	"github.com/joshuatakyi/shop/internal"
	"github.com/joshuatakyi/shop/internal/server"
	"go.mongodb.org/mongo-driver/bson"
)

func DoesSlugAlreadyExist(slug string) bool {
	// Check if the slug already exists in the database
	Doc, err := server.Client.Database(internal.DbName).Collection(internal.ProductCollection).CountDocuments(context.Background(), bson.M{"slug": slug})
	if err != nil {
		return false
	}
	// If the count is greater than 0, the slug already exists
	return Doc > 0
}

func GenerateSlug(title, description, category string) string {
	t := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := fmt.Sprintf("%s-%s-%s", title, description, category)
	slug = t.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	slug = strings.ToLower(slug)
	return slug
}

func CloudinaryInstance(imageId string) (bool, error) {
	ctx := context.Background()
	if err := godotenv.Load(".env.local"); err != nil {
		fmt.Println("Error loading .env file")
	}

	CLOUDINARY_NAME := os.Getenv("CLOUDINARY_NAME")
	if CLOUDINARY_NAME == "" {
		fmt.Println("CLOUDINARY_NAME not set in .env file")
		return false, fmt.Errorf("CLOUDINARY_NAME not set in .env file")
	}
	CLOUDINARY_API_KEY := os.Getenv("CLOUDINARY_API_KEY")
	if CLOUDINARY_API_KEY == "" {
		fmt.Println("CLOUDINARY_API_KEY not set in .env file")
		return false, fmt.Errorf("CLOUDINARY_API_KEY not set in .env file")
	}
	CLOUDINARY_API_SECRET := os.Getenv("CLOUDINARY_API_SECRET")
	if CLOUDINARY_API_SECRET == "" {
		fmt.Println("CLOUDINARY_API_SECRET not set in .env file")
		return false, fmt.Errorf("CLOUDINARY_API_SECRET not set in .env file")
	}
	// Add your Cloudinary product environment credentials.

	cld, _ := cloudinary.NewFromParams(CLOUDINARY_NAME, CLOUDINARY_API_KEY, CLOUDINARY_API_SECRET)

	resp, err := cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     imageId,
		ResourceType: "image",
	})

	if err != nil {
		return false, fmt.Errorf("failed to destroy image: %w", err)
	}

	if resp.Result != "ok" {
		return false, fmt.Errorf("failed to destroy image: %w", err)
	}
	return true, nil
}

// TokenizeSearchQuery splits a search query into individual tokens/words
// to improve search relevance by matching individual terms
func TokenizeSearchQuery(query string) []string {
	// Convert query to lowercase for case-insensitive matching
	query = strings.ToLower(query)

	// Remove special characters and extra spaces
	// Keep letters, numbers and spaces
	reg := regexp.MustCompile(`[^a-z0-9\s]`)
	query = reg.ReplaceAllString(query, " ")

	// Split by whitespace
	words := strings.Fields(query)

	// Filter out words that are too short (less than 2 characters)
	filtered := []string{}
	for _, word := range words {
		if len(word) >= 2 {
			filtered = append(filtered, word)
		}
	}

	return filtered
}
