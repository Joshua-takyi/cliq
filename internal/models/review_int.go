package models

import "go.mongodb.org/mongo-driver/bson/primitive"

func (m *MongoClient) AddReview(review Review) error {
	// Placeholder implementation
	return nil
}

func (m *MongoClient) GetProductReviews(productID primitive.ObjectID) ([]Review, error) {
	// Placeholder implementation
	return nil, nil
}
