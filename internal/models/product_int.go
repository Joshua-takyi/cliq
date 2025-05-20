package models

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/joshuatakyi/shop/internal"
	"github.com/joshuatakyi/shop/internal/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (m *MongoClient) AddProduct(ctx context.Context, product Product) (string, error) {
	// Check if MongoDB client is initialized
	if m.client == nil {
		return "", fmt.Errorf("MongoDB client is not initialized")
	}

	if err := validate.Struct(product); err != nil {
		return "", fmt.Errorf("failed to vaildate product struct %v", err.Error())
	}
	// Set timestamps and default product flags
	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now
	product.IsAvailable = true

	// Set IsNew to true by default for all newly created products
	// This ensures new products will show up in "Latest Arrivals" section
	product.IsNew = true

	// Generate a new ObjectID if not provided
	if product.ID.IsZero() {
		product.ID = primitive.NewObjectID()
	}

	// If slug is empty, generate it from title, first part of description, and first category
	if product.Slug == "" {
		if len(product.Category) > 0 {
			product.Slug = helpers.GenerateSlug(product.Title, product.Description[:30], product.Category[0])
		} else {
			return "", fmt.Errorf("product category is required for slug generation")
		}
	}

	exist := helpers.DoesSlugAlreadyExist(product.Slug)
	if exist {
		return "", fmt.Errorf("slug already exists")
	}

	// Set default values for optional fields if they are zero values
	if product.Rating == 0 {
		product.Rating = 0
	}
	if product.ReviewCount == 0 {
		product.ReviewCount = 0
	}

	// Get collection reference
	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)

	// Insert the product into MongoDB
	_, err := collectionRef.InsertOne(ctx, product)
	if err != nil {
		return "", err
	}

	// Return the inserted ID as string
	return product.ID.Hex(), nil
}

func (m *MongoClient) ListProducts(ctx context.Context, page, limit int) ([]Product, error) {
	if m.client == nil {
		return nil, fmt.Errorf("MongoDB client is not initialized")
	}

	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)

	// Initialize products as an empty slice rather than nil
	products := []Product{}

	opts := options.Find()
	opts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	filter := bson.M{}
	cursor, err := collectionRef.Find(ctx, filter, opts)
	if err != nil {
		// Return detailed error for debugging
		return nil, fmt.Errorf("MongoDB Find failed: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var product Product
		if err := cursor.Decode(&product); err != nil {
			return nil, fmt.Errorf("failed to decode product: %v", err)
		}
		products = append(products, product)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	// Return products (will be an empty array if no products found)
	return products, nil
}

func (m *MongoClient) GetProductByID(ctx context.Context, id primitive.ObjectID) (*Product, error) {
	// Placeholder implementation
	if m.client == nil {
		return nil, fmt.Errorf("MongoDB client is not initialized")
	}

	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)
	filter := bson.M{"_id": id}
	var product Product
	err := collectionRef.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to retrieve product: %v", err)
	}

	return &product, nil
}

func (m *MongoClient) GetProductBySlug(ctx context.Context, slug string) (*Product, error) {
	// Placeholder implementation
	if m.client == nil {
		return nil, fmt.Errorf("MongoDB client is not initialized")
	}

	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)
	var product Product
	filter := bson.M{"slug": slug}
	err := collectionRef.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("product not found")
		} else {
			return nil, fmt.Errorf("failed to retrieve product: %v", err)
		}
	}

	return &product, nil
}

func (m *MongoClient) UpdateProduct(ctx context.Context, id primitive.ObjectID, product map[string]interface{}) (string, error) {
	if m.client == nil {
		return "", fmt.Errorf("MongoDB client is not initialized")
	}
	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)
	filter := bson.M{"_id": id}
	update := bson.M{"$set": product}
	_, err := collectionRef.UpdateOne(ctx, filter, update)
	if err != nil {
		return "", fmt.Errorf("failed to update product: %v", err)
	}

	return id.Hex(), nil
}

func (m *MongoClient) DeleteProduct(ctx context.Context, id primitive.ObjectID) error {
	if m.client == nil {
		return fmt.Errorf("MongoDB client is not initialized")
	}

	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)
	filter := bson.M{"_id": id}

	// Check if the product exists before attempting to delete
	err := collectionRef.FindOne(ctx, filter).Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("product not found")
		}
		return fmt.Errorf("failed to check product existence: %v", err)
	}

	// Perform the deletion
	result, err := collectionRef.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete product: %v", err)
	}

	// Ensure a product was actually deleted
	if result.DeletedCount == 0 {
		return fmt.Errorf("no product was deleted")
	}

	return nil
}

