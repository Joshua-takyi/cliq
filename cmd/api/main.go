package main

import (
	"fmt"
	"os"

	"github.com/joshuatakyi/shop/internal/router"
	"github.com/joshuatakyi/shop/internal/server"
)

func main() {
	if err := server.InitializeConnection(); err != nil {
		fmt.Printf("Error initializing connection: %v\n", err)
		return
	}

	defer server.Disconnect()

	port :=os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified
	}
	r := router.Router()
	if err := r.Start(":" + port); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
