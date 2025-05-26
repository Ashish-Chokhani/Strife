// client/interceptors/transaction.go

package interceptors

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

var (
	// Global transaction cache for retry idempotency
	transactionCache = make(map[string]string)
	cacheMutex       sync.RWMutex
)

// TransactionMetadata contains all information needed for idempotency
type TransactionMetadata struct {
    ClientName   string
    ReceiverName string
    Amount       string
    RequestId    string // Unique request ID
}



func ensureTransactionID(ctx context.Context, metadata TransactionMetadata) context.Context {
    // Create a deterministic cache key based on all fields
    cacheKey := fmt.Sprintf("%s:%s:%s:%s", 
        metadata.ClientName, 
        metadata.ReceiverName, 
        metadata.Amount,
        metadata.RequestId)
    
    // Get or generate transaction ID with proper locking
    transactionID := getOrCreateTransactionID(cacheKey)
    
    // Add transaction ID to the context metadata
    return addTransactionIDToContext(ctx, transactionID)
}

// getOrCreateTransactionID retrieves an existing transaction ID or creates a new one
func getOrCreateTransactionID(cacheKey string) string {
	cacheMutex.RLock()
	transactionID, exists := transactionCache[cacheKey]
	cacheMutex.RUnlock()
	
	if exists {
		log.Printf("[CLIENT] Reusing transaction ID %s for key %s", 
			transactionID, cacheKey)
		return transactionID
	}
	
	// Generate a new transaction ID if this is the first attempt
	transactionID = uuid.New().String()
	
	cacheMutex.Lock()
	transactionCache[cacheKey] = transactionID
	cacheMutex.Unlock()
	
	log.Printf("[CLIENT] Generated new transaction ID %s for key %s", 
		transactionID, cacheKey)
	
	return transactionID
}

// addTransactionIDToContext adds the transaction ID to the outgoing context
func addTransactionIDToContext(ctx context.Context, transactionID string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md = md.Copy()
	md.Set("transaction_id", transactionID)
	return metadata.NewOutgoingContext(ctx, md)
}

// ClearTransactionCache clears the transaction cache (useful for testing)
func ClearTransactionCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	transactionCache = make(map[string]string)
}