package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"fmt"
	"encoding/json"

	"github.com/example/gateway/auth"
	"github.com/example/gateway/banking"
	"github.com/example/gateway/config"
	"github.com/example/gateway/interceptors"
	"github.com/example/gateway/payment"
	"github.com/example/gateway/storage"
	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
)

// GatewayServer implements the payment gateway service
type GatewayServer struct {
	authService                 *auth.Service
	paymentService              *payment.Service
	balanceService              *payment.BalanceService
	transactionMgr              *payment.TransactionManager
	bankingService              banking.BankService
	transactionMonitoringService *TransactionMonitoringService
	pb.UnimplementedPaymentGatewayServer
}

func (s *GatewayServer) GetTransactionStatus(ctx context.Context, req *pb.TransactionStatusRequest) (*pb.TransactionStatusResponse, error) {
	log.Printf("GetTransactionStatus called for transaction: %s", req.TransactionId)
	return s.transactionMonitoringService.GetTransactionStatus(ctx, req)
}

// TransactionMonitoringService provides functionality to monitor 2PC transactions
type TransactionMonitoringService struct {
	transactionMgr *payment.TransactionManager
}

// NewTransactionMonitoringService creates a new transaction monitoring service
func NewTransactionMonitoringService(transactionMgr *payment.TransactionManager) *TransactionMonitoringService {
	return &TransactionMonitoringService{
		transactionMgr: transactionMgr,
	}
}

// GetTransactionStatus provides detailed information about a transaction
func (s *TransactionMonitoringService) GetTransactionStatus(ctx context.Context, req *pb.TransactionStatusRequest) (*pb.TransactionStatusResponse, error) {
	txData, exists := s.transactionMgr.GetTransaction(req.TransactionId)
	if !exists {
		return &pb.TransactionStatusResponse{
			TransactionId: req.TransactionId,
			Status:        "not_found",
			Message:       "Transaction not found",
		}, nil
	}

	// Check if it's a two-phase transaction
	if tx, ok := txData.(*payment.TwoPhaseTransactionRecord); ok {
		// Create a detailed status response with 2PC specific information
		stateStr := "unknown"
		switch tx.State {
		case payment.TransactionPreparing:
			stateStr = "preparing"
		case payment.TransactionPrepared:
			stateStr = "prepared"
		case payment.TransactionCommitting:
			stateStr = "committing"
		case payment.TransactionCommitted:
			stateStr = "committed"
		case payment.TransactionAborting:
			stateStr = "aborting"
		case payment.TransactionAborted:
			stateStr = "aborted"
		}

		// Log detailed transaction state
		log.Printf("[2PC-MONITOR] Transaction %s is in state: %s", req.TransactionId, stateStr)
		
		// Get votes using the accessor methods
		prepareVotes := tx.GetPrepareVotes()
		commitVotes := tx.GetCommitVotes()
		
		// Convert votes to string representation
		prepareVotesJSON, _ := json.Marshal(prepareVotes)
		commitVotesJSON, _ := json.Marshal(commitVotes)
		
		return &pb.TransactionStatusResponse{
			TransactionId: req.TransactionId,
			Status:        stateStr,
			Message:       fmt.Sprintf("2PC Transaction in state: %s", stateStr),
			Metadata: map[string]string{
				"type":           "two_phase_commit",
				"sender":         tx.Sender,
				"receiver":       tx.Receiver,
				"amount":         fmt.Sprintf("%.2f", tx.Amount),
				"timestamp":      tx.Timestamp.Format(time.RFC3339),
				"prepare_votes":  string(prepareVotesJSON),
				"commit_votes":   string(commitVotesJSON),
				"elapsed_time":   time.Since(tx.Timestamp).String(),
			},
		}, nil
	}

	// Handle simple transactions
	if tx, ok := txData.(payment.SimpleTransactionRecord); ok {
		return &pb.TransactionStatusResponse{
			TransactionId: req.TransactionId,
			Status:        "completed",
			Message:       "Simple transaction (non-2PC) completed",
			Metadata: map[string]string{
				"type":      "simple",
				"client":    tx.ClientName,
				"amount":    fmt.Sprintf("%.2f", tx.Amount),
				"timestamp": tx.ProcessedAt.Format(time.RFC3339),
				"success":   fmt.Sprintf("%t", tx.Result.Success),
			},
		}, nil
	}

	return &pb.TransactionStatusResponse{
		TransactionId: req.TransactionId,
		Status:        "unknown_type",
		Message:       "Transaction exists but is of unknown type",
	}, nil
}

