package database

import "github.com/labstack/echo/v4"

// Example of an endpoint to verify the session
func VerifySession(c echo.Context) error {
    // The middleware has already verified the token and added the claims
    userId := c.Get("userId").(string)
    email := c.Get("email").(string)
    role := c.Get("role").(string)
    
    return c.JSON(200, echo.Map{
        "userId": userId,
        "email": email,
        "role": role,
        "authenticated": true,
    })
}