package ui

import (
	"bufio"
	"fmt"
)

// UIManager handles all user interface operations
type UIManager struct {
	scanner *bufio.Scanner
}

// NewUIManager creates a new UI manager
func NewUIManager(scanner *bufio.Scanner) *UIManager {
	return &UIManager{
		scanner: scanner,
	}
}

// GetUserInput gets user input from the console
func (ui *UIManager) GetUserInput() string {
	ui.scanner.Scan()
	return ui.scanner.Text()
}

// DisplayMainMenu displays the main menu
func (ui *UIManager) DisplayMainMenu() {
	fmt.Println("\n===== PAYMENT GATEWAY CLIENT =====")
	fmt.Println("1. Client Operations")
	fmt.Println("2. Exit")
	fmt.Println("Choose option:")
}

// DisplayClientPrompt displays the client name prompt
func (ui *UIManager) DisplayClientPrompt() {
	fmt.Println("\n===== CLIENT OPERATIONS =====")
	fmt.Println("Enter client name:")
}

// DisplayOperationsMenu displays the operations menu
func (ui *UIManager) DisplayOperationsMenu() {
	fmt.Println("Choose operation:")
	fmt.Println("1. Check Balance")
	fmt.Println("2. Make Payment")
	fmt.Println("3. View Pending Payments")
	fmt.Println("4. Check Transaction Status")
	fmt.Println("5. View Payment Notifications")
	fmt.Println("6. Back to Main Menu")
}

// DisplayTransferTypeMenu shows payment type options
func (ui *UIManager) DisplayTransferTypeMenu() {
    fmt.Println("\n--- Payment Type ---")
    fmt.Println("1. Simple Payment (withdraw from your account)")
    fmt.Println("2. Transfer to Another Account (Two-Phase Commit)")
    fmt.Print("Enter your choice: ")
}

// DisplayPaymentDetails displays payment details
func (ui *UIManager) DisplayPaymentDetails(clientName, transactionID string, amount float32, success bool, message string) {
	if success {
		fmt.Printf("Payment of %.2f for %s successful!\n", amount, clientName)
		fmt.Printf("Transaction ID: %s\n", transactionID)
	} else {
		fmt.Printf("Payment failed: %s\n", message)
		fmt.Printf("Transaction ID: %s\n", transactionID)
	}
}

// DisplayBalance displays the client's balance
func (ui *UIManager) DisplayBalance(clientName string, balance float32, currency string) {
	fmt.Printf("Balance for %s: %.2f %s\n", clientName, balance, currency)
}

// DisplayError displays an error message
func (ui *UIManager) DisplayError(message string, err error) {
	fmt.Printf("%s: %v\n", message, err)
}