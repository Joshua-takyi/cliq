package models

import (
	"context"
	"fmt"
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
	// Set timestamps
	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now
	product.IsAvailable = true

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
