// client/payment/models.go

package payment

import (
	"time"
)

// PendingPayment represents a payment that couldn't be processed due to connectivity issues
type PendingPayment struct {
	ClientName    string    `json:"clientName"`
	Amount        float32   `json:"amount"`
	TransactionID string    `json:"transactionID"`
	ReceiverName  string    `json:"receiver_name,omitempty"` // Add this field
	RequestId     string
	CreatedAt     time.Time `json:"createdAt"`
	RetryCount    int       `json:"retryCount"`
	Status        string    `json:"status"` // "pending", "processing", "completed", "failed"
}

type Balance struct {
	Amount   float32
	Currency string
}