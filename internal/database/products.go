package database

import (
	"fmt"
	"time"

	"github.com/joshuatakyi/shop/internal/models"
	"github.com/joshuatakyi/shop/internal/server"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateProduct(c echo.Context) error {
	// Get role from context set by AuthMiddleware
	// The middleware.AuthMiddleware() function set this when validating the JWT token
	role, ok := c.Get("role").(string)
	if !ok {
		// If role is not found in context, log the error and return 401 Unauthorized
		c.Logger().Error("Failed to retrieve role from context - user might not be authenticated properly")
		return c.JSON(401, echo.Map{
			"success": false,
			"message": "Authentication required",
		})
	}

	ctx := c.Request().Context()

	// Verify user has admin role
	if role != "admin" {
		c.Logger().Error("Unauthorized access attempt - user has role: " + role)
		return c.JSON(403, echo.Map{
			"success": false,
			"message": "Forbidden: You do not have permission to perform this action",
		})
	}

	// Bind the request body directly to the product struct
	// We'll remove the double binding that was causing EOF errors
	var product models.Product
	if err := c.Bind(&product); err != nil {
		c.Logger().Error("Failed to bind product data to struct: ", err)
		return c.JSON(400, echo.Map{
			"success": false,
			"message": "Invalid input structure",
			"error":   err.Error(),
		})
	}

	// Log the received product data for debugging
	c.Logger().Debug("Received product data: ", product)

	// No need to set slug here as the AddProduct method will handle it

	shopRepo := models.NewMongoClient(server.Client)
	id, err := shopRepo.AddProduct(ctx, product)
	if err != nil {
		c.Logger().Error("Failed to add product: ", err)
		return c.JSON(500, echo.Map{
			"success": false,
			"message": "Failed to create product",
			"error":   err.Error(),
		})
	}

	// Return response in the format expected by the frontend ApiResponse interface
	return c.JSON(201, echo.Map{
		"success": true,
		"message": "Product added successfully",
		"product": map[string]interface{}{
			"_id": id, // Include the ID in a product object to match ApiResponse<ProductProps>
		},
	})
}

// GETPRODUCTS RETRIEVES A LIST OF PRODUCTS WITH OPTIONAL FILTERING
func ListProducts(c echo.Context) error {
	// calculate the time it took for the data to get fetched
	start := time.Now()
	ctx := c.Request().Context()
	page := 1
	limit := 10

	// Parse query parameters
	if pageParam := c.QueryParam("page"); pageParam != "" {
		fmt.Sscanf(pageParam, "%d", &page)
		if page < 1 {
			page = 1
		}
	}

	if limitParam := c.QueryParam("limit"); limitParam != "" {
		fmt.Sscanf(limitParam, "%d", &limit)
		if limit < 1 {
			limit = 10
		} else if limit > 100 {
			limit = 100 // Cap maximum limit
		}
	}

	// Get products from database
	shopRepo := models.NewMongoClient(server.Client)
	products, err := shopRepo.ListProducts(ctx, page, limit)
	if err != nil {
		// Enhanced error logging with more details to help troubleshoot the issue
		c.Logger().Errorf("Failed to retrieve products: %v", err)

		// Check if the error is related to the accessory_type decoding issue
		if err.Error() == "failed to decode product: error decoding key accessory_type: SliceDecodeValue can only decode a string into a byte array, got string" {
			// This is likely a schema mismatch issue in the database
			c.Logger().Error("Schema mismatch detected with accessory_type field. Check product model definition.")
			return c.JSON(500, echo.Map{
				"message": "Data schema error: Issue with accessory_type field format",
				"error":   "There is a mismatch between the stored data type and the expected type",
			})
		}

		return c.JSON(500, echo.Map{
			"message": "Failed to retrieve products",
			"error":   err.Error(), // Including the error message helps with debugging
		})
	}

	end := time.Now()
	count := len(products)

	var message string
	if count > 0 {
		message = "Products retrieved successfully"
	} else {
		message = "No products found"
	}

	return c.JSON(200, echo.Map{
		"message":  message,
		"products": products,
		"count":    count,
		"duration": end.Sub(start).String(),
	})
}

func GetProductByID(c echo.Context) error {
	ctx := c.Request().Context()
	paramsId := c.Param("id")
	if paramsId == "" {
		return c.JSON(400, echo.Map{
			"message": "Product ID is required",
		})
	}
	shopRepo := models.NewMongoClient(server.Client)
	convertedId, err := primitive.ObjectIDFromHex(paramsId)
	if err != nil {
		c.Logger().Error("Failed to convert product ID: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid product ID format",
			"error":   err.Error(),
		})
	}

	product, err := shopRepo.GetProductByID(ctx, convertedId)
	if err != nil {
		c.Logger().Error("Failed to retrieve product: ", err)
		if err.Error() == "product not found" {
			return c.JSON(404, echo.Map{
				"message": "Product not found",
			})
		}
		return c.JSON(500, echo.Map{
			"message": "Internal server error",
		})
	}

	return c.JSON(200, echo.Map{
		"message":  "Product retrieved successfully",
		"product":  product,
		"duration": time.Since(time.Now()).String(),
	})
}