func (m *MongoClient) SearchProducts(ctx context.Context, query string, page, limit int) ([]Product, error) {
	if m.client == nil {
		return nil, fmt.Errorf("MongoDB client is not initialized")
	}

	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)

	// Initialize products as an empty slice rather than nil
	products := []Product{}

	opts := options.Find()
	opts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))

	filter := bson.M{
		"$or": []bson.M{
			// key and values and the "i" is for case insensitive
			{"title": bson.M{"$regex": query, "$options": "i"}},
			{"description": bson.M{"$regex": query, "$options": "i"}},
			{"category": bson.M{"$regex": query, "$options": "i"}},
			// loop through the tags and create a regex for each
			{"tags": bson.M{"$regex": query, "$options": "i"}},
			{"accessory_type": bson.M{"$regex": query, "$options": "i"}},
			{"models": bson.M{"$regex": query, "$options": "i"}},
			{"brand": bson.M{"$regex": query, "$options": "i"}},
		},
	}

	cursor, err := collectionRef.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("MongoDB Find failed: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var product Product
		if err := cursor.Decode(&product); err != nil {
			return nil, fmt.Errorf("failed to decode product: %v", err)
		}
		products = append(products, product)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	return products, nil
}

// func (m *MongoClient) DeleteImage(ctx context.Context, id primitive.ObjectID, imageId string) error {
// 	if m.client == nil {
// 		return fmt.Errorf("MongoDB client is not initialized")
// 	}

// 	// we are deleting  fro m cloudinary and the database
// 	// delete from cloudinary
// 	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)
// 	filter := bson.M{"_id": id}
// 	update := bson.M{"$pull": bson.M{"images": bson.M{"id": imageId}}}

// 	// cloudinary instance which connects to cloudinary and deletes the image
// 	ok, err := helpers.CloudinaryInstance(imageId)
// 	if err != nil {
// 		return fmt.Errorf("failed to delete image from cloudinary: %v", err)
// 	}

// 	if !ok {
// 		return fmt.Errorf("failed to delete image from cloudinary: image not found")
// 	}
// 	// delete from database
// 	_, err = collectionRef.UpdateOne(ctx, filter, update)
// 	if err != nil {
// 		return fmt.Errorf("failed to update product: %v", err)
// 	}
// 	// check if the image was deleted
// 	var product Product
// 	err = collectionRef.FindOne(ctx, filter).Decode(&product)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			return fmt.Errorf("product not found")
// 		}
// 		return fmt.Errorf("failed to retrieve product: %v", err)
// 	}

// 	return nil
// }

