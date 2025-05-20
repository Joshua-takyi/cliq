package database

import (
	"github.com/joshuatakyi/shop/internal/models"
	"github.com/joshuatakyi/shop/internal/server"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func AddToCart(c echo.Context) error {
	ctx := c.Request().Context()
	var cart models.CartItem

	// Retrieve userId from context
	userId, ok := c.Get("userId").(string)
	if !ok {
		c.Logger().Error("Failed to retrieve userId from context")
		return c.JSON(500, echo.Map{
			"message": "Internal server error",
		})
	}

	// Bind the request body to the cart struct
	if err := c.Bind(&cart); err != nil {
		c.Logger().Error("Failed to bind cart data: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid input",
			"error":   err.Error(),
		})
	}

	shopRepo := models.NewMongoClient(server.Client)

	// Convert userId to ObjectID
	convertedId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		c.Logger().Error("Failed to convert userId to ObjectID: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid userId",
			"error":   err.Error(),
		})
	}

	err = shopRepo.AddToCart(ctx, convertedId, cart)
	if err != nil {
		c.Logger().Error("Failed to add item to cart: ", err)
		return c.JSON(500, echo.Map{
			"message": "Failed to add item to cart",
			"error":   err.Error(),
		})
	}

	return c.JSON(201, echo.Map{
		"message": "Item added to cart successfully",
	})
}

func UpdateCart(c echo.Context) error {
	userId, ok := c.Get("userId").(string)
	if !ok {
		c.Logger().Error("Failed to retrieve userId from context")
		return c.JSON(500, echo.Map{
			"message": "Internal server error",
		})
	}
	if userId == "" {
		c.Logger().Error("userId is empty")
		return c.JSON(400, echo.Map{
			"message": "Unauthorized",
		})
	}

	ctx := c.Request().Context()

	// Create a struct to capture both cart item and action
	type CartUpdateRequest struct {
		models.CartItem
		Action     string `json:"action"`
		Product_Id string `json:"product_Id"` // Add field to capture the frontend's camelCase version
	}

	var updateReq CartUpdateRequest
	if err := c.Bind(&updateReq); err != nil {
		c.Logger().Error("Failed to bind cart data: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid input",
			"error":   err.Error(),
		})
	}

	// Log the request for debugging
	c.Logger().Info("Request data: ", updateReq)

	// Convert the product_Id to ProductID if needed
	if updateReq.CartItem.ProductID.IsZero() && updateReq.Product_Id != "" {
		productID, err := primitive.ObjectIDFromHex(updateReq.Product_Id)
		if err != nil {
			c.Logger().Error("Failed to convert product_Id to ObjectID: ", err)
			return c.JSON(400, echo.Map{
				"message": "Invalid product_Id",
				"error":   err.Error(),
			})
		}
		updateReq.CartItem.ProductID = productID
	}

	shopRepo := models.NewMongoClient(server.Client)

	// Convert userId to ObjectID
	convertedId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		c.Logger().Error("Failed to convert userId to ObjectID: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid userId",
			"error":   err.Error(),
		})
	}

	// Define action - handle both the query param and body, with body taking precedence
	// Also handle the typo "increament" vs "increment"
	var actions models.CartActions
	action := c.QueryParam("action")

	// If action is in the request body, use that instead
	if updateReq.Action != "" {
		action = updateReq.Action
	}

	// Handle the action (including potential typo "increament")
	if action == "increment" || action == "increament" {
		actions.Increment = true
	} else if action == "decrement" {
		actions.Decrement = true
	}

	// This call will properly recalculate the total amount as the sum of item total prices
	err = shopRepo.UpdateCartItem(ctx, convertedId, updateReq.CartItem, actions)
	if err != nil {
		c.Logger().Error("Failed to update cart: ", err)
		return c.JSON(500, echo.Map{
			"message": "Failed to update cart",
			"error":   err.Error(),
		})
	}

	return c.JSON(200, echo.Map{
		"message": "Cart updated successfully",
	})
}

