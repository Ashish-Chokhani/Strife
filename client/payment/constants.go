// client/payment/constants.go

package payment

import "time"

// Constants for the payment package
const (
	ConnectionCheckInterval    = 15 * time.Second
	NotificationCheckInterval  = 5 * time.Second
	PaymentTimeout             = 3 * time.Second
	PaymentProcessingDelay     = 500 * time.Millisecond
	ConnectionCheckTimeout     = 2 * time.Second
)

// Status constants
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)