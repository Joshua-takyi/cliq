package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

// UserInfo represents the user data returned from Next Auth verification
type UserInfo struct {
	UserId string `json:"userId"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}

func AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := ""
			isNextAuthToken := false

			// Check for next-auth session cookie first
			sessionCookie, err := c.Cookie("authjs.session-token")
			if err == nil && sessionCookie.Value != "" {
				token = sessionCookie.Value
				isNextAuthToken = true
			}

			// If not found, check for our custom auth-token cookie
			if token == "" {
				authCookie, err := c.Cookie("auth-token")
				if err == nil && authCookie.Value != "" {
					token = authCookie.Value
				}
			}

			// Finally, check Authorization header (Bearer token)
			if token == "" {
				authHeader := c.Request().Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			// If no token found in any location
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
			}

			// For Next Auth tokens, we need to handle them differently
			if isNextAuthToken {
				// Verify the Next Auth token by calling our Next.js API endpoint
				userInfo, err := verifyNextAuthToken(c.Request().Header.Get("Cookie"))
				if err != nil {
					return echo.NewHTTPError(http.StatusUnauthorized, "Invalid Next Auth token: "+err.Error())
				}

				// Set user information from Next Auth
				c.Set("userId", userInfo.UserId)
				c.Set("email", userInfo.Email)
				c.Set("name", userInfo.Name)
				c.Set("role", userInfo.Role)
				c.Set("isNextAuthToken", true)
				return next(c)
			}

			// Validate token and extract claims
			claims, err := ValidateJWT(token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid Token: "+err.Error())
			}

			// For standard JWTs, we use the sub claim for userId
			if sub, ok := claims["sub"].(string); ok {
				c.Set("userId", sub)
			} else if id, ok := claims["userId"].(string); ok {
				c.Set("userId", id)
			}

			// Set email if available
			if email, ok := claims["email"].(string); ok {
				c.Set("email", email)
			}

			// Handle optional role
			if role, ok := claims["role"].(string); ok {
				c.Set("role", role)
			}
			// If no errors occurred, proceed to the next handler
			return next(c)
		}
	}
}

// verifyNextAuthToken calls the Next.js API to verify a Next Auth token and get user info

func verifyNextAuthToken(cookies string) (*UserInfo, error) {
	// Get the Next.js API URL from environment or use default
	// load frontend URL from environment variable
	if err := godotenv.Load(".env.local"); err != nil {
		return nil, fmt.Errorf("error loading .env.local file: %v", err)
	}

	nextJsApiUrl := os.Getenv("NEXT_API_URL")
	if nextJsApiUrl == "" {
		nextJsApiUrl = "http://localhost:3000"
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", nextJsApiUrl+"/api/auth/verify", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set cookies from the original request to maintain the session
	if cookies != "" {
		req.Header.Set("Cookie", cookies)
	}

	// Send the request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling Next.js API: %v", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("next.js API returned status: %d", resp.StatusCode)
	}

	// Parse the response
	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &userInfo, nil
}

func ValidateJWT(token string) (map[string]any, error) {
	// Get JWT secret from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// Fallback to auth secret if JWT_SECRET is not set
		jwtSecret = os.Getenv("BETTER_AUTH_SECRET")
		if jwtSecret == "" {
			return nil, fmt.Errorf("JWT secret not configured")
		}
	}

	// Parse and validate the JWT token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	// Check token validity
	if !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract and validate claims
	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
		// Verify expiration manually if needed
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return nil, fmt.Errorf("token is expired")
			}
		}

		return claims, nil
	}

	return nil, fmt.Errorf("invalid claims format")
}
