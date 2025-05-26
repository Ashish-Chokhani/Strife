// gateway/payment/processor.go

package payment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/example/gateway/auth"
	"github.com/example/gateway/banking"
	pb "github.com/example/protofiles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TransactionState represents the state of a transaction
type TransactionState int

const (
	// Simple transaction states
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"

	// 2PC transaction states
	TransactionPreparing TransactionState = iota
	TransactionPrepared
	TransactionCommitting
	TransactionCommitted
	TransactionAborting
	TransactionAborted
)

// BalanceStorage defines the interface for balance storage operations
type BalanceStorage interface {
	GetBalance(clientName string) (Balance, bool)
	UpdateBalance(clientName string, balance Balance) error
	SaveBalances() error
}

// Service handles payment processing
type Service struct {
	balanceStorage  BalanceStorage
	transactionMgr  *TransactionManager
	bankingService  banking.BankService
	paymentTimeout  time.Duration
	usesTwoPhase    bool // Flag to determine if 2PC should be used
}

// ServiceConfig holds configuration for the payment service
type ServiceConfig struct {
	BalanceStorage BalanceStorage
	TransactionMgr *TransactionManager
	BankingService banking.BankService
	PaymentTimeout time.Duration
	UseTwoPhase    bool // Flag to enable/disable 2PC
	LogLevel       string // Add this field to support debug logging
}

// NewService creates a new payment service
func NewService(config ServiceConfig) *Service {
	return &Service{
		balanceStorage: config.BalanceStorage,
		transactionMgr: config.TransactionMgr,
		bankingService: config.BankingService,
		paymentTimeout: config.PaymentTimeout,
		usesTwoPhase:   config.UseTwoPhase,
	}
}

// ProcessPayment handles payment requests, using either simple processing or 2PC
func (s *Service) ProcessPayment(ctx context.Context, req *pb.PaymentDetails) (*pb.PaymentResponse, error) {
	log.Printf("ProcessPayment called for client: %s, amount: %f", req.ClientName, req.Amount)


	// Extract username from metadata
	username, err := auth.ExtractUsername(ctx)
	if err != nil {
		return nil, err
	}

	// Check if the user is making a payment for their own account
	if username != req.ClientName {
		return &pb.PaymentResponse{
			Success: false,
			Message: "You can only make payments for your own account",
		}, nil
	}

	// Extract transaction ID from metadata
	transactionID, err := extractTransactionID(ctx)
	if err != nil {
		return nil, err
	}

	log.Println(username,transactionID)

	// Check for existing transaction with this ID
	if record, exists := s.transactionMgr.GetTransaction(transactionID); exists {
		log.Printf("Found existing transaction %s for client %s, returning cached result",
			transactionID, req.ClientName)
		
		// Handle different record types based on whether we're using 2PC or not
		if s.usesTwoPhase {
			if tpcRecord, ok := record.(*TwoPhaseTransactionRecord); ok {
				// Only return success if transaction is fully committed
				if tpcRecord.State == TransactionCommitted {
					return &pb.PaymentResponse{
						Success: true,
						Message: "Transaction already processed successfully",
						TransactionId: transactionID,
						Amount: req.Amount,
					}, nil
				} else if tpcRecord.State == TransactionAborted {
					return &pb.PaymentResponse{
						Success: false,
						Message: "Transaction was previously aborted",
						TransactionId: transactionID,
					}, nil
				}
				// For in-progress transactions, continue processing
			}
		} else {
			if simpleRecord, ok := record.(SimpleTransactionRecord); ok {
				return simpleRecord.Result, nil
			}
		}
	}

	// Get account balance
	balance, exists := s.balanceStorage.GetBalance(req.ClientName)
	if !exists {
		log.Printf("No balance information for client: %s", req.ClientName)
		return &pb.PaymentResponse{
			Success: false,
			Message: "No balance information for client",
		}, nil
	}

	// Check sufficient funds
	if balance.Amount < req.Amount {
		log.Printf("Insufficient funds for client: %s", req.ClientName)
		return &pb.PaymentResponse{
			Success: false,
			Message: "Insufficient funds",
		}, nil
	}
	// Process payment using either 2PC or simple processing
	if s.usesTwoPhase && req.ReceiverName != "" {
		return s.processTwoPhasePayment(ctx, req, transactionID, balance)
	} else {
		return s.processSimplePayment(ctx, req, transactionID, balance)
	}
}

