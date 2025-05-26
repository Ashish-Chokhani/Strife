// client/payment/storage.go

package payment

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// Storage handles persistence operations for pending payments
type Storage struct {
	queueFile string
}

// NewStorage creates a new storage handler
func NewStorage(queueFile string) *Storage {
	s := &Storage{
		queueFile: queueFile,
	}
	
	// Create directory for queue file if it doesn't exist
	dir := filepath.Dir(queueFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
	
	return s
}

// LoadPendingPayments loads the pending payments from disk
func (s *Storage) LoadPendingPayments() ([]PendingPayment, error) {
	var payments []PendingPayment

	// Check if file exists
	if _, err := os.Stat(s.queueFile); os.IsNotExist(err) {
		log.Printf("[OFFLINE] No pending payments file found")
		return payments, nil
	}

	data, err := ioutil.ReadFile(s.queueFile)
	if err != nil {
		log.Printf("[OFFLINE] Failed to read pending payments: %v", err)
		return payments, err
	}

	if len(data) == 0 {
		log.Printf("[OFFLINE] Empty pending payments file")
		return payments, nil
	}

	err = json.Unmarshal(data, &payments)
	if err != nil {
		log.Printf("[OFFLINE] Failed to unmarshal pending payments: %v", err)
		return payments, err
	}

	log.Printf("[OFFLINE] Loaded %d pending payments", len(payments))
	return payments, nil
}

// SavePendingPayments saves the pending payments to disk
func (s *Storage) SavePendingPayments(payments []PendingPayment) error {
	data, err := json.MarshalIndent(payments, "", "  ")
	if err != nil {
		log.Printf("[OFFLINE] Failed to marshal pending payments: %v", err)
		return err
	}

	err = ioutil.WriteFile(s.queueFile, data, 0644)
	if err != nil {
		log.Printf("[OFFLINE] Failed to save pending payments: %v", err)
		return err
	}
	
	return nil
}