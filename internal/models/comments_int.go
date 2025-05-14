package models

import (
	"context"
	"fmt"
	"time"

	"github.com/joshuatakyi/shop/internal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (m *MongoClient) AddComment(ctx context.Context, comment Comments, userId string, productId primitive.ObjectID) error {
	if m.client == nil {
		return fmt.Errorf("MongoDB client is not initialized")
	}

	// product collection
	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)
	// check if the product exists
	filter := bson.M{"_id": productId}
	update := bson.M{
		"$push": bson.M{
			"comments": bson.M{
				"_id":       primitive.NewObjectID(),
				"productId": productId,
				"userId":    userId,
				"comment":   comment.Comment,
				"createdAt": time.Now(),
				"updatedAt": time.Now(),
			},
		},
	}

	// Update the product document by adding the comment to the comments array
	result, err := collectionRef.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to add comment: %v", err)
	}

	// Check if the product was found and updated
	if result.MatchedCount == 0 {
		return fmt.Errorf("product not found")
	}

	return nil
}

// GET COMMENTS FOR  A SPECIFIC PRODUCT BY ID
func (m *MongoClient) GetComments(ctx context.Context, productId primitive.ObjectID) ([]Comments, error) {

	if m.client == nil {
		return nil, fmt.Errorf("MongoDB client is not initialized")
	}

	// product collection
	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)
	// check if the product exists
	filter := bson.M{"_id": productId}
	var product Product
	err := collectionRef.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve product: %v", err)
	}

	return product.Comments, nil
}

func (m *MongoClient) DeleteComment(ctx context.Context, id primitive.ObjectID, userId string) error {
	if m.client == nil {
		return fmt.Errorf("MongoDB client is not initialized")
	}

	// product collection
	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)
	// check if the product exists
	filter := bson.M{"comments._id": id, "comments.userId": userId}
	//  the mongodb pull operator removes from an array all instances of a value or values that match a specified condition
	update := bson.M{
		"$pull": bson.M{
			"comments": bson.M{
				"_id": id,
			},
		},
	}

	result, err := collectionRef.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("comment not found")
	}

	return nil
}

func (m *MongoClient) UpdateComment(ctx context.Context, id primitive.ObjectID, userId string, comment Comments) error {
	if m.client == nil {
		return fmt.Errorf("MongoDB client is not initialized")
	}

	// product collection
	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)
	// check if the product exists
	filter := bson.M{"comments._id": id, "comments.userId": userId}
	update := bson.M{
		"$set": bson.M{
			"comments.$.comment":   comment.Comment,
			"comments.$.updatedAt": time.Now(),
		},
	}

	result, err := collectionRef.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update comment: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("comment not found")
	}

	return nil
}

func (m *MongoClient) GetCommentById(ctx context.Context, id primitive.ObjectID) (*Comments, error) {
	if m.client == nil {
		return nil, fmt.Errorf("MongoDB client is not initialized")
	}

	// product collection
	collectionRef := m.client.Database(internal.DbName).Collection(internal.ProductCollection)
	// check if the product exists
	filter := bson.M{"comments._id": id}
	var product Product
	err := collectionRef.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve comment: %v", err)
	}

	for _, comment := range product.Comments {
		if comment.ID == id {
			return &comment, nil
		}
	}

	return nil, fmt.Errorf("comment not found")
}