// processSimplePayment processes a payment using the original simple method
func (s *Service) processSimplePayment(ctx context.Context, req *pb.PaymentDetails, transactionID string, balance Balance) (*pb.PaymentResponse, error) {
	// Get bank client for this user
	bankClient, err := s.bankingService.GetBankClient(req.ClientName)
	if err != nil {
		log.Printf("Bank not available for client: %s", req.ClientName)
		return &pb.PaymentResponse{
			Success: false,
			Message: "Bank not available",
		}, nil
	}

	// Create a timeout context for the bank call
	timeoutCtx, cancel := context.WithTimeout(ctx, s.paymentTimeout)
	defer cancel()

	// Create bank request with transaction ID
	bankCtx := metadata.AppendToOutgoingContext(timeoutCtx, "transaction_id", transactionID)

	// Get account number for this client
	accountNumber, err := s.bankingService.GetAccountNumber(req.ClientName)
	if err != nil {
		return &pb.PaymentResponse{
			Success: false,
			Message: fmt.Sprintf("Account lookup error: %v", err),
		}, nil
	}

	// Process payment through bank
	resp, err := bankClient.ProcessPayment(bankCtx, &pb.PaymentRequest{
		AccountNumber: accountNumber,
		Amount:        req.Amount,
		TransactionId: transactionID,
	})

	if err != nil {
		log.Printf("Error processing payment: %v", err)
		return &pb.PaymentResponse{
			Success: false,
			Message: fmt.Sprintf("Payment processing error: %v", err),
		}, nil
	}

	log.Printf("Bank response: %v", resp)

	// Prepare the response
	paymentResponse := &pb.PaymentResponse{
		Success: resp.Success,
		Message: resp.Message,
	}

	// Update balance if payment was successful
	if resp.Success {
		balance.Amount -= req.Amount
		if err := s.balanceStorage.UpdateBalance(req.ClientName, balance); err != nil {
			log.Printf("Error updating balance: %v", err)
			// Continue anyway since payment was successful at the bank
		} else {
			log.Printf("Updated balance for %s: %f", req.ClientName, balance.Amount)

			// Save balances to file
			go func() {
				if err := s.balanceStorage.SaveBalances(); err != nil {
					log.Printf("Failed to save balances after payment: %v", err)
				} else {
					log.Printf("Successfully saved account balances after payment for %s", req.ClientName)
				}
			}()
		}

		// Store transaction record for idempotency
		s.transactionMgr.StoreSimpleTransaction(SimpleTransactionRecord{
			TransactionID: transactionID,
			ClientName:    req.ClientName,
			Amount:        req.Amount,
			ProcessedAt:   time.Now(),
			Result:        paymentResponse,
		})
	}

	return paymentResponse, nil
}