func (m *MongoClient) BuildQuery(ctx context.Context, filter map[string]interface{}) (bson.M, error) {
	if m.client == nil {
		return nil, fmt.Errorf("MongoDB client is not initialized")
	}

	// Initialize the query as an empty bson.M
	query := bson.M{}

	// Process each filter parameter
	for key, value := range filter {
		switch key {
		case "category":
			// For category, we want to find products that have the specified category in their category array
			if categories, ok := value.([]string); ok && len(categories) > 0 {
				query["category"] = bson.M{"$in": categories}
			} else if category, ok := value.(string); ok && category != "" {
				query["category"] = category
			}

		case "price_min", "price_max":
			// Handle price range filtering
			if priceFilter, ok := query["price"]; ok {
				// Price filter already exists, update it
				priceMap := priceFilter.(bson.M)
				if key == "price_min" {
					priceMap["$gte"] = value
				} else {
					priceMap["$lte"] = value
				}
			} else {
				// Create new price filter
				if key == "price_min" {
					query["price"] = bson.M{"$gte": value}
				} else {
					query["price"] = bson.M{"$lte": value}
				}
			}

		case "tags":
			// For tags, we want to find products that have any of the specified tags
			if tags, ok := value.([]string); ok && len(tags) > 0 {
				query["tags"] = bson.M{"$in": tags}
			} else if tag, ok := value.(string); ok && tag != "" {
				query["tags"] = tag
			}

		case "models":
			// For models, similar to tags
			if models, ok := value.([]string); ok && len(models) > 0 {
				query["models"] = bson.M{"$in": models}
			} else if model, ok := value.(string); ok && model != "" {
				query["models"] = model
			}

		// Note: is_new is handled in the general boolean filters case below

		case "search":
			// For search, we want to search in title, description, category, tags, etc.
			if searchTerm, ok := value.(string); ok && searchTerm != "" {
				// Log the search term for debugging
				fmt.Printf("Searching for: %s\n", searchTerm)

				// Split search term into words for more flexible matching
				searchWords := helpers.TokenizeSearchQuery(searchTerm)
				fmt.Printf("Tokenized search words: %v\n", searchWords)

				// Build a more sophisticated search query that handles multiple words
				if len(searchWords) > 0 {
					orConditions := []bson.M{}

					// For each word in the search query, create a condition
					for _, word := range searchWords {
						// Make the regex more flexible - ensure we escape any regex special characters in the word
						escapedWord := regexp.QuoteMeta(word)
						// Create a pattern that matches: word as a whole word (\bword\b) OR word anywhere (word)
						wordRegex := fmt.Sprintf("\\b%s\\b|%s", escapedWord, escapedWord)

						// Add conditions for each field we want to search in
						orConditions = append(orConditions,
							bson.M{"title": bson.M{"$regex": wordRegex, "$options": "i"}},
							bson.M{"description": bson.M{"$regex": wordRegex, "$options": "i"}},
							bson.M{"category": bson.M{"$regex": wordRegex, "$options": "i"}},
							bson.M{"tags": bson.M{"$regex": wordRegex, "$options": "i"}},
							bson.M{"models": bson.M{"$regex": wordRegex, "$options": "i"}},
							bson.M{"brand": bson.M{"$regex": wordRegex, "$options": "i"}},
						)
					}

					// Also include the full original search term for exact phrase matches
					escapedSearchTerm := regexp.QuoteMeta(searchTerm)
					orConditions = append(orConditions,
						// Exact phrase match with word boundaries
						bson.M{"title": bson.M{"$regex": fmt.Sprintf("\\b%s\\b", escapedSearchTerm), "$options": "i"}},
						bson.M{"description": bson.M{"$regex": fmt.Sprintf("\\b%s\\b", escapedSearchTerm), "$options": "i"}},
						bson.M{"category": bson.M{"$regex": fmt.Sprintf("\\b%s\\b", escapedSearchTerm), "$options": "i"}},
						bson.M{"tags": bson.M{"$regex": fmt.Sprintf("\\b%s\\b", escapedSearchTerm), "$options": "i"}},
						bson.M{"models": bson.M{"$regex": fmt.Sprintf("\\b%s\\b", escapedSearchTerm), "$options": "i"}},
						bson.M{"brand": bson.M{"$regex": fmt.Sprintf("\\b%s\\b", escapedSearchTerm), "$options": "i"}},

						// Also allow partial matches for the whole phrase
						bson.M{"title": bson.M{"$regex": escapedSearchTerm, "$options": "i"}},
						bson.M{"description": bson.M{"$regex": escapedSearchTerm, "$options": "i"}},
						bson.M{"category": bson.M{"$regex": escapedSearchTerm, "$options": "i"}},
						bson.M{"tags": bson.M{"$regex": escapedSearchTerm, "$options": "i"}},
						bson.M{"models": bson.M{"$regex": escapedSearchTerm, "$options": "i"}},
						bson.M{"brand": bson.M{"$regex": escapedSearchTerm, "$options": "i"}},
					)

					// Use $or to match any of these conditions
					query["$or"] = orConditions
				} else {
					// Fallback to simple search if no words were parsed
					// Escape the search term to avoid regex issues
					escapedSearchTerm := regexp.QuoteMeta(searchTerm)
					query["$or"] = []bson.M{
						{"title": bson.M{"$regex": escapedSearchTerm, "$options": "i"}},
						{"description": bson.M{"$regex": escapedSearchTerm, "$options": "i"}},
						{"category": bson.M{"$regex": escapedSearchTerm, "$options": "i"}},
						{"tags": bson.M{"$regex": escapedSearchTerm, "$options": "i"}},
						{"models": bson.M{"$regex": escapedSearchTerm, "$options": "i"}},
						{"brand": bson.M{"$regex": escapedSearchTerm, "$options": "i"}},
					}
				}
			}

		case "colors":
			// For colors
			if colors, ok := value.([]string); ok && len(colors) > 0 {
				query["colors"] = bson.M{"$in": colors}
			} else if color, ok := value.(string); ok && color != "" {
				query["colors"] = color
			}

		case "materials":
			// For materials
			if materials, ok := value.([]string); ok && len(materials) > 0 {
				query["materials"] = bson.M{"$in": materials}
			} else if material, ok := value.(string); ok && material != "" {
				query["materials"] = material
			}

		case "is_available", "is_new", "is_on_sale", "is_featured", "is_best_seller":
			// Boolean filters
			if boolValue, ok := value.(bool); ok {
				query[key] = boolValue
			}

		default:
			// For other fields, use direct equality match
			query[key] = value
		}
	}

	return query, nil
}