// NewGatewayServer creates a new payment gateway server instance
func NewGatewayServer(cfg *config.Config) (*GatewayServer, error) {
	// Initialize storage providers
	userStorage, err := storage.NewUserStorage(cfg.UsersFilePath)
	if err != nil {
		return nil, err
	}

	balanceStorage, err := storage.NewBalanceStorage(cfg.BalancesFilePath)
	if err != nil {
		return nil, err
	}

	// Initialize transaction manager with enhanced logging
	transactionMgr := payment.NewTransactionManager(cfg.TransactionTTL, cfg.CleanupInterval)
	
	// Set up logging level for 2PC operations
	log.Printf("Configuring 2PC with verbose logging enabled")

	// Initialize banking service
	bankingService, err := banking.InitializeBankService(cfg)
	if err != nil {
		log.Printf("Warning: Banking service initialization had issues: %v", err)
	}

	// Initialize auth service
	authService := auth.NewService(userStorage)

	// Initialize payment service with 2PC monitoring
	paymentService := payment.NewService(payment.ServiceConfig{
		BalanceStorage: balanceStorage,
		TransactionMgr: transactionMgr,
		BankingService: bankingService,
		PaymentTimeout: 30 * time.Second,
		UseTwoPhase:    true, // Enable 2PC
		LogLevel:       "debug", // Add this field to ServiceConfig
	})

	// Initialize balance service
	balanceService := payment.NewBalanceService(balanceStorage)

	// Initialize transaction monitoring service
	transactionMonitoringService := NewTransactionMonitoringService(transactionMgr)

	return &GatewayServer{
		authService:                authService,
		paymentService:             paymentService,
		balanceService:             balanceService,
		transactionMgr:             transactionMgr,
		bankingService:             bankingService,
		transactionMonitoringService: transactionMonitoringService,
	}, nil
}

func (s *GatewayServer) ListActiveTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	log.Printf("ListActiveTransactions called")
	
	transactions := s.transactionMgr.GetActiveTransactions()
	
	response := &pb.ListTransactionsResponse{
		Transactions: make([]*pb.TransactionInfo, 0, len(transactions)),
	}
	
	for txID, txData := range transactions {
		// Extract transaction info based on type
		if tx, ok := txData.(*payment.TwoPhaseTransactionRecord); ok {
			// Get transaction state
			stateStr := "unknown"
			switch tx.State {
			case payment.TransactionPreparing:
				stateStr = "preparing"
			case payment.TransactionPrepared:
				stateStr = "prepared"
			case payment.TransactionCommitting:
				stateStr = "committing"
			case payment.TransactionCommitted:
				stateStr = "committed"
			case payment.TransactionAborting:
				stateStr = "aborting"
			case payment.TransactionAborted:
				stateStr = "aborted"
			}
			
			// Log transaction details
			log.Printf("[2PC-MONITOR] Active transaction %s: %s -> %s, amount: %.2f, state: %s, age: %s",
				txID, tx.Sender, tx.Receiver, tx.Amount, stateStr, time.Since(tx.Timestamp))
			
			info := &pb.TransactionInfo{
				Id:        txID,
				Type:      "two_phase_commit",
				Sender:    tx.Sender,
				Receiver:  tx.Receiver,
				Amount:    tx.Amount,
				Status:    stateStr,
				Timestamp: tx.Timestamp.Format(time.RFC3339),
			}
			response.Transactions = append(response.Transactions, info)
		}
	}
	
	return response, nil
}

// RunServer starts the gRPC server
func RunServer(server *GatewayServer, cfg *config.Config) error {
	// Set up TLS credentials
	tlsCredentials, err := config.SetupTLSCredentials(cfg.TLSConfig)
	if err != nil {
		return err
	}

	// Initialize interceptor chain
	interceptorChain := interceptors.ChainInterceptors(
		interceptors.RecoveryInterceptor(),
		interceptors.LoggingInterceptor(),
		auth.NewAuthInterceptor(server.authService),
	)

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.Creds(tlsCredentials),
		grpc.UnaryInterceptor(interceptorChain),
	)

	// Register server implementation
	pb.RegisterPaymentGatewayServer(grpcServer, server)

	// Start listening
	lis, err := net.Listen("tcp", cfg.ServerAddress)
	if err != nil {
		return err
	}

	// Set up graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Start server in a goroutine
	errChan := make(chan error)
	go func() {
		log.Printf("Payment Gateway server running on %s", cfg.ServerAddress)
		errChan <- grpcServer.Serve(lis)
	}()

	// Wait for interrupt or server error
	select {
	case err := <-errChan:
		return err
	case <-stopChan:
		log.Println("Shutting down server...")
		grpcServer.GracefulStop()
		log.Println("Server stopped gracefully")
		return nil
	}
}

// Implement the required gRPC methods

// Authenticate handles user authentication
func (s *GatewayServer) Authenticate(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	return s.authService.Authenticate(ctx, req)
}

// MakePayment processes a payment
func (s *GatewayServer) MakePayment(ctx context.Context, req *pb.PaymentDetails) (*pb.PaymentResponse, error) {
	return s.paymentService.ProcessPayment(ctx, req)
}

// GetBalance retrieves client balance
func (s *GatewayServer) GetBalance(ctx context.Context, req *pb.BalanceRequest) (*pb.BalanceResponse, error) {
	return s.balanceService.GetBalance(ctx, req)
}

// Main function to make this file runnable
func main() {
	// Load configuration (using default config if config.json not available)
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Printf("Warning: Failed to load configuration from file: %v", err)
		log.Println("Using default configuration...")
	}
	
	// Create gateway server
	server, err := NewGatewayServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	
	// Run the server
	if err := RunServer(server, cfg); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}