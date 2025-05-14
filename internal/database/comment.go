package database

import (
	"github.com/joshuatakyi/shop/internal/models"
	"github.com/joshuatakyi/shop/internal/server"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func AddComment(c echo.Context) error {
	ctx := c.Request().Context()
	var comment models.Comments

	// Retrieve productId from request parameters
	productId := c.Param("id")
	if productId == "" {
		c.Logger().Error("Product ID is missing in the request")
		return c.JSON(400, echo.Map{
			"message": "Product ID is required",
		})
	}

	userId, ok := c.Get("userId").(string)
	if !ok {
		c.Logger().Error("Failed to retrieve userId from context")
		return c.JSON(500, echo.Map{
			"message": "Internal server error",
		})
	}

	// Bind the request body to the comment struct
	if err := c.Bind(&comment); err != nil {
		c.Logger().Error("Failed to bind comment data: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid input",
			"error":   err.Error(),
		})
	}

	shopRepo := models.NewMongoClient(server.Client)

	convertedId, err := primitive.ObjectIDFromHex(productId)
	if err != nil {
		c.Logger().Error("Invalid product ID format: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid product ID format",
			"error":   err.Error(),
		})
	}
	// Add the comment to the product
	err = shopRepo.AddComment(ctx, comment, userId, convertedId)
	if err != nil {
		c.Logger().Error("Failed to add comment: ", err)
		return c.JSON(500, echo.Map{
			"message": "Failed to create comment",
			"error":   err.Error(),
		})
	}

	return c.JSON(201, echo.Map{
		"message": "Comment added successfully",
		"id":      convertedId,
	})
}

func GetComments(c echo.Context) error {
	ctx := c.Request().Context()
	productId := c.Param("id")
	if productId == "" {
		c.Logger().Error("Product ID is missing in the request")
		return c.JSON(400, echo.Map{
			"message": "Product ID is required",
		})
	}

	role, ok := c.Get("role").(string)
	if !ok {
		c.Logger().Error("Failed to retrieve role from context")
		return c.JSON(500, echo.Map{
			"message": "Internal server error",
		})
	}

	if role != "admin" {
		c.Logger().Error("Unauthorized access attempt")
		return c.JSON(403, echo.Map{
			"message": "Forbidden: You do not have permission to perform this action",
		})
	}

	convertedId, err := primitive.ObjectIDFromHex(productId)
	if err != nil {
		c.Logger().Error("Invalid product ID format: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid product ID format",
			"error":   err.Error(),
		})
	}

	shopRepo := models.NewMongoClient(server.Client)
	comments, err := shopRepo.GetComments(ctx, convertedId)
	if err != nil {
		c.Logger().Error("Failed to retrieve comments: ", err)
		return c.JSON(500, echo.Map{
			"message": "Failed to retrieve comments",
			"error":   err.Error(),
		})
	}

	return c.JSON(200, comments)
}

func DeleteComment(c echo.Context) error {
	ctx := c.Request().Context()
	userId, ok := c.Get("userId").(string)
	if !ok {
		c.Logger().Error("Failed to retrieve userId from context")
		return c.JSON(500, echo.Map{
			"message": "Internal server error",
		})
	}

	if userId == "" {
		c.Logger().Error("User ID is missing in the request")
		return c.JSON(400, echo.Map{
			"message": "User ID is required",
		})
	}

	// Retrieve id from request body
	var requestBody struct {
		Id string `json:"id"`
	}

	if err := c.Bind(&requestBody); err != nil {
		c.Logger().Error("Failed to bind request body: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid input",
			"error":   err.Error(),
		})
	}

	if requestBody.Id == "" {
		c.Logger().Error("Comment ID is missing in the request body")
		return c.JSON(400, echo.Map{
			"message": "Comment ID is required",
		})
	}

	convertedId, err := primitive.ObjectIDFromHex(requestBody.Id)
	if err != nil {
		c.Logger().Error("Invalid comment ID format: ", err)
		return c.JSON(400, echo.Map{
			"message": "Invalid comment ID format",
			"error":   err.Error(),
		})
	}

	shopRepo := models.NewMongoClient(server.Client)

	// First, fetch the comment to check ownership
	comment, err := shopRepo.GetCommentById(ctx, convertedId)
	if err != nil {
		c.Logger().Error("Failed to retrieve comment: ", err)
		return c.JSON(404, echo.Map{
			"message": "Comment not found",
		})
	}

	// Verify the user is the author of the comment
	if userId != comment.UserId {
		c.Logger().Error("Unauthorized access attempt")
		return c.JSON(403, echo.Map{
			"message": "Forbidden: Only the comment author can delete this comment",
		})
	}

	// Delete the comment
	err = shopRepo.DeleteComment(ctx, convertedId, userId)
	if err != nil {
		c.Logger().Error("Failed to delete comment: ", err)
		if err.Error() == "comment not found" {
			return c.JSON(404, echo.Map{
				"message": "Comment not found",
			})
		}
		return c.JSON(500, echo.Map{
			"message": "Failed to delete comment",
			"error":   err.Error(),
		})
	}

	return c.JSON(200, echo.Map{
		"message": "Comment deleted successfully",
	})
}
