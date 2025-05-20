package models

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

type PaymentBody struct {
	Amount float64 `json:"amount"`
	Email  string  `json:"email"`
}

// USE DETAILED ERROR MESSAGES FOR DEBUGGING

func InitializeCheckout(c echo.Context) error {
	_, ok := c.Get("role").(string)
	if !ok {
		c.Logger().Error("Failed to get user role from context")
		return c.JSON(500, map[string]string{"error": "Internal server error", "message": "Failed to get user role"})
	}

	var paymentBody PaymentBody
	if err := c.Bind(&paymentBody); err != nil {
		c.Logger().Error("Failed to bind payment body: %v", err)
		return c.JSON(400, map[string]string{"error": "Bad request", "message": "Failed to bind payment body"})
	}
	if paymentBody.Amount <= 0 {
		return c.JSON(400, map[string]string{"error": "Bad request", "message": "Invalid payment amount"})
	}
	if paymentBody.Email == "" {
		return c.JSON(400, map[string]string{"error": "Bad request", "message": "Email is required"})
	}

	// Make a request to Paystack to initialize the transaction
	// Paystack expects amount in kobo (smallest currency unit)
	// Convert the amount to kobo (multiply by 100)
	amountInKobo := int(paymentBody.Amount * 100)

	// Create the request body with the required parameters
	requestBody := map[string]interface{}{
		"email":  paymentBody.Email,
		"amount": amountInKobo, // Amount in kobo
	}

	// Marshal the request body to JSON
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		c.Logger().Error("Failed to marshal request body: %v", err)
		return c.JSON(500, map[string]string{"error": "Internal server error", "message": "Failed to prepare payment data"})
	}

	httpClient := &http.Client{}

	// Create a new request with the JSON body
	httpReq, err := http.NewRequest("POST", "https://api.paystack.co/transaction/initialize",
		bytes.NewBuffer(requestBodyBytes)) // Pass the JSON body here
	if err != nil {
		c.Logger().Error("Failed to create HTTP request: %v", err)
		return c.JSON(500, map[string]string{"error": "Internal server error", "message": "Failed to create HTTP request"})
	}

	// Set headers for the request
	httpReq.Header.Set("Authorization", "Bearer "+os.Getenv("PAYSTACK_SECRET_KEY"))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		c.Logger().Error("Failed to send HTTP request: %v", err)
		return c.JSON(500, map[string]string{"error": "Internal server error", "message": "Failed to send HTTP request"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Logger().Error("Paystack API returned non-200 status: %v", resp.Status)
		return c.JSON(500, map[string]string{"error": "Internal server error", "message": "Paystack API returned non-200 status"})
	}

	// Read the response body
	var paystackResponse struct {
		Data struct {
			AuthorizationURL string `json:"authorization_url"`
			AccessCode       string `json:"access_code"`
			Reference        string `json:"reference"`
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&paystackResponse); err != nil {
		c.Logger().Error("Failed to decode Paystack API response: %v", err)
		return c.JSON(500, map[string]string{"error": "Internal server error", "message": "Failed to decode Paystack API response"})
	}

	return c.JSON(200, paystackResponse.Data)
}
func VerifyTransaction(c echo.Context) error {
	// Get the reference from the query
	// Get the reference from the query parameters
	reference := c.QueryParam("reference")
	if reference == "" {
		return c.JSON(400, map[string]string{"error": "Bad request", "message": "Reference is required"})
	}

	// Make a request to Paystack to verify the transaction
	httpClient := &http.Client{}

	httpReq, err := http.NewRequest("GET", "https://api.paystack.co/transaction/verify/"+reference, nil)
	if err != nil {
		c.Logger().Error("Failed to create HTTP request: %v", err)
		return c.JSON(500, map[string]string{"error": "Internal server error", "message": "Failed to create HTTP request"})
	}

	httpReq.Header.Set("Authorization", "Bearer "+os.Getenv("PAYSTACK_SECRET_KEY"))

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		c.Logger().Error("Failed to send HTTP request: %v", err)
		return c.JSON(500, map[string]string{"error": "Internal server error", "message": "Failed to send HTTP request"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Logger().Error("Paystack API returned non-200 status: %v", resp.Status)
		return c.JSON(500, map[string]string{"error": "Internal server error", "message": "Paystack API returned non-200 status"})
	}

	// Parse the response body to extract the transaction status
	var paystackResponse struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&paystackResponse); err != nil {
		c.Logger().Error("Failed to decode Paystack API response: %v", err)
		return c.JSON(500, map[string]string{"error": "Internal server error", "message": "Failed to decode Paystack API response"})
	}

	// Use the extracted status to determine the transaction state
	switch paystackResponse.Data.Status {
	case "success":
		c.Logger().Info("Transaction successful")
		return c.JSON(200, map[string]string{"status": "success", "message": "Transaction verified successfully"})
	case "pending":
		return c.JSON(200, map[string]string{"status": "pending", "message": "Transaction is pending"})
	case "failed":
		return c.JSON(200, map[string]string{"status": "failed", "message": "Transaction failed"})
	case "abandoned":
		return c.JSON(200, map[string]string{"status": "abandoned", "message": "Transaction was abandoned"})
	case "cancelled":
		return c.JSON(200, map[string]string{"status": "cancelled", "message": "Transaction was cancelled"})
	default:
		return c.JSON(200, map[string]string{"status": "unknown", "message": "Transaction status is unknown"})
	}
}

// use webhook to create order if transaction is successful
