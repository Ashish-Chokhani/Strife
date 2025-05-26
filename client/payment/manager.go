// client/payment/manager.go

package payment

import (
	"log"
	"sync"
	"time"
	"fmt"

	pb "github.com/example/protofiles"
)

// OfflinePaymentManager handles offline payment functionality
type OfflinePaymentManager struct {
	pendingPayments    []PendingPayment
	mu                 sync.Mutex
	isConnected        bool
	stopChan           chan struct{}
	
	// Dependencies
	storage            *Storage
	processor          *PaymentProcessor
	notifier           *NotificationManager
}

// OfflinePaymentManagerConfig contains configuration for the payment manager
type OfflinePaymentManagerConfig struct {
    Client      pb.PaymentGatewayClient
    AuthClient  pb.AuthServiceClient // Add this line
    QueueFile   string
}

func (m *OfflinePaymentManager) GetProcessor() *PaymentProcessor {
	return m.processor
}

// NewOfflinePaymentManager creates a new payment manager
func NewOfflinePaymentManager(config OfflinePaymentManagerConfig) *OfflinePaymentManager {
	storage := NewStorage(config.QueueFile)
	processor := NewPaymentProcessor(config.Client, config.AuthClient) // Pass authClient here
	notifier := NewNotificationManager()
	
	manager := &OfflinePaymentManager{
		isConnected:     false,
		stopChan:        make(chan struct{}),
		storage:         storage,
		processor:       processor,
		notifier:        notifier,
	}

	// Load pending payments from disk
	manager.loadPendingPayments()

	// Start connection checker and notification checker
	go manager.startConnectionChecker()
	manager.notifier.StartNotificationChecker()

	return manager
}

// Stop stops the offline payment manager
func (m *OfflinePaymentManager) Stop() {
	close(m.stopChan)
	m.notifier.Stop()
}

// QueuePayment adds a payment to the offline queue
func (m *OfflinePaymentManager) QueuePayment(clientName string, amount float32, transactionID string, receiverName string, requestID string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Check if this payment is already in the queue
    for _, payment := range m.pendingPayments {
        if payment.TransactionID == transactionID || 
           (payment.ClientName == clientName && payment.RequestId == requestID && requestID != "") {
            log.Printf("[OFFLINE] Payment with transaction ID %s or request ID %s already queued", 
                transactionID, requestID)
            return
        }
    }

    payment := PendingPayment{
        ClientName:    clientName,
        Amount:        amount,
        TransactionID: transactionID,
        ReceiverName:  receiverName,
        RequestId:     requestID,    // Store the requestID
        CreatedAt:     time.Now(),
        RetryCount:    0,
        Status:        StatusPending,
    }

    m.pendingPayments = append(m.pendingPayments, payment)
    
    // Update logging to include request ID
    logMsg := "[OFFLINE] "
    if receiverName != "" {
        logMsg += fmt.Sprintf("Transfer queued: %s → %s, Amount: %.2f", 
            clientName, receiverName, amount)
    } else {
        logMsg += fmt.Sprintf("Payment queued: %s, Amount: %.2f", 
            clientName, amount)
    }
    logMsg += fmt.Sprintf(", Transaction ID: %s, Request ID: %s", transactionID, requestID)
    log.Println(logMsg)

    // Add to notifications
    m.notifier.AddNotification(payment)

    // Save updated queue
    m.storage.SavePendingPayments(m.pendingPayments)

    // Try to process if we're connected
    if m.isConnected {
        go m.processPayments()
    }
}

// GetNotification returns a notification for a transaction and removes it
func (m *OfflinePaymentManager) GetNotification(transactionID string) (PendingPayment, bool) {
	return m.notifier.GetNotification(transactionID)
}

// GetAllNotifications returns all notifications and clears them
func (m *OfflinePaymentManager) GetAllNotifications() map[string]PendingPayment {
	return m.notifier.GetAllNotifications()
}

// loadPendingPayments loads the pending payments from disk
func (m *OfflinePaymentManager) loadPendingPayments() {
	payments, err := m.storage.LoadPendingPayments()
	if err != nil {
		return
	}
	
	m.mu.Lock()
	m.pendingPayments = payments
	m.mu.Unlock()

	// Add all pending payments to notifications
	for _, payment := range payments {
		if payment.Status == StatusPending {
			m.notifier.AddNotification(payment)
		}
	}
}

