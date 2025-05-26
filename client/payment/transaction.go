// client/payment/transaction.go

package payment

import (
	"sync"

	"github.com/google/uuid"
)

// TransactionManager handles transaction ID management
type TransactionManager struct {
	cache map[string]string
	mu    sync.Mutex
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager() *TransactionManager {
	return &TransactionManager{
		cache: make(map[string]string),
	}
}

// GetOrCreateTransactionID gets an existing transaction ID or creates a new one
func (tm *TransactionManager) GetOrCreateTransactionID(cacheKey string) string {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	transactionID, exists := tm.cache[cacheKey]
	if !exists {
		transactionID = uuid.New().String()
		tm.cache[cacheKey] = transactionID
	}
	
	return transactionID
}

// RemoveTransactionID removes a transaction ID from the cache
func (tm *TransactionManager) RemoveTransactionID(cacheKey string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	delete(tm.cache, cacheKey)
}