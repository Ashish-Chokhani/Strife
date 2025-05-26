package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/example/client/auth"
	"github.com/example/client/config"
	"github.com/example/client/interceptors"
	"github.com/example/client/payment"
	"github.com/example/client/ui"  
	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ClientApp encapsulates the client application
type ClientApp struct {
	client             pb.PaymentGatewayClient
	authManager        *auth.AuthManager
	offlineManager     *payment.OfflinePaymentManager
	transactionManager *payment.TransactionManager
    requestCounter     *payment.RequestCounter
	uiManager          *ui.UIManager  // New UI manager
	scanner            *bufio.Scanner
}

// NewClientApp creates a new client application
func NewClientApp(
    client pb.PaymentGatewayClient,
    authManager *auth.AuthManager,
    offlineManager *payment.OfflinePaymentManager,
    transactionManager *payment.TransactionManager) *ClientApp {
    
    scanner := bufio.NewScanner(os.Stdin)
    
    return &ClientApp{
        client:             client,
        authManager:        authManager,
        offlineManager:     offlineManager,
        transactionManager: transactionManager,
        requestCounter:     payment.NewRequestCounter("data/request_counters.json"),
        uiManager:          ui.NewUIManager(scanner),
        scanner:            scanner,
    }
}

func main() {
    // Initialize configuration
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Set up TLS credentials
    creds, err := config.SetupTLSCredentials(cfg.CertPaths)
    if err != nil {
        log.Fatalf("Failed to setup TLS credentials: %v", err)
    }

    // Set up client interceptors
    interceptorChain := grpc.WithChainUnaryInterceptor(
        interceptors.LoggingInterceptor(),
        interceptors.RetryInterceptor(),
    )

    // Connect to gRPC server
    conn, err := grpc.Dial(
        cfg.ServerAddress,
        grpc.WithTransportCredentials(creds),
        interceptorChain,
    )

    if err != nil {
        log.Printf("Failed to connect to gateway: %v", err)
        log.Println("Starting in offline mode...")
    }
    defer conn.Close()

    client := pb.NewPaymentGatewayClient(conn)
    authClient := pb.NewAuthServiceClient(conn)

    // Initialize transaction manager
    transactionManager := payment.NewTransactionManager()

    offlineManager := payment.NewOfflinePaymentManager(payment.OfflinePaymentManagerConfig{
        Client:     client, // Your gRPC payment client
        AuthClient: authClient, // You need to create or get this from somewhere
        QueueFile:  "data/pending_payments.json",
    })
    defer offlineManager.Stop()

    // Initialize auth manager (but don't auto-authenticate)
    authManager := auth.NewAuthManager(client)

    // Create and run client app
    app := NewClientApp(client, authManager, offlineManager, transactionManager)
    app.Run()
}

// Run starts the interactive console
func (app *ClientApp) Run() {
    for {
        if !app.showMainMenu() {
            return
        }
    }
}

// showMainMenu displays the main menu and handles user selection
func (app *ClientApp) showMainMenu() bool {
    app.uiManager.DisplayMainMenu() // This will need to be updated to show only client operations option
    option := app.uiManager.GetUserInput()

    switch option {
    case "1":
        app.handleClientOperations()
    case "2":
        return false
    default:
        fmt.Println("Invalid option")
    }
    
    return true
}

// handleClientOperations manages client-specific operations
func (app *ClientApp) handleClientOperations() {
    app.uiManager.DisplayClientPrompt()
    clientName := app.uiManager.GetUserInput()

    // Authenticate the user if not already authenticated
    if !app.authManager.IsAuthenticated(clientName) {
        if !app.authenticateUser(clientName) {
            return
        }
    }

    // Now that the user is authenticated, show operations menu
    app.uiManager.DisplayOperationsMenu()
    operation := app.uiManager.GetUserInput()

    switch operation {
    case "1":
        app.handleCheckBalance(clientName)
    case "2":
        app.handleMakePayment(clientName)
    case "3":
        app.displayPendingPayments()
    case "4":
        app.handleCheckTransactionStatus()
    case "5":
        app.displayNotifications()
    case "6":
        return
    default:
        fmt.Println("Invalid operation")
    }
}

// authenticateUser handles user authentication
func (app *ClientApp) authenticateUser(clientName string) bool {
    fmt.Printf("Please authenticate user %s\n", clientName)
    fmt.Println("Enter password:")
    password := app.uiManager.GetUserInput()
    
    // Use your existing authentication mechanism
    user := auth.User{
        Username: clientName,
        Password: password,
    }
    
    app.authManager.AuthenticateUsers([]auth.User{user})
    
    if !app.authManager.IsAuthenticated(clientName) {
        fmt.Printf("Authentication failed for user %s\n", clientName)
        return false
    }
    
    // Store credentials for offline use
    app.offlineManager.GetProcessor().StoreCredentials(clientName, password)
    
    fmt.Printf("User %s authenticated successfully!\n", clientName)
    return true
}

