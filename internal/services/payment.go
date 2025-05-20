package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

// PaymentService handles payment operations using Paystack
type PaymentService struct {
	SecretKey string
	PublicKey string
	BaseURL   string
}

// PaystackTransactionRequest represents the request body for initializing a transaction
type PaystackTransactionRequest struct {
	Email       string `json:"email"`
	Amount      int    `json:"amount"` // Amount in kobo (for NGN) or cents (for other currencies)
	Reference   string `json:"reference,omitempty"`
	Currency    string `json:"currency,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
}

// PaystackResponse represents the standard response structure from Paystack
type PaystackResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PaystackTransactionResponse represents the data returned after initializing a transaction
type PaystackTransactionResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	AccessCode       string `json:"access_code"`
	Reference        string `json:"reference"`
}

// NewPaymentService creates and initializes a new payment service
func NewPaymentService() (*PaymentService, error) {
	// Load environment variables from .env.local file
	if err := godotenv.Load(".env.local"); err != nil {
		log.Printf("Warning: Error loading .env.local file: %v", err)
		// Continue execution, as environment variables might be set elsewhere
	}

	secretKey := os.Getenv("PAYSTACK_SECRET_KEY")
	publicKey := os.Getenv("PAYSTACK_PUBLIC_KEY")

	if secretKey == "" || publicKey == "" {
		return nil, fmt.Errorf("paystack keys are not set in the environment variables")
	}

	return &PaymentService{
		SecretKey: secretKey,
		PublicKey: publicKey,
		BaseURL:   "https://api.paystack.co",
	}, nil
}

// InitializeTransaction creates a new payment transaction
func (p *PaymentService) InitializeTransaction(email string, amountInSmallestUnit int, reference, currency, callbackURL string) (*PaystackTransactionResponse, error) {
	// Create the request payload
	payload := PaystackTransactionRequest{
		Email:       email,
		Amount:      amountInSmallestUnit, // Remember: Amount in kobo (for NGN) or cents (for others)
		Reference:   reference,
		Currency:    currency,
		CallbackURL: callbackURL,
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request payload: %v", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", p.BaseURL+"/transaction/initialize", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+p.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to Paystack: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Check if request was successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("paystack API returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var paystackResp PaystackResponse
	if err := json.Unmarshal(body, &paystackResp); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %v", err)
	}

	// Check if the transaction was successful
	if !paystackResp.Status {
		return nil, fmt.Errorf("paystack transaction initialization failed: %s", paystackResp.Message)
	}

	// Extract the transaction data
	dataBytes, err := json.Marshal(paystackResp.Data)
	if err != nil {
		return nil, fmt.Errorf("error re-marshalling data: %v", err)
	}

	// Unmarshal into the transaction response struct
	var transactionResp PaystackTransactionResponse
	if err := json.Unmarshal(dataBytes, &transactionResp); err != nil {
		return nil, fmt.Errorf("error unmarshalling transaction data: %v", err)
	}

	return &transactionResp, nil
}

// VerifyTransaction confirms if a transaction was successful
func (p *PaymentService) VerifyTransaction(reference string) (*PaystackResponse, error) {
	// Create the request
	req, err := http.NewRequest("GET", p.BaseURL+"/transaction/verify/"+reference, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating verification request: %v", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+p.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending verification request to Paystack: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading verification response body: %v", err)
	}

	// Check if request was successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("paystack verification API returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var paystackResp PaystackResponse
	if err := json.Unmarshal(body, &paystackResp); err != nil {
		return nil, fmt.Errorf("error unmarshalling verification response: %v", err)
	}

	return &paystackResp, nil
}