func GetProductBySlug(c echo.Context) error {
	ctx := c.Request().Context()
	paramsSlug := c.Param("slug")

	if paramsSlug == "" {
		c.JSON(400, echo.Map{
			"message": "Product slug is required",
		})
		return nil
	}

	shopRepo := models.NewMongoClient(server.Client)
	product, err := shopRepo.GetProductBySlug(ctx, paramsSlug)
	if err != nil {
		c.Logger().Error("Failed to retrieve product: ", err)
		if err.Error() == "product not found" {
			return c.JSON(404, echo.Map{
				"message": " slug not found",
				"slug":    paramsSlug,
			})
		}
		return c.JSON(500, echo.Map{
			"message": "Internal server error",
		})
	}

	return c.JSON(200, product)
}

func UpdateProduct(c echo.Context) error {
	ctx := c.Request().Context()
	paramsId := c.Param("id")
	if paramsId == "" {
		c.JSON(400, echo.Map{
			"message": "Product ID is required",
		})
	}

	var product map[string]interface{}

	// get role from cookies
	role := c.Get("role").(string)

	// Check if the user is authorized to update the product
	if role != "admin" {
		c.Logger().Error("Unauthorized access attempt")
		return c.JSON(403, echo.Map{
			"message": "Forbidden: You do not have permission to perform this action",
		})
	}

	// Bind the request body to the product struct
	if err := c.Bind(&product); err != nil {
		c.Logger().Error("Failed to bind product data: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid input",
			"error":   err.Error(),
		})
	}
	convertedId, err := primitive.ObjectIDFromHex(paramsId)
	if err != nil {
		c.Logger().Error("Failed to convert product ID: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid product ID format",
			"error":   err.Error(),
		})
	}
	shopRepo := models.NewMongoClient(server.Client)
	id, err := shopRepo.UpdateProduct(ctx, convertedId, product)
	if err != nil {
		c.Logger().Error("Failed to update product: ", err)
		return c.JSON(500, echo.Map{
			"message": "Failed to update product",
			"error":   err.Error(),
		})
	}

	return c.JSON(200, echo.Map{
		"message": "Product updated successfully",
		"id":      id,
	})
}