func GetUserCart(c echo.Context) error {
	userId, ok := c.Get("userId").(string)
	if !ok || userId == "" {
		c.Logger().Error("Failed to retrieve userId from context")
		return c.JSON(401, echo.Map{
			"message": "Unauthorized",
		})
	}

	ctx := c.Request().Context()
	shopRepo := models.NewMongoClient(server.Client)

	// Convert userId to ObjectID
	convertedId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		c.Logger().Error("Failed to convert userId to ObjectID: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid userId",
			"error":   err.Error(),
		})
	}

	cart, err := shopRepo.GetUserCart(ctx, convertedId)
	if err != nil {
		c.Logger().Error("Failed to get user cart: ", err)
		return c.JSON(500, echo.Map{
			"message": "Failed to retrieve cart",
			"error":   err.Error(),
		})
	}

	// Ensure we're not returning a cart with a zero ID
	if cart.ID.IsZero() {
		// Generate a new ID for the cart if it's zero (temporary ID for frontend)
		cart.ID = primitive.NewObjectID()
	}

	return c.JSON(200, cart)
}

func ClearCart(c echo.Context) error {
	userId, ok := c.Get("userId").(string)
	if !ok || userId == "" {
		c.Logger().Error("Failed to retrieve userId from context")
		return c.JSON(401, echo.Map{
			"message": "Unauthorized",
		})
	}

	ctx := c.Request().Context()
	shopRepo := models.NewMongoClient(server.Client)

	userObjectId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		c.Logger().Error("Failed to convert userId to ObjectID: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid userId",
		})
	}

	err = shopRepo.ClearCart(ctx, userObjectId)
	if err != nil {
		c.Logger().Error("Failed to clear cart: ", err)
		if err.Error() == "no documents in result" {
			return c.JSON(404, echo.Map{
				"message": "Cart not found",
			})
		}
		return c.JSON(500, echo.Map{
			"message": "Failed to clear cart",
			"error":   err.Error(),
		})
	}

	return c.JSON(200, echo.Map{
		"message": "Cart cleared successfully",
	})
}

func RemoveCartItem(c echo.Context) error {
	ctx := c.Request().Context()
	userId, ok := c.Get("userId").(string)
	var requestBody struct {
		Id string `json:"id"`
	}

	// Check user authentication first
	if !ok || userId == "" {
		c.Logger().Error("Failed to retrieve userId from context")
		return c.JSON(401, echo.Map{
			"message": "Unauthorized",
		})
	}

	if err := c.Bind(&requestBody); err != nil {
		c.Logger().Error("Failed to bind request body: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid input",
			"error":   err.Error(),
		})
	}

	if requestBody.Id == "" {
		c.Logger().Error("Cart item ID is missing in the request body")
		return c.JSON(400, echo.Map{
			"message": "Cart item ID is required",
		})
	}

	// Convert cart item ID from request
	cartItemObjectId, err := primitive.ObjectIDFromHex(requestBody.Id)
	if err != nil {
		c.Logger().Error("Failed to convert cart item ID: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid cart item ID format",
			"error":   err.Error(),
		})
	}

	// Convert user ID
	userObjectId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		c.Logger().Error("Failed to convert user ID: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid user ID format",
			"error":   err.Error(),
		})
	}

	shopRepo := models.NewMongoClient(server.Client)

	// Call the model's RemoveCartItem method
	err = shopRepo.RemoveCartItem(ctx, userObjectId, cartItemObjectId)
	if err != nil {
		c.Logger().Error("Failed to remove item from cart: ", err)
		return c.JSON(500, echo.Map{
			"message": "Failed to remove item from cart",
			"error":   err.Error(),
		})
	}

	return c.JSON(200, echo.Map{
		"message": "Item removed from cart successfully",
	})
}