// handleCheckBalance checks a client's balance
func (app *ClientApp) handleCheckBalance(clientName string) {
    // Create authenticated context
    md := metadata.Pairs("username", clientName)
    ctx := metadata.NewOutgoingContext(context.Background(), md)

    balResp, err := app.client.GetBalance(ctx, &pb.BalanceRequest{
        ClientName: clientName,
    })
    if err != nil {
        fmt.Printf("Failed to get balance: %v (server may be offline)\n", err)
        return
    }
    fmt.Printf("Balance for %s: %.2f %s\n", clientName, balResp.Balance, balResp.Currency)
}

// handleMakePayment processes a payment
func (app *ClientApp) handleMakePayment(clientName string) {
    fmt.Println("Enter payment amount:")
    amountStr := app.uiManager.GetUserInput()
    amount, err := strconv.ParseFloat(amountStr, 32)
    if err != nil {
        fmt.Printf("Invalid amount: %v\n", err)
        return
    }

    // Use the DisplayTransferTypeMenu instead of direct console output
    app.uiManager.DisplayTransferTypeMenu()
    transferChoice := app.uiManager.GetUserInput()
    
    var receiverName string
    if transferChoice == "2" {
        fmt.Println("Enter receiver's account name:")
        receiverName = app.uiManager.GetUserInput()
    }

    // Get next request ID (but don't increment yet)
    requestID := app.requestCounter.GetNextRequestID(clientName)

    // Create context with username
    md := metadata.Pairs("username", clientName)
    ctx := metadata.NewOutgoingContext(context.Background(), md)

    // Set a timeout for the payment request
    timeoutCtx, cancel := context.WithTimeout(ctx, payment.PaymentTimeout)
    defer cancel()

    paymentReq := &pb.PaymentDetails{
        ClientName:   clientName,
        Amount:       float32(amount),
        ReceiverName: receiverName,
        RequestId:    requestID,  // Add the request ID
    }
    
    fmt.Printf("Making payment with request ID: %s\n", requestID)
    paymentResp, err := app.client.MakePayment(timeoutCtx, paymentReq)

    if err != nil {
        fmt.Printf("Failed to make payment: %v\n", err)
        fmt.Println("Queuing payment for offline processing...")

        // Get transaction ID from metadata after call attempted
        md, ok := metadata.FromOutgoingContext(timeoutCtx)
        var transactionID string
        if ok {
            values := md.Get("transaction_id")
            if len(values) > 0 {
                transactionID = values[0]
            }
        }

        // Queue the payment for offline processing with request ID
        app.offlineManager.QueuePayment(clientName, float32(amount), transactionID, receiverName, requestID)
        fmt.Printf("Payment queued successfully with transaction ID: %s\n", transactionID)
        fmt.Println("It will be processed when connection is restored.")
        return
    }

    // Extract transaction ID from response context or metadata
    md, ok := metadata.FromOutgoingContext(timeoutCtx)
    var transactionID string
    if ok {
        values := md.Get("transaction_id")
        if len(values) > 0 {
            transactionID = values[0]
        }
    }

    if paymentResp.Success {
        app.uiManager.DisplayPaymentDetails(clientName, transactionID, float32(amount), true, "")
        // Only increment the request counter on success
        app.requestCounter.IncrementCounter(clientName)
    } else {
        app.uiManager.DisplayPaymentDetails(clientName, transactionID, float32(amount), false, paymentResp.Message)
        fmt.Println("You can try again with a new transaction.")
    }
}

// displayPendingPayments shows all pending payments
func (app *ClientApp) displayPendingPayments() {
    fmt.Println("\n===== PENDING PAYMENTS =====")
    pendingPayments := app.offlineManager.GetPendingPayments()
    
    if len(pendingPayments) == 0 {
        fmt.Println("No pending payments in queue")
    } else {
        for i, payment := range pendingPayments {
            fmt.Printf("%d. Client: %s, Amount: %.2f, Status: %s, Retry Count: %d, Transaction ID: %s\n", 
                i+1, payment.ClientName, payment.Amount, payment.Status, payment.RetryCount, payment.TransactionID)
        }
    }
    fmt.Println("============================")
}

// handleCheckTransactionStatus checks a specific transaction
func (app *ClientApp) handleCheckTransactionStatus() {
    fmt.Println("Enter transaction ID to check:")
    transactionID := app.uiManager.GetUserInput()
    
    // Use the new method from NotificationManager
    found := app.offlineManager.GetNotifier().DisplayTransactionStatus(transactionID)
    
    if !found {
        fmt.Printf("No information found for transaction ID: %s\n", transactionID)
    }
}

// displayNotifications shows all notifications
func (app *ClientApp) displayNotifications() {
    app.offlineManager.GetNotifier().DisplayNotifications()
}