// startConnectionChecker periodically checks the connection and processes payments when online
func (m *OfflinePaymentManager) startConnectionChecker() {
	ticker := time.NewTicker(ConnectionCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			connected := m.processor.CheckConnection()
			
			// If connection state changed from offline to online
			if connected && !m.isConnected {
				log.Printf("[OFFLINE] Connection restored, processing pending payments")
				m.processPayments()
			}
			
			m.isConnected = connected
			
		case <-m.stopChan:
			return
		}
	}
}

// processPayments processes all pending payments
func (m *OfflinePaymentManager) processPayments() {
	m.mu.Lock()
	// Make a copy of pending payments to avoid holding the lock during processing
	pendingPayments := make([]PendingPayment, len(m.pendingPayments))
	copy(pendingPayments, m.pendingPayments)
	m.mu.Unlock()

	if len(pendingPayments) == 0 {
		return
	}

	log.Printf("[OFFLINE] Processing %d pending payments", len(pendingPayments))
	
	for i, payment := range pendingPayments {
		log.Println(payment.Status)
		if payment.Status != StatusPending {
			continue
		}
		
		// Enhanced logging that includes receiver information for transfers
		if payment.ReceiverName != "" {
			log.Printf("[OFFLINE] Processing transfer %d/%d: %s → %s, Amount: %.2f, Transaction ID: %s", 
				i+1, len(pendingPayments), payment.ClientName, payment.ReceiverName, payment.Amount, payment.TransactionID)
		} else {
			log.Printf("[OFFLINE] Processing payment %d/%d: %s, Amount: %.2f, Transaction ID: %s", 
				i+1, len(pendingPayments), payment.ClientName, payment.Amount, payment.TransactionID)
		}
		
		// Mark as processing
		m.updatePaymentStatus(payment.TransactionID, StatusProcessing)
		
		// Process the payment - this calls the processor which needs to handle the ReceiverName
		success, message := m.processor.ProcessPayment(payment)
		
		if success {
			m.updatePaymentStatus(payment.TransactionID, StatusCompleted)
			
			// Update notification
			m.notifier.UpdateNotification(payment.TransactionID, StatusCompleted)
		} else {
			// Check if this is a connectivity issue or a permanent failure
			if m.processor.IsConnectionError(message) {
				// Connection issue, mark as pending again
				m.updatePaymentRetry(payment.TransactionID)
				
				// Exit the loop as we likely can't process other payments either
				log.Printf("[OFFLINE] Connection issues detected, stopping payment processing")
				break
			} else {
				// Permanent failure
				m.updatePaymentStatus(payment.TransactionID, StatusPending)
				
				// Update notification
				m.notifier.UpdateNotification(payment.TransactionID, StatusPending)
			}
		}
		
		// Small delay between processing payments
		time.Sleep(PaymentProcessingDelay)
	}
	
	// Remove completed payments after all processing is done
	m.cleanupCompletedPayments()
}

// updatePaymentStatus updates the status of a payment
func (m *OfflinePaymentManager) updatePaymentStatus(transactionID, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for i, payment := range m.pendingPayments {
		if payment.TransactionID == transactionID {
			m.pendingPayments[i].Status = status
			break
		}
	}
	
	m.storage.SavePendingPayments(m.pendingPayments)
}

// updatePaymentRetry increments the retry count of a payment
func (m *OfflinePaymentManager) updatePaymentRetry(transactionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for i, payment := range m.pendingPayments {
		if payment.TransactionID == transactionID {
			m.pendingPayments[i].RetryCount++
			m.pendingPayments[i].Status = StatusPending
			break
		}
	}
	
	m.storage.SavePendingPayments(m.pendingPayments)
}

// cleanupCompletedPayments removes completed payments from the queue
func (m *OfflinePaymentManager) cleanupCompletedPayments() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var activePayments []PendingPayment
	for _, payment := range m.pendingPayments {
		if payment.Status != StatusCompleted {
			activePayments = append(activePayments, payment)
		}
	}
	
	if len(m.pendingPayments) != len(activePayments) {
		log.Printf("[OFFLINE] Cleaned up %d completed payments", len(m.pendingPayments)-len(activePayments))
		m.pendingPayments = activePayments
		m.storage.SavePendingPayments(m.pendingPayments)
	}
}

// GetPendingPayments returns a copy of all pending payments
func (m *OfflinePaymentManager) GetPendingPayments() []PendingPayment {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Create a copy to avoid exposing internal state
	pendingPayments := make([]PendingPayment, len(m.pendingPayments))
	copy(pendingPayments, m.pendingPayments)
	
	return pendingPayments
}

// GetNotifier returns the notification manager
func (m *OfflinePaymentManager) GetNotifier() *NotificationManager {
	return m.notifier
}