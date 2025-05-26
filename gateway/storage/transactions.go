package storage

import (
	"log"
	"sync"
	"time"

	pb "github.com/example/protofiles"
)

// TransactionRecord stores information about processed transactions
type TransactionRecord struct {
	TransactionID   string              // Unique transaction ID
	ClientName      string              // Client who initiated the transaction
	Amount          float32             // Transaction amount
	ProcessedAt     time.Time           // When the transaction was processed
	Result          *pb.PaymentResponse // The result of the transaction
}

// TransactionStorage handles transaction records for idempotency
type TransactionStorage struct {
	processedTransactions map[string]TransactionRecord
	cleanupTicker         *time.Ticker
	mu                    sync.Mutex
	transactionTTL        time.Duration
}

// NewTransactionStorage creates a new transaction storage
func NewTransactionStorage(ttl time.Duration) *TransactionStorage {
	storage := &TransactionStorage{
		processedTransactions: make(map[string]TransactionRecord),
		transactionTTL:        ttl,
	}
	
	return storage
}

// StartCleanupRoutine initializes a periodic cleanup for old transactions
func (s *TransactionStorage) StartCleanupRoutine(interval time.Duration) {
	s.cleanupTicker = time.NewTicker(interval)
	
	go func() {
		for range s.cleanupTicker.C {
			s.CleanupExpiredTransactions()
		}
	}()
}

// StopCleanupRoutine stops the cleanup routine
func (s *TransactionStorage) StopCleanupRoutine() {
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}
}

// CleanupExpiredTransactions removes transactions older than TTL
func (s *TransactionStorage) CleanupExpiredTransactions() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	
	before := len(s.processedTransactions)
	
	for id, record := range s.processedTransactions {
		if now.Sub(record.ProcessedAt) > s.transactionTTL {
			delete(s.processedTransactions, id)
		}
	}
	
	after := len(s.processedTransactions)
	if before != after {
		log.Printf("Cleaned up %d expired transaction records, %d remaining", before-after, after)
	}
}

// StoreTransaction saves a transaction record
func (s *TransactionStorage) StoreTransaction(record TransactionRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.processedTransactions[record.TransactionID] = record
	log.Printf("Stored transaction record %s, total records: %d", 
		record.TransactionID, len(s.processedTransactions))
}

// GetTransaction retrieves a transaction record if it exists
func (s *TransactionStorage) GetTransaction(transactionID string) (TransactionRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	record, exists := s.processedTransactions[transactionID]
	return record, exists
}

// HasTransaction checks if a transaction ID exists
func (s *TransactionStorage) HasTransaction(transactionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	_, exists := s.processedTransactions[transactionID]
	return exists
}