package models

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/joshuatakyi/shop/internal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (m *MongoClient) AddToCart(ctx context.Context, userID primitive.ObjectID, item CartItem) error {
	if m.client == nil {
		return fmt.Errorf("MongoDB client is not initialized %v", m.client)
	}
	dbRef := m.client.Database(internal.DbName)
	cartColRef := dbRef.Collection(internal.CartCollection)
	if err := validate.Struct(item); err != nil {
		return fmt.Errorf("validation error: %v", err)
	}

	// check if product exists
	productDb := dbRef.Collection(internal.ProductCollection)
	var product Product
	err := productDb.FindOne(ctx, bson.M{"_id": item.ProductID}).Decode(&product)
	if err != nil {
		return fmt.Errorf("failed to find product: %v", err)
	}

	// check if quantity is greater than 0 and enough stock
	if item.Quantity <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}
	if item.Quantity > product.Stock {
		return fmt.Errorf("not enough stock, requested: %d, available: %d", item.Quantity, product.Stock)
	}

	// Calculate item's price and total price
	item.Price = product.Price
	if product.Discount > 0 {
		// Calculate discounted price and round to 2 decimal places
		discountAmount := product.Price * (product.Discount / 100)
		item.Price = math.Round((product.Price-discountAmount)*100) / 100
	}
	item.TotalPrice = math.Round(item.Price*float64(item.Quantity)*100) / 100

	// Start a session for transaction
	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %v", err)
	}
	defer session.EndSession(ctx)

	// Use transaction to ensure atomicity
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Find user's cart or create new one
		var cart Cart
		err := cartColRef.FindOne(sessCtx, bson.M{"user_id": userID}).Decode(&cart)

		// If cart doesn't exist, create a new one
		if err != nil {
			if err == mongo.ErrNoDocuments {
				cart = Cart{
					UserID:      userID,
					Items:       []CartItem{item},
					TotalAmount: item.TotalPrice, // This is correct as it's the only item
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				_, err = cartColRef.InsertOne(sessCtx, cart)
				return nil, err
			}
			return nil, fmt.Errorf("error finding cart: %v", err)
		}

		// Check if item already exists in cart
		itemExists := false
		for i, cartItem := range cart.Items {
			if cartItem.ProductID == item.ProductID && cartItem.Color == item.Color {
				// Update item quantity and total price
				cart.Items[i].Quantity += item.Quantity
				cart.Items[i].TotalPrice = math.Round(cart.Items[i].Price*float64(cart.Items[i].Quantity)*100) / 100 // Added proper rounding
				itemExists = true
				break
			}
		}

		// If item doesn't exist in cart, add it
		if !itemExists {
			cart.Items = append(cart.Items, item)
		}

		// Update total amount with proper rounding
		totalAmount := 0.0
		for _, cartItem := range cart.Items {
			totalAmount += cartItem.TotalPrice
		}
		cart.TotalAmount = math.Round(totalAmount*100) / 100 // Ensure proper rounding to 2 decimal places
		cart.UpdatedAt = time.Now()

		// Update cart in database - ensure we're using the correct field name
		updateResult, err := cartColRef.UpdateOne(
			sessCtx,
			bson.M{"user_id": userID},
			bson.M{"$set": bson.M{
				"items":        cart.Items,
				"total_amount": cart.TotalAmount, // Using consistent field name
				"updated_at":   cart.UpdatedAt,
			}},
		)
		if err != nil || updateResult.ModifiedCount == 0 {
			return nil, fmt.Errorf("failed to update cart: %v", err)
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %v", err)
	}

	return nil
}

