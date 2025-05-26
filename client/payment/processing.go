// client/payment/processing.go

package payment

import (
	"context"
	"log"
	"net"
	"strings"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// PaymentProcessor handles payment processing logic
type PaymentProcessor struct {
	client      pb.PaymentGatewayClient
	authClient  pb.AuthServiceClient
	credentials map[string]string    // Store credentials for offline payments
	authenticated map[string]bool    // Track authenticated users locally
}

// NewPaymentProcessor creates a new payment processor
func NewPaymentProcessor(client pb.PaymentGatewayClient, authClient pb.AuthServiceClient) *PaymentProcessor {
	return &PaymentProcessor{
		client:      client,
		authClient:  authClient,
		credentials: make(map[string]string),
		authenticated: make(map[string]bool),
	}
}

// AddCredentials adds or updates credentials for a user
func (pp *PaymentProcessor) AddCredentials(username, password string) {
	pp.credentials[username] = password
	// Reset authentication status when credentials change
	pp.authenticated[username] = false
}

// StoreCredentials stores user credentials for offline reauth
// This should be called when a user successfully authenticates
func (pp *PaymentProcessor) StoreCredentials(username, password string) {
	pp.credentials[username] = password
}

// CheckConnection checks if we can connect to the payment gateway
func (pp *PaymentProcessor) CheckConnection() bool {
	// Try a simple ping to the server (could be replaced with a health check RPC)
	conn, err := net.DialTimeout("tcp", "localhost:50053", ConnectionCheckTimeout)
	if err != nil {
		log.Printf("[OFFLINE] Connection check failed: %v", err)
		return false
	}
	conn.Close()
	return true
}

// ProcessPayment processes a single payment and returns result
func (pp *PaymentProcessor) ProcessPayment(payment PendingPayment) (bool, string) {
    // Enhanced logging for transfers
    if payment.ReceiverName != "" {
        log.Printf("[OFFLINE] Processing transfer: %s → %s, Amount: %.2f, Transaction ID: %s", 
            payment.ClientName, payment.ReceiverName, payment.Amount, payment.TransactionID)
    } else {
        log.Printf("[OFFLINE] Processing payment: %s, Amount: %.2f, Transaction ID: %s", 
            payment.ClientName, payment.Amount, payment.TransactionID)
    }
        
    // Send the payment
    success, message := pp.sendPayment(payment)
    
    if success {
        if payment.ReceiverName != "" {
            log.Printf("[OFFLINE] Transfer successful: %s → %s, ID: %s", 
                payment.ClientName, payment.ReceiverName, payment.TransactionID)
        } else {
            log.Printf("[OFFLINE] Payment successful: %s", payment.TransactionID)
        }
    } else {
        if payment.ReceiverName != "" {
            log.Printf("[OFFLINE] Transfer failed: %s → %s, ID: %s, Message: %s", 
                payment.ClientName, payment.ReceiverName, payment.TransactionID, message)
        } else {
            log.Printf("[OFFLINE] Payment failed: %s, Message: %s", payment.TransactionID, message)
        }
    }
    
    return success, message
}

// IsConnectionError determines if an error message indicates a connectivity issue
func (pp *PaymentProcessor) IsConnectionError(message string) bool {
	return strings.Contains(message, "unavailable") || 
		   strings.Contains(message, "deadline exceeded") ||
		   strings.Contains(message, "connection refused")
}

// authenticateUser attempts to authenticate the user with stored credentials
func (pp *PaymentProcessor) authenticateUser(username string) bool {
	// First check if user is already authenticated
	if pp.authenticated[username] {
		log.Printf("[OFFLINE] User %s already authenticated", username)
		return true
	}
	
	// Check if we have credentials for this user
	password, exists := pp.credentials[username]
	if !exists {
		log.Printf("[OFFLINE] No stored credentials for user %s", username)
		return false
	}
	
	// Set up timeout for the authentication request
	ctx, cancel := context.WithTimeout(context.Background(), PaymentTimeout)
	defer cancel()
	
	// Try to authenticate
	authResp, err := pp.authClient.Authenticate(ctx, &pb.AuthRequest{
		Username: username,
		Password: password,
	})
	
	if err != nil {
		log.Printf("[OFFLINE] Authentication request failed: %v", err)
		return false
	}
	
	if authResp.Success {
		// Mark user as authenticated
		pp.authenticated[username] = true
		log.Printf("[OFFLINE] User %s authenticated successfully", username)
	} else {
		log.Printf("[OFFLINE] Authentication failed for user %s: %s", username, authResp.Message)
	}
	
	return authResp.Success
}


// sendPayment sends a payment to the payment gateway
// sendPayment sends a payment to the payment gateway
func (pp *PaymentProcessor) sendPayment(payment PendingPayment) (bool, string) {
    // Try to authenticate the user first
    authenticated := pp.authenticateUser(payment.ClientName)
    if !authenticated {
        // return false, "user not authenticated"
    }
    
    // Create context with username and transaction_id
    md := metadata.Pairs(
        "username", payment.ClientName,
        "transaction_id", payment.TransactionID,
    )
    ctx := metadata.NewOutgoingContext(context.Background(), md)
    
    // Set a timeout for the payment request
    timeoutCtx, cancel := context.WithTimeout(ctx, PaymentTimeout)
    defer cancel()
    
    // Create payment request
    paymentDetails := &pb.PaymentDetails{
        ClientName: payment.ClientName,
        Amount:     payment.Amount,
		ReceiverName: payment.ReceiverName,
        RequestId:    payment.RequestId,  // Include the requestID
    }
    
    // Add receiver name if present (for transfers using 2PC)
    if payment.ReceiverName != "" {
        paymentDetails.ReceiverName = payment.ReceiverName
    }
    
    paymentResp, err := pp.client.MakePayment(timeoutCtx, paymentDetails)
    
    if err != nil {
        st, _ := status.FromError(err)
        return false, st.Message()
    }
    
    return paymentResp.Success, paymentResp.Message
}