func (m *MongoClient) FilterProducts(ctx context.Context, filterParams map[string]interface{}, page, limit int) ([]Product, int, error) {
	// Check if MongoDB client is initialized
	if m.client == nil {
		return nil, 0, fmt.Errorf("MongoDB client is not initialized")
	}

	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)

	// Debug log the filter parameters
	fmt.Printf("Filter parameters before building query: %+v\n", filterParams)

	// Check if search parameter exists
	if searchParam, exists := filterParams["search"]; exists {
		fmt.Printf("Search parameter found: %v\n", searchParam)
	} else {
		fmt.Println("No search parameter found in filterParams")
	}

	// Build query from filter parameters
	query, err := m.BuildQuery(ctx, filterParams)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build query: %v", err)
	}

	// Debug log the built query
	fmt.Printf("Built MongoDB query: %+v\n", query)

	// Set up options for pagination and sorting
	opts := options.Find()
	opts.SetSort(bson.D{{Key: "createdAt", Value: -1}}) // Default sort by newest

	// Handle custom sorting if specified
	if sortField, ok := filterParams["sort_by"].(string); ok && sortField != "" {
		sortOrder := 1 // Default ascending
		if sortDir, ok := filterParams["sort_dir"].(string); ok && sortDir == "desc" {
			sortOrder = -1
		}
		opts.SetSort(bson.D{{Key: sortField, Value: sortOrder}})
	}

	// Apply pagination
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))

	// Get total count for pagination
	totalCount, err := collectionRef.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count documents: %v", err)
	}

	// Perform the query
	cursor, err := collectionRef.Find(ctx, query, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("MongoDB Find failed: %v", err)
	}
	defer cursor.Close(ctx)

	// Decode results
	products := []Product{}
	for cursor.Next(ctx) {
		var product Product
		if err := cursor.Decode(&product); err != nil {
			return nil, 0, fmt.Errorf("failed to decode product: %v", err)
		}
		products = append(products, product)
	}

	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("cursor error: %v", err)
	}

	return products, int(totalCount), nil
}

func (m *MongoClient) GetSimilarProducts(ctx context.Context, productId primitive.ObjectID) ([]Product, error) {
	// Check if MongoDB client is initialized
	if m.client == nil {
		return nil, fmt.Errorf("MongoDB client is not initialized")
	}

	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)

	// Find the product by ID
	var product Product
	if err := collectionRef.FindOne(ctx, bson.M{"_id": productId}).Decode(&product); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("product not found: %v", err)
		}
		return nil, fmt.Errorf("failed to find product: %v", err)
	}

	// Find similar products (e.g., by category or tags)
	// Build a query to find similar products based strictly on model and category
	query := bson.M{
		"$or": []bson.M{
			{"category": bson.M{"$in": product.Category}}, // Match products with similar categories
			{"models": bson.M{"$in": product.Models}},     // Match products with similar models
		},
		"_id": bson.M{"$ne": productId}, // Exclude the current product from the results
	}

	cursor, err := collectionRef.Find(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find similar products: %v", err)
	}
	defer cursor.Close(ctx)

	// Decode results
	var similarProducts []Product
	for cursor.Next(ctx) {
		var similarProduct Product
		if err := cursor.Decode(&similarProduct); err != nil {
			return nil, fmt.Errorf("failed to decode similar product: %v", err)
		}
		similarProducts = append(similarProducts, similarProduct)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	return similarProducts, nil
}