func (m *MongoClient) UpdateCartItem(ctx context.Context, userID primitive.ObjectID, item CartItem, actions CartActions) error {
	// Placeholder implementation
	if m.client == nil {
		return fmt.Errorf("MongoDB client is not initialized %v", m.client)
	}

	dbRef := m.client.Database(internal.DbName)
	cartColRef := dbRef.Collection(internal.CartCollection)
	if err := validate.Struct(item); err != nil {
		return fmt.Errorf("validation error: %v", err)
	}

	// check if product exists
	productDb := dbRef.Collection(internal.ProductCollection)
	var product Product
	filter := bson.M{"_id": item.ProductID}
	if err := productDb.FindOne(ctx, filter).Decode(&product); err != nil {
		return fmt.Errorf("failed to find product: %v", err)
	}

	// Start a session for transaction
	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %v", err)
	}
	defer session.EndSession(ctx)

	// Use transaction to ensure atomicity
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Find user's cart
		var cart Cart
		filter := bson.M{"user_id": userID}
		err := cartColRef.FindOne(sessCtx, filter).Decode(&cart)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				// If cart doesn't exist, create a new one with the item
				// Set initial quantity based on the action
				if actions.Increment {
					item.Quantity = 1
				}
				// Calculate item's price and total price
				item.Price = product.Price
				if product.Discount > 0 {
					// Calculate discounted price and round to 2 decimal places
					discountAmount := product.Price * (product.Discount / 100)
					item.Price = math.Round((product.Price-discountAmount)*100) / 100
				}
				item.TotalPrice = math.Round(item.Price*float64(item.Quantity)*100) / 100

				cart = Cart{
					UserID:      userID,
					Items:       []CartItem{item},
					TotalAmount: item.TotalPrice,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				_, err = cartColRef.InsertOne(sessCtx, cart)
				return nil, err
			}
			return nil, fmt.Errorf("error finding cart: %v", err)
		}

		// Find the item in the cart
		itemFound := false
		for i, cartItem := range cart.Items {
			if cartItem.ProductID == item.ProductID && cartItem.Color == item.Color {
				// Handle increment/decrement actions
				if actions.Increment {
					cart.Items[i].Quantity++
				} else if actions.Decrement {
					cart.Items[i].Quantity--
					// Remove item if quantity becomes 0
					if cart.Items[i].Quantity <= 0 {
						// Remove this item from cart
						cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
						itemFound = true
						break
					}
				} else {
					// Direct quantity update
					cart.Items[i].Quantity = item.Quantity
				}

				// Check if requested quantity is available in stock
				if cart.Items[i].Quantity > product.Stock {
					return nil, fmt.Errorf("not enough stock, requested: %d, available: %d",
						cart.Items[i].Quantity, product.Stock)
				}

				// Update the total price for this item with proper rounding
				cart.Items[i].TotalPrice = math.Round(cart.Items[i].Price*float64(cart.Items[i].Quantity)*100) / 100
				itemFound = true
				break
			}
		}

		// Add the item to cart if not found and we're incrementing
		if !itemFound {
			if actions.Increment {
				// Initialize with quantity 1 for increment action
				item.Quantity = 1

				// Calculate item's price and total price
				item.Price = product.Price
				if product.Discount > 0 {
					// Calculate discounted price and round to 2 decimal places
					discountAmount := product.Price * (product.Discount / 100)
					item.Price = math.Round((product.Price-discountAmount)*100) / 100
				}
				item.TotalPrice = math.Round(item.Price*float64(item.Quantity)*100) / 100

				cart.Items = append(cart.Items, item)
				itemFound = true
			} else {
				return nil, fmt.Errorf("item not found in cart")
			}
		}

		// Recalculate cart total with proper rounding
		totalAmount := 0.0
		for _, cartItem := range cart.Items {
			totalAmount += cartItem.TotalPrice
		}
		cart.TotalAmount = math.Round(totalAmount*100) / 100
		cart.UpdatedAt = time.Now()

		// Update cart in database with consistent field names
		_, err = cartColRef.UpdateOne(
			sessCtx,
			bson.M{"user_id": userID},
			bson.M{
				"$set": bson.M{
					"items":        cart.Items,
					"total_amount": cart.TotalAmount, // Using consistent field name
					"updated_at":   cart.UpdatedAt,
				},
				// Remove the old field if it exists
				"$unset": bson.M{
					"totalamount": "", // Remove the incorrect field name
				},
			},
		)

		return nil, err
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %v", err)
	}

	return nil
}