// DELETEPRODUCT HANDLES THE DELETION OF A PRODUCT
func DeleteProduct(c echo.Context) error {
	var requestBody struct {
		ID string `json:"id"`
	}

	if err := c.Bind(&requestBody); err != nil {
		c.Logger().Error("Failed to bind request body: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}
	// Retrieve role from context
	role, ok := c.Get("role").(string)
	if !ok {
		c.Logger().Error("Failed to retrieve role from context")
		return c.JSON(500, echo.Map{
			"message": "Internal server error",
		})
	}

	// Check if the user is authorized to delete the product
	if role != "admin" {
		c.Logger().Error("Unauthorized access attempt")
		return c.JSON(403, echo.Map{
			"message": "Forbidden: You do not have permission to perform this action",
		})
	}

	// Validate that the product ID is provided
	if requestBody.ID == "" {
		return c.JSON(400, echo.Map{
			"message": "Product ID is required",
		})
	}

	// Convert the product ID to ObjectID and proceed with deletion
	ctx := c.Request().Context()
	convertedId, err := primitive.ObjectIDFromHex(requestBody.ID)
	if err != nil {
		c.Logger().Error("Failed to convert product ID: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid product ID format",
			"error":   err.Error(),
		})
	}

	// Delete the product from the database
	shopRepo := models.NewMongoClient(server.Client)
	err = shopRepo.DeleteProduct(ctx, convertedId)
	if err != nil {
		c.Logger().Error("Failed to delete product: ", err)
		if err.Error() == "product not found" {
			return c.JSON(404, echo.Map{
				"message": "Product not found",
			})
		}
		return c.JSON(500, echo.Map{
			"message": "Failed to delete product",
		})
	}

	return c.JSON(200, echo.Map{
		"message": "Product deleted successfully",
	})
}

// func DeleteProductImage(c echo.Context) error {
// 	ctx := c.Request().Context()
// 	// Retrieve the user's role from the context and ensure only admins can delete images
// 	role, ok := c.Get("role").(string)
// 	if !ok || role == "" {
// 		c.Logger().Error("Failed to retrieve role from context")
// 		return c.JSON(500, echo.Map{
// 			"message": "Internal server error",
// 		})
// 	}
// 	if role != "admin" {
// 		c.Logger().Error("Unauthorized access attempt")
// 		return c.JSON(403, echo.Map{
// 			"message": "Forbidden: You do not have permission to perform this action",
// 		})
// 	}

// 	// Get the product ID from the URL parameter
// 	paramId := c.Param("id")
// 	if paramId == "" {
// 		c.Logger().Error("Product ID is required")
// 		return c.JSON(400, echo.Map{
// 			"message": "Product ID is required",
// 		})
// 	}

// 	// Convert the product ID from string to MongoDB ObjectID
// 	convertedId, err := primitive.ObjectIDFromHex(paramId)
// 	if err != nil {
// 		c.Logger().Error("Failed to convert product ID: ", err)
// 		return c.JSON(400, echo.Map{
// 			"message": "Invalid product ID format",
// 			"error":   err.Error(),
// 		})
// 	}

// 	shopRepo := models.NewMongoClient(server.Client)
// 	// Retrieve the product by its ID
// 	product, err := shopRepo.GetProductByID(ctx, convertedId)
// 	if err != nil {
// 		c.Logger().Error("Failed to retrieve product: ", err)
// 		if err.Error() == "product not found" {
// 			return c.JSON(404, echo.Map{
// 				"message": "Product not found",
// 			})
// 		}
// 		return c.JSON(500, echo.Map{
// 			"message": "Internal server error",
// 		})
// 	}

// 	// Check if the product has images to delete
// 	if len(product.Images) == 0 {
// 		c.Logger().Warn("No images found for product")
// 		return c.JSON(404, echo.Map{
// 			"message": "No images found for this product",
// 		})
// 	}

// 	// For demonstration, delete all images associated with the product.
// 	// You may want to modify this to delete a specific image by image ID.
// 	for _, img := range product.Images {
// 		if err := shopRepo.DeleteImage(ctx, product.ID, img.Id); err != nil {
// 			c.Logger().Error("Failed to delete product image: ", err)
// 			return c.JSON(500, echo.Map{
// 				"message": "Internal server error",
// 				"error":   err.Error(),
// 			})
// 		}
// 	}

// 	// Return a success response after deleting images
// 	return c.JSON(200, echo.Map{
// 		"message": "Product images deleted successfully",
// 	})
// }
