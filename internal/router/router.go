// router/router.go
package router

import (
	"fmt"
	"os"

	"github.com/joshuatakyi/shop/internal/database"
	"github.com/joshuatakyi/shop/internal/middleware"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

func Router() *echo.Echo {
	e := echo.New()

	// Middleware
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	// Remove the default CORS middleware as we're using a custom configuration below

	frontendUrl := os.Getenv("NEXT_API_URL")
	if frontendUrl == "" {
		frontendUrl = "http://localhost:3000" // Default to localhost if not set
		fmt.Println("NEXT_API_URL not set, using default:", frontendUrl)
	}

	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:     []string{frontendUrl},                                                                             // Allow specific origin (frontend URL)
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},                                      // Allow specific HTTP methods
		AllowHeaders:     []string{"Content-Type", "Authorization", "Accept", "Origin", "X-Requested-With", "X-CSRF-Token"}, // Extended allowed headers
		AllowCredentials: true,                                                                                              // Allow credentials (cookies, authorization headers, etc.)
		MaxAge:           86400,                                                                                             // Cache preflight requests for 24 hours
	}))
	v1 := e.Group("/api/v1")

	// healthcheck
	v1.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"status":  "ok",
			"message": "Server is running",
		})
	})

	// Public routes for products
	v1.GET("/products", database.ListProducts)
	// v1.GET("/products/:id", database.GetProductByID)
	v1.GET("/products/slug/:slug", database.GetProductBySlug)
	v1.GET("/get_product_by_id/:id", database.GetProductByID)

	// PROTECTED ROUTES

	protected := v1.Group("/protected")
	protected.Use(middleware.AuthMiddleware())

	{
		// admin routes
		protected.POST("/create_product", database.CreateProduct)
		protected.PATCH("/update_product/:id", database.UpdateProduct)
		protected.DELETE("/delete_product", database.DeleteProduct)
		// protected.GET("/delete_image/:id", database.DeleteProductImage)

		protected.GET("/verify", database.VerifySession)
		protected.POST("/add_comment/:id", database.AddComment)
		protected.GET("/get_comments/:id", database.GetComments)
		protected.PATCH("/delete_comment", database.DeleteComment)

		// protected.GET("/get_user_comments", database.GetUserComments)

		// Cart routes
		// protected.GET("/cart", database.GetUserCart)
		protected.POST("/add_to_cart", database.AddToCart)
		protected.GET("/get_cart", database.GetUserCart)
		protected.PATCH("/update_cart", database.UpdateCart)
		protected.DELETE("/clear_cart", database.ClearCart)
		protected.DELETE("/remove_from_cart", database.RemoveCartItem)
	}

	return e
}