func (m *MongoClient) RemoveCartItem(ctx context.Context, userID primitive.ObjectID, productID primitive.ObjectID) error {
	if m.client == nil {
		return fmt.Errorf("MongoDB client is not initialized")
	}

	cartColRef := m.client.Database(internal.DbName).Collection(internal.CartCollection)

	// Start a session for transaction
	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %v", err)
	}
	defer session.EndSession(ctx)

	// Use transaction to ensure atomicity
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Find user's cart
		var cart Cart
		filter := bson.M{"user_id": userID}
		err := cartColRef.FindOne(sessCtx, filter).Decode(&cart)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, fmt.Errorf("cart not found")
			}
			return nil, fmt.Errorf("error finding cart: %v", err)
		}

		// Find and remove the item from cart items
		foundIndex := -1
		for i, item := range cart.Items {
			// Check both ProductID and productid field (to handle database inconsistency)
			if item.ProductID == productID {
				foundIndex = i
				break
			}
		}

		// If item not found by ProductID, check if we need to use raw BSON to inspect the document
		if foundIndex == -1 {
			// Log the items for debugging
			for i, item := range cart.Items {
				// Log the item details to help diagnose the issue
				fmt.Printf("Item %d: ProductID=%v, Quantity=%d, Color=%s\n",
					i, item.ProductID, item.Quantity, item.Color)
			}

			// Get the raw document to check field names
			var rawCart bson.M
			if err := cartColRef.FindOne(sessCtx, filter).Decode(&rawCart); err == nil {
				if items, ok := rawCart["items"].([]interface{}); ok {
					for i, rawItem := range items {
						if item, ok := rawItem.(bson.M); ok {
							// Check if the item uses "productid" instead of "product_id"
							if pid, exists := item["productid"]; exists {
								if pidObj, ok := pid.(primitive.ObjectID); ok && pidObj == productID {
									foundIndex = i
									break
								}
							}
						}
					}
				}
			}

			// If still not found, return error
			if foundIndex == -1 {
				return nil, fmt.Errorf("item not found in cart")
			}
		}

		// Remove item from the slice
		cart.Items = append(cart.Items[:foundIndex], cart.Items[foundIndex+1:]...)

		// Recalculate cart total with proper rounding
		totalAmount := 0.0
		for _, item := range cart.Items {
			totalAmount += item.TotalPrice
		}
		cart.TotalAmount = math.Round(totalAmount*100) / 100 // Ensure proper rounding to 2 decimal places
		cart.UpdatedAt = time.Now()

		// Update or delete cart based on whether items remain
		if len(cart.Items) == 0 {
			// If cart is empty, delete it
			_, err = cartColRef.DeleteOne(sessCtx, filter)
		} else {
			// Otherwise update it with consistent field names
			_, err = cartColRef.UpdateOne(
				sessCtx,
				filter,
				bson.M{
					"$set": bson.M{
						"items":        cart.Items,
						"total_amount": cart.TotalAmount, // Using consistent field name
						"updated_at":   cart.UpdatedAt,
					},
					"$unset": bson.M{
						"totalamount": "", // Remove the incorrect field name if it exists
					},
				},
			)
		}

		return nil, err
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %v", err)
	}

	return nil
}

func (m *MongoClient) ClearCart(ctx context.Context, userID primitive.ObjectID) error {
	if m.client == nil {
		return fmt.Errorf("MongoDB client is not initialized")
	}
	collectionRef := m.client.Database(internal.DbName).Collection(internal.CartCollection)
	filter := bson.M{"user_id": userID}

	// get the items in the cart
	var cart Cart

	err := collectionRef.FindOne(ctx, filter).Decode(&cart)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("cart not found")
		}
		return fmt.Errorf("failed to retrieve cart: %v", err)
	}
	// Check if the userID matches the cart's user_id
	if cart.UserID != userID {
		return fmt.Errorf("user ID does not match the cart's user ID %v", userID)
	}

	// Clear the cart - ensure we're using the correct field name
	_, err = collectionRef.UpdateOne(ctx, filter, bson.M{
		"$set": bson.M{
			"items":        []CartItem{},
			"total_amount": 0, // Using consistent field name
			"updated_at":   time.Now(),
		},
		"$unset": bson.M{
			"totalamount": "", // Remove the incorrect field name if it exists
		},
	})
	if err != nil {
		return fmt.Errorf("failed to clear cart: %v", err)
	}
	return nil
}

func (m *MongoClient) GetUserCart(ctx context.Context, userID primitive.ObjectID) (*Cart, error) {
	if m.client == nil {
		return nil, fmt.Errorf("MongoDB client is not initialized")
	}

	cartColRef := m.client.Database(internal.DbName).Collection(internal.CartCollection)
	filter := bson.M{"user_id": userID}

	var cart Cart
	err := cartColRef.FindOne(ctx, filter).Decode(&cart)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return an empty cart if none exists
			return &Cart{
				UserID:      userID,
				Items:       []CartItem{},
				TotalAmount: 0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to retrieve cart: %v", err)
	}

	// Check if the userID matches the cart's user_id
	if cart.UserID != userID {
		return nil, fmt.Errorf("user ID does not match the cart's user ID")
	}

	return &cart, nil
}
