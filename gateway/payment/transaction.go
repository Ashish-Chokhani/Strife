// gateway/payment/transaction.go

package payment

import (
	"context"
	"log"
	"sync"
	"time"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc/metadata"
)

// SimpleTransactionRecord stores information about processed single-phase transactions
type SimpleTransactionRecord struct {
	TransactionID string
	ClientName    string
	Amount        float32
	ProcessedAt   time.Time
	Result        *pb.PaymentResponse
}

// TwoPhaseTransactionRecord stores information about 2PC transactions
type TwoPhaseTransactionRecord struct {
	ID           string
	Sender       string
	Receiver     string
	Amount       float32
	State        TransactionState
	PrepareVotes map[string]bool // Bank votes for prepare phase
	CommitVotes  map[string]bool // Bank votes for commit phase
	Timestamp    time.Time
	Timeout      time.Duration
	mu           sync.RWMutex
}

// TransactionManager handles transaction idempotency and storage for both transaction types
type TransactionManager struct {
	transactions   map[string]interface{} // Can store either SimpleTransactionRecord or TwoPhaseTransactionRecord
	mu             sync.RWMutex
	ttl            time.Duration
	cleanupTicker  *time.Ticker
	done           chan struct{}
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(ttl, cleanupInterval time.Duration) *TransactionManager {
	tm := &TransactionManager{
		transactions:  make(map[string]interface{}),
		ttl:           ttl,
		done:          make(chan struct{}),
		cleanupTicker: time.NewTicker(cleanupInterval),
	}

	// Start cleanup routine
	go tm.cleanupRoutine()
	return tm
}

// Stop stops the transaction manager
func (tm *TransactionManager) Stop() {
	if tm.cleanupTicker != nil {
		tm.cleanupTicker.Stop()
	}
	close(tm.done)
}

func (tm *TransactionManager) GetActiveTransactions() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	// Create a copy of the transactions map to return
	result := make(map[string]interface{})
	for id, tx := range tm.transactions {
		result[id] = tx
	}
	
	return result
}

// GetTransactionID extracts transaction ID from context
func (tm *TransactionManager) GetTransactionID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get("transaction_id")
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func (tx *TwoPhaseTransactionRecord) GetPrepareVotes() map[string]bool {
	tx.mu.RLock()
	defer tx.mu.RUnlock()
	
	// Create a copy of the votes map
	result := make(map[string]bool)
	for bank, vote := range tx.PrepareVotes {
		result[bank] = vote
	}
	
	return result
}

// GetCommitVotes returns a copy of the commit votes map
func (tx *TwoPhaseTransactionRecord) GetCommitVotes() map[string]bool {
	tx.mu.RLock()
	defer tx.mu.RUnlock()
	
	// Create a copy of the votes map
	result := make(map[string]bool)
	for bank, vote := range tx.CommitVotes {
		result[bank] = vote
	}
	
	return result
}

// StoreSimpleTransaction stores a simple transaction record
func (tm *TransactionManager) StoreSimpleTransaction(record SimpleTransactionRecord) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.transactions[record.TransactionID] = record
	log.Printf("Stored simple transaction record %s, total records: %d",
		record.TransactionID, len(tm.transactions))
}

// StoreTwoPhaseTransaction stores a 2PC transaction record
func (tm *TransactionManager) StoreTwoPhaseTransaction(id string, record *TwoPhaseTransactionRecord) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.transactions[id] = record
	log.Printf("Stored 2PC transaction record %s, total records: %d",
		id, len(tm.transactions))
}

// GetTransaction retrieves a transaction record (either type)
func (tm *TransactionManager) GetTransaction(id string) (interface{}, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	record, exists := tm.transactions[id]
	return record, exists
}

// HasActiveTransaction checks if a transaction is active
func (tm *TransactionManager) HasActiveTransaction(id string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	_, exists := tm.transactions[id]
	return exists
}

// CleanupTransaction removes a transaction
func (tm *TransactionManager) CleanupTransaction(id string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.transactions, id)
	log.Printf("Cleaned up transaction %s", id)
}

// cleanupRoutine periodically cleans up expired transactions
func (tm *TransactionManager) cleanupRoutine() {
	for {
		select {
		case <-tm.done:
			return
		case <-tm.cleanupTicker.C:
			tm.cleanupExpiredTransactions()
		}
	}
}

// cleanupExpiredTransactions removes expired transaction records
func (tm *TransactionManager) cleanupExpiredTransactions() {
	now := time.Now()
	tm.mu.Lock()
	defer tm.mu.Unlock()

	before := len(tm.transactions)
	var expiredIDs []string

	for id, data := range tm.transactions {
		var timestamp time.Time
		var expired bool

		// Handle different record types
		switch record := data.(type) {
		case SimpleTransactionRecord:
			timestamp = record.ProcessedAt
			expired = now.Sub(timestamp) > tm.ttl
		case *TwoPhaseTransactionRecord:
			timestamp = record.Timestamp
			expired = now.Sub(timestamp) > tm.ttl
			// If 2PC transaction is in progress, mark it as aborted if expired
			if expired && record.State != TransactionCommitted && record.State != TransactionAborted {
				record.State = TransactionAborted
				log.Printf("[CLEANUP] Transaction %s timed out, marking as aborted", id)
			}
		default:
			// Unknown record type, remove it
			expired = true
		}

		if expired {
			expiredIDs = append(expiredIDs, id)
		}
	}

	// Remove expired records
	for _, id := range expiredIDs {
		delete(tm.transactions, id)
	}

	after := len(tm.transactions)
	if before != after {
		log.Printf("Cleaned up %d expired transaction records, %d remaining", before-after, after)
	}
}