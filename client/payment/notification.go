// client/payment/notification.go

package payment

import (
	"fmt"
	"sync"
	"time"
)

// NotificationManager handles payment notifications
type NotificationManager struct {
	notifications   map[string]PendingPayment
	mu              sync.Mutex
	stopChan        chan struct{}
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		notifications: make(map[string]PendingPayment),
		stopChan:      make(chan struct{}),
	}
}

// AddNotification adds a payment to notifications
func (nm *NotificationManager) AddNotification(payment PendingPayment) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	
	nm.notifications[payment.TransactionID] = payment
}

// UpdateNotification updates an existing notification
func (nm *NotificationManager) UpdateNotification(transactionID, status string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	
	if payment, exists := nm.notifications[transactionID]; exists {
		payment.Status = status
		nm.notifications[transactionID] = payment
	}
}

// GetNotification returns a notification for a transaction and removes it
func (nm *NotificationManager) GetNotification(transactionID string) (PendingPayment, bool) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	notification, exists := nm.notifications[transactionID]
	if exists {
		delete(nm.notifications, transactionID)
	}
	return notification, exists
}

// GetAllNotifications returns all notifications and clears them
func (nm *NotificationManager) GetAllNotifications() map[string]PendingPayment {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	notifications := make(map[string]PendingPayment)
	for k, v := range nm.notifications {
		notifications[k] = v
	}
	nm.notifications = make(map[string]PendingPayment)
	return notifications
}

// StartNotificationChecker starts a ticker to check for notifications periodically
func (nm *NotificationManager) StartNotificationChecker() {
	notificationTicker := time.NewTicker(NotificationCheckInterval)
	
	go func() {
		for {
			select {
			case <-notificationTicker.C:
				nm.checkNotifications()
			case <-nm.stopChan:
				notificationTicker.Stop()
				return
			}
		}
	}()
}

// Stop stops the notification manager
func (nm *NotificationManager) Stop() {
	close(nm.stopChan)
}

// checkNotifications periodically checks for notifications to display
func (nm *NotificationManager) checkNotifications() {
	notifications := nm.GetAllNotifications()
	if len(notifications) > 0 {
		fmt.Println("\n===== PAYMENT NOTIFICATIONS =====")
		for _, payment := range notifications {
			status := payment.Status
			if status == StatusCompleted {
				fmt.Printf("✅ Payment of %.2f for %s SUCCEEDED\n", 
					payment.Amount, payment.ClientName)
			} else if status == StatusFailed {
				fmt.Printf("❌ Payment of %.2f for %s FAILED\n", 
					payment.Amount, payment.ClientName)
			} else if status == StatusPending {
				fmt.Printf("⏳ Payment of %.2f for %s QUEUED for later processing\n", 
					payment.Amount, payment.ClientName)
			}
		}
		fmt.Println("=================================")
	}
}

// DisplayNotifications displays all notifications to the user (moved from client.go)
func (nm *NotificationManager) DisplayNotifications() {
	notifications := nm.GetAllNotifications()
	
	fmt.Println("\n===== PAYMENT NOTIFICATIONS =====")
	if len(notifications) == 0 {
		fmt.Println("No new notifications")
	} else {
		for txID, payment := range notifications {
			status := payment.Status
			if status == "completed" {
				fmt.Printf("✅ Payment of %.2f for %s SUCCEEDED (ID: %s)\n", 
					payment.Amount, payment.ClientName, txID)
			} else if status == "failed" {
				fmt.Printf("❌ Payment of %.2f for %s FAILED (ID: %s)\n", 
					payment.Amount, payment.ClientName, txID)
			} else if status == "pending" {
				fmt.Printf("⏳ Payment of %.2f for %s QUEUED for later processing (ID: %s)\n", 
					payment.Amount, payment.ClientName, txID)
			}
		}
	}
	fmt.Println("=================================")
}

// DisplayTransactionStatus displays details about a specific transaction (moved from client.go)
func (nm *NotificationManager) DisplayTransactionStatus(transactionID string) bool {
	notification, exists := nm.GetNotification(transactionID)
	
	if exists {
		fmt.Printf("\n===== TRANSACTION STATUS =====\n")
		fmt.Printf("Transaction ID: %s\n", transactionID)
		fmt.Printf("Client: %s\n", notification.ClientName)
		fmt.Printf("Amount: %.2f\n", notification.Amount)
		fmt.Printf("Status: %s\n", notification.Status)
		fmt.Printf("Created At: %s\n", notification.CreatedAt.Format(time.RFC3339))
		if notification.RetryCount > 0 {
			fmt.Printf("Retry Attempts: %d\n", notification.RetryCount)
		}
		fmt.Println("==============================")
		return true
	}
	
	return false
}