// processTwoPhasePayment processes a payment using two-phase commit protocol
func (s *Service) processTwoPhasePayment(ctx context.Context, req *pb.PaymentDetails, transactionID string, balance Balance) (*pb.PaymentResponse, error) {
	// Check if receiver is specified
	if req.ReceiverName == "" {
		return &pb.PaymentResponse{
			Success: false,
			Message: "Receiver name is required for two-phase commit payment",
		}, nil
	}

	// Check if there's a running transaction with this ID
	if s.transactionMgr.HasActiveTransaction(transactionID) {
		txData, _ := s.transactionMgr.GetTransaction(transactionID)
		if tx, ok := txData.(*TwoPhaseTransactionRecord); ok {
			// Use the pointer directly instead of copying
			if tx.State == TransactionPreparing || tx.State == TransactionCommitting {
				return &pb.PaymentResponse{
					Success: false,
					Message: "Transaction already in progress",
				}, nil
			}
		}
	}

	// Create transaction record
	transaction := &TwoPhaseTransactionRecord{
		ID:           transactionID,
		Sender:       req.ClientName,
		Receiver:     req.ReceiverName,
		Amount:       req.Amount,
		State:        TransactionPreparing,
		PrepareVotes: make(map[string]bool),
		CommitVotes:  make(map[string]bool),
		Timestamp:    time.Now(),
		Timeout:      s.paymentTimeout,
	}

	// Register transaction
	s.transactionMgr.StoreTwoPhaseTransaction(transactionID, transaction)

	// Set up context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, s.paymentTimeout)
	defer cancel()

	// Run two-phase commit protocol
	success, message := s.runTwoPhaseCommit(timeoutCtx, transaction)
	
	if !success {
		// Clean up the transaction if it failed
		s.transactionMgr.CleanupTransaction(transactionID)
	}

	return &pb.PaymentResponse{
		Success:       success,
		Message:       message,
		TransactionId: transactionID,
		Amount:        req.Amount,
		Currency:      balance.Currency,
	}, nil
}

// runTwoPhaseCommit executes the two-phase commit protocol
func (s *Service) runTwoPhaseCommit(ctx context.Context, tx *TwoPhaseTransactionRecord) (bool, string) {
	// PHASE 1: Prepare
	prepareSuccess, prepareMsg := s.preparePhase(ctx, tx)
	if !prepareSuccess {
		return false, prepareMsg
	}

	// PHASE 2: Commit or Abort
	commitSuccess, commitMsg := s.commitPhase(ctx, tx)
	if !commitSuccess {
		return false, commitMsg
	}

	return true, "Payment processed successfully"
}

// preparePhase executes the prepare phase of 2PC
func (s *Service) preparePhase(ctx context.Context, tx *TwoPhaseTransactionRecord) (bool, string) {
	log.Printf("[PREPARE] Starting prepare phase for transaction %s", tx.ID)
	
	// Get bank connections for sender and receiver
	senderBank, err := s.bankingService.GetBankForClient(tx.Sender)
	if err != nil {
		s.abortTransaction(tx, "Failed to determine sender's bank")
		return false, "Failed to determine sender's bank: " + err.Error()
	}

	receiverBank, err := s.bankingService.GetBankForClient(tx.Receiver)
	if err != nil {
		s.abortTransaction(tx, "Failed to determine receiver's bank")
		return false, "Failed to determine receiver's bank: " + err.Error()
	}

	// Update transaction state
	tx.mu.Lock()
	tx.State = TransactionPreparing
	tx.mu.Unlock()

	// Prepare phase: ask banks to lock resources and vote
	var wg sync.WaitGroup
	var prepareErr error
	var errMu sync.Mutex

	// Ask sender bank to prepare
	wg.Add(1)
	go func() {
		defer wg.Done()
		prepared, err := senderBank.PrepareDebit(ctx, tx.ID, tx.Sender, tx.Amount)
		
		tx.mu.Lock()
		tx.PrepareVotes[senderBank.GetName()] = prepared
		tx.mu.Unlock()
		
		if err != nil || !prepared {
			errMu.Lock()
			prepareErr = errors.New("sender bank declined: " + err.Error())
			errMu.Unlock()
		}
	}()

	// Ask receiver bank to prepare
	wg.Add(1)
	go func() {
		defer wg.Done()
		prepared, err := receiverBank.PrepareCredit(ctx, tx.ID, tx.Receiver, tx.Amount)
		
		tx.mu.Lock()
		tx.PrepareVotes[receiverBank.GetName()] = prepared
		tx.mu.Unlock()
		
		if err != nil || !prepared {
			errMu.Lock()
			if prepareErr == nil {
				prepareErr = errors.New("receiver bank declined: " + err.Error())
			}
			errMu.Unlock()
		}
	}()

	// Wait for all votes or timeout
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		// All votes received
	case <-ctx.Done():
		// Timeout
		s.abortTransaction(tx, "Prepare phase timed out")
		return false, "Transaction timed out during prepare phase"
	}

	// Check if any errors occurred
	if prepareErr != nil {
		s.abortTransaction(tx, prepareErr.Error())
		return false, "Prepare phase failed: " + prepareErr.Error()
	}

	// Check all votes
	tx.mu.RLock()
	allPrepared := tx.PrepareVotes[senderBank.GetName()] && tx.PrepareVotes[receiverBank.GetName()]
	tx.mu.RUnlock()

	if !allPrepared {
		s.abortTransaction(tx, "Not all participants voted to commit")
		return false, "One or more banks declined the transaction"
	}

	// Update transaction state to prepared
	tx.mu.Lock()
	tx.State = TransactionPrepared
	tx.mu.Unlock()

	log.Printf("[PREPARE] Prepare phase completed successfully for transaction %s", tx.ID)
	return true, "Prepare phase successful"
}

// commitPhase executes the commit phase of 2PC
func (s *Service) commitPhase(ctx context.Context, tx *TwoPhaseTransactionRecord) (bool, string) {
	log.Printf("[COMMIT] Starting commit phase for transaction %s", tx.ID)
	
	// Get bank connections for sender and receiver
	senderBank, err := s.bankingService.GetBankForClient(tx.Sender)
	if err != nil {
		s.abortTransaction(tx, "Failed to determine sender's bank")
		return false, "Failed to determine sender's bank during commit: " + err.Error()
	}

	receiverBank, err := s.bankingService.GetBankForClient(tx.Receiver)
	if err != nil {
		s.abortTransaction(tx, "Failed to determine receiver's bank")
		return false, "Failed to determine receiver's bank during commit: " + err.Error()
	}

	// Update transaction state
	tx.mu.Lock()
	tx.State = TransactionCommitting
	tx.mu.Unlock()

	// Commit phase: tell banks to commit the transaction
	var wg sync.WaitGroup
	var commitErr error
	var errMu sync.Mutex

	// Tell sender bank to commit
	wg.Add(1)
	go func() {
		defer wg.Done()
		committed, err := senderBank.CommitDebit(ctx, tx.ID, tx.Sender, tx.Amount)
		
		tx.mu.Lock()
		tx.CommitVotes[senderBank.GetName()] = committed
		tx.mu.Unlock()
		
		if err != nil || !committed {
			errMu.Lock()
			commitErr = errors.New("sender bank commit failed: " + err.Error())
			errMu.Unlock()
		}
	}()

	// Tell receiver bank to commit
	wg.Add(1)
	go func() {
		defer wg.Done()
		committed, err := receiverBank.CommitCredit(ctx, tx.ID, tx.Receiver, tx.Amount)
		
		tx.mu.Lock()
		tx.CommitVotes[receiverBank.GetName()] = committed
		tx.mu.Unlock()
		
		if err != nil || !committed {
			errMu.Lock()
			if commitErr == nil {
				commitErr = errors.New("receiver bank commit failed: " + err.Error())
			}
			errMu.Unlock()
		}
	}()

	// Wait for all commits or timeout
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		// All commits received
	case <-ctx.Done():
		// Timeout
		s.abortTransaction(tx, "Commit phase timed out")
		return false, "Transaction timed out during commit phase"
	}

	// Check if any errors occurred
	if commitErr != nil {
		// This is a critical situation since we're in commit phase
		// We should retry until success or manual intervention
		log.Printf("[CRITICAL] Commit phase partially failed: %v", commitErr)
		return false, "Commit phase failed: " + commitErr.Error()
	}

	// Check all commits
	tx.mu.RLock()
	allCommitted := tx.CommitVotes[senderBank.GetName()] && tx.CommitVotes[receiverBank.GetName()]
	tx.mu.RUnlock()

	if !allCommitted {
		log.Printf("[CRITICAL] Not all participants confirmed commit for transaction %s", tx.ID)
		return false, "One or more banks failed to commit the transaction"
	}

	// Update transaction state to committed
	tx.mu.Lock()
	tx.State = TransactionCommitted
	tx.mu.Unlock()

	// Update local balances
	err = s.updateTwoPhaseBalances(tx.Sender, tx.Receiver, tx.Amount)
	if err != nil {
		log.Printf("[WARNING] Failed to update local balances: %v", err)
		// We don't fail the transaction here as the banks have already committed
	}

	log.Printf("[COMMIT] Commit phase completed successfully for transaction %s", tx.ID)
	return true, "Payment successfully processed"
}

// abortTransaction handles transaction abort
func (s *Service) abortTransaction(tx *TwoPhaseTransactionRecord, reason string) {
	log.Printf("[ABORT] Aborting transaction %s: %s", tx.ID, reason)
	
	tx.mu.Lock()
	tx.State = TransactionAborting
	tx.mu.Unlock()

	// Get bank connections for sender and receiver
	senderBank, err := s.bankingService.GetBankForClient(tx.Sender)
	if err != nil {
		log.Printf("[ERROR] Failed to get sender bank during abort: %v", err)
	} else {
		// Tell sender bank to abort
		_, err := senderBank.Abort(context.Background(), tx.ID)
		if err != nil {
			log.Printf("[ERROR] Failed to abort transaction on sender bank: %v", err)
		}
	}

	receiverBank, err := s.bankingService.GetBankForClient(tx.Receiver)
	if err != nil {
		log.Printf("[ERROR] Failed to get receiver bank during abort: %v", err)
	} else {
		// Tell receiver bank to abort
		_, err := receiverBank.Abort(context.Background(), tx.ID)
		if err != nil {
			log.Printf("[ERROR] Failed to abort transaction on receiver bank: %v", err)
		}
	}

	tx.mu.Lock()
	tx.State = TransactionAborted
	tx.mu.Unlock()

	log.Printf("[ABORT] Transaction %s aborted", tx.ID)
}

// updateTwoPhaseBalances updates the local balances after a successful transaction
func (s *Service) updateTwoPhaseBalances(sender, receiver string, amount float32) error {
    // Get sender balance
    senderBalance, exists := s.balanceStorage.GetBalance(sender)
    if !exists {
        return fmt.Errorf("sender balance not found")
    }

    // Update sender balance
    senderBalance.Amount -= amount
    err := s.balanceStorage.UpdateBalance(sender, senderBalance)
    if err != nil {
        return err
    }

    // Get receiver balance
    receiverBalance, exists := s.balanceStorage.GetBalance(receiver)
    if !exists {
        return fmt.Errorf("receiver balance not found")
    }

    // Update receiver balance
    receiverBalance.Amount += amount
    err = s.balanceStorage.UpdateBalance(receiver, receiverBalance)
    if err != nil {
        return err
    }

    log.Printf("[DEBUG] Saving balances - Sender %s: %.2f %s, Receiver %s: %.2f %s", 
        sender, senderBalance.Amount, senderBalance.Currency, 
        receiver, receiverBalance.Amount, receiverBalance.Currency)

    // Save balances with retry
    const maxRetries = 3
    for attempt := 0; attempt < maxRetries; attempt++ {
        err = s.balanceStorage.SaveBalances()
        if err == nil {
            break
        }
        log.Printf("[WARNING] Failed to save balances, attempt %d: %v", attempt+1, err)
        time.Sleep(100 * time.Millisecond)
    }
    
    if err != nil {
        log.Printf("[ERROR] Failed to save balances after %d attempts: %v", maxRetries, err)
        return err
    }

    return nil
}

// extractTransactionID extracts the transaction ID from context metadata
func extractTransactionID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.InvalidArgument, "missing context metadata")
	}

	transactionIDs := md.Get("transaction_id")
	if len(transactionIDs) == 0 {
		return "", status.Errorf(codes.InvalidArgument, "transaction_id required for idempotency")
	}

	return transactionIDs[0], nil
}