package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
)

// Transaction represents a pending transaction
type Transaction struct {
	ID     string
	Client string
	Amount float32
	State  string // "prepared", "committed", "aborted"
}

type BankServer struct {
	pb.UnimplementedBankServiceServer
	Name         string
	clients      map[string]string // account number -> client name
	transactions map[string]*Transaction
	balances     map[string]float32 // client name -> balance
	mu           sync.RWMutex
}

// NewBankServer creates a new instance of BankServer with data loaded from files
func NewBankServer(name string) *BankServer {
	server := &BankServer{
		Name:         name,
		transactions: make(map[string]*Transaction),
	}
	
	// Load clients and balances from data files
	server.clients = loadBankClients(name)
	server.balances = loadBalancesFromCommonFile(name)
	
	return server
}

// Load bank clients from a JSON file
func loadBankClients(bankName string) map[string]string {
	// Ensure the data directory exists
	ensureDataDirExists()

	// Try to read the file
	data, err := os.ReadFile("data/bank_clients.json")
	if err != nil {
		log.Fatalf("Failed to read bank clients data: %v", err)
	}

	var allClients map[string]map[string]string
	if err := json.Unmarshal(data, &allClients); err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Get this bank's clients, or return empty map if not found
	clients, ok := allClients[bankName]
	if !ok {
		log.Printf("Warning: No clients found for bank %s", bankName)
		return make(map[string]string)
	}
	
	return clients // account number -> client name
}

// Load client balances from the common balances.json file
func loadBalancesFromCommonFile(bankName string) map[string]float32 {
	balances := make(map[string]float32)
	ensureDataDirExists()
	
	// First get the list of clients for this bank
	clients := loadBankClients(bankName)
	clientSet := make(map[string]bool)
	for _, clientName := range clients {
		clientSet[clientName] = true
	}
	
	// Now load from the common balances file
	data, err := os.ReadFile("data/balances.json")
	if err != nil {
		log.Printf("Failed to read balances file: %v", err)
		return balances
	}
	
	var balanceRecords []struct {
		ClientName string  `json:"clientName"`
		Balance    float32 `json:"balance"`
		Currency   string  `json:"currency"`
	}
	
	if err := json.Unmarshal(data, &balanceRecords); err != nil {
		log.Printf("Failed to unmarshal balances JSON: %v", err)
		return balances
	}
	
	// Only include balances for clients of this bank
	for _, record := range balanceRecords {
		if clientSet[record.ClientName] {
			balances[record.ClientName] = record.Balance
		}
	}
	return balances
}

// Process payment RPC
func (s *BankServer) ProcessPayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	log.Printf("[%s] Processing payment: %v", s.Name, req)
	if req.Amount <= 0 {
		return &pb.PaymentResponse{Success: false, Message: "Invalid amount"}, nil
	}
	
	// Find client name from account number
	clientName := ""
	for accNum, client := range s.clients {
		if accNum == req.AccountNumber {
			clientName = client
			break
		}
	}
	
	if clientName == "" {
		return &pb.PaymentResponse{Success: false, Message: "Account not found"}, nil
	}
	
	// Check balance
	s.mu.Lock()
	defer s.mu.Unlock()
	
	balance, exists := s.balances[clientName]
	if !exists {
		return &pb.PaymentResponse{Success: false, Message: "Client has no balance record"}, nil
	}
	
	if balance < req.Amount {
		return &pb.PaymentResponse{Success: false, Message: "Insufficient funds"}, nil
	}
	
	// Deduct amount
	s.balances[clientName] -= req.Amount
	
	// // Save updated balances
	// go func() {
	// 	if err := s.saveBalances(); err != nil {
	// 		log.Printf("[%s] Error saving balances: %v", s.Name, err)
	// 	}
	// }()
	return &pb.PaymentResponse{Success: true, Message: "Payment processed successfully"}, nil
}

// PrepareDebit implements the prepare phase for debiting an account
func (s *BankServer) PrepareDebit(ctx context.Context, req *pb.TwoPhaseRequest) (*pb.TwoPhaseResponse, error) {
	log.Printf("[%s] Preparing debit for transaction %s, client %s, amount %f", 
		s.Name, req.TransactionId, req.ClientName, req.Amount)
	
	// Check if client exists in this bank
	exists := false
	for _, client := range s.clients {
		if client == req.ClientName {
			exists = true
			break
		}
	}
	
	if !exists {
		return &pb.TwoPhaseResponse{
			Success: false, 
			Message: fmt.Sprintf("Client %s not found in bank %s", req.ClientName, s.Name),
		}, nil
	}
	
	// Lock for transaction processing
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check for sufficient funds
	if s.balances[req.ClientName] < req.Amount {
		return &pb.TwoPhaseResponse{
			Success: false, 
			Message: "Insufficient funds",
		}, nil
	}
	
	// Store transaction in preparing state
	s.transactions[req.TransactionId] = &Transaction{
		ID:     req.TransactionId,
		Client: req.ClientName,
		Amount: req.Amount,
		State:  "prepared",
	}
	
	return &pb.TwoPhaseResponse{Success: true, Message: "Debit prepared"}, nil
}

// PrepareCredit implements the prepare phase for crediting an account
func (s *BankServer) PrepareCredit(ctx context.Context, req *pb.TwoPhaseRequest) (*pb.TwoPhaseResponse, error) {
	log.Printf("[%s] Preparing credit for transaction %s, client %s, amount %f", 
		s.Name, req.TransactionId, req.ClientName, req.Amount)
	
	// Check if client exists in this bank
	exists := false
	for _, client := range s.clients {
		if client == req.ClientName {
			exists = true
			break
		}
	}
	
	if !exists {
		return &pb.TwoPhaseResponse{
			Success: false, 
			Message: fmt.Sprintf("Client %s not found in bank %s", req.ClientName, s.Name),
		}, nil
	}
	
	// Store transaction in preparing state
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.transactions[req.TransactionId] = &Transaction{
		ID:     req.TransactionId,
		Client: req.ClientName,
		Amount: req.Amount,
		State:  "prepared",
	}
	
	return &pb.TwoPhaseResponse{Success: true, Message: "Credit prepared"}, nil
}

// CommitDebit implements the commit phase for debiting an account
func (s *BankServer) CommitDebit(ctx context.Context, req *pb.TwoPhaseRequest) (*pb.TwoPhaseResponse, error) {
	log.Printf("[%s] Committing debit for transaction %s, client %s, amount %f", 
		s.Name, req.TransactionId, req.ClientName, req.Amount)
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if transaction was prepared
	tx, exists := s.transactions[req.TransactionId]
	log.Println(exists,tx.State)
	if !exists || tx.State != "prepared" {
		return &pb.TwoPhaseResponse{
			Success: false, 
			Message: "Transaction not prepared",
		}, nil
	}
	
	// Deduct amount
	s.balances[req.ClientName] -= req.Amount
	
	// Update transaction state
	tx.State = "committed"
	
	// Save updated balances
	// go s.saveBalances()
	
	return &pb.TwoPhaseResponse{Success: true, Message: "Debit committed"}, nil
}

// CommitCredit implements the commit phase for crediting an account
func (s *BankServer) CommitCredit(ctx context.Context, req *pb.TwoPhaseRequest) (*pb.TwoPhaseResponse, error) {
	log.Printf("[%s] Committing credit for transaction %s, client %s, amount %f", 
		s.Name, req.TransactionId, req.ClientName, req.Amount)
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if transaction was prepared
	tx, exists := s.transactions[req.TransactionId]
	if !exists || tx.State != "prepared" {
		return &pb.TwoPhaseResponse{
			Success: false, 
			Message: "Transaction not prepared",
		}, nil
	}
	
	// Add amount
	s.balances[req.ClientName] += req.Amount
	
	// Update transaction state
	tx.State = "committed"
	
	// Save updated balances
	// go s.saveBalances()
	
	return &pb.TwoPhaseResponse{Success: true, Message: "Credit committed"}, nil
}

// Abort implements the abort phase for a transaction
func (s *BankServer) Abort(ctx context.Context, req *pb.TwoPhaseRequest) (*pb.TwoPhaseResponse, error) {
	log.Printf("[%s] Aborting transaction %s", s.Name, req.TransactionId)
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if transaction exists
	tx, exists := s.transactions[req.TransactionId]
	if !exists {
		return &pb.TwoPhaseResponse{
			Success: true, 
			Message: "Transaction not found, considered aborted",
		}, nil
	}
	
	// Update transaction state
	tx.State = "aborted"
	
	return &pb.TwoPhaseResponse{Success: true, Message: "Transaction aborted"}, nil
}

// GetBankName returns the name of this bank
func (s *BankServer) GetBankName(ctx context.Context, req *pb.TwoPhaseRequest) (*pb.TwoPhaseResponse, error) {
	return &pb.TwoPhaseResponse{
		Success: true,
		Message: s.Name,
	}, nil
}

// Helper function to ensure the data directory exists
func ensureDataDirExists() {
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		if err := os.Mkdir("data", 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}
	}
}


// Start the bank server
func startBankServer(name string, port int) {
	server := grpc.NewServer()
	bank := &BankServer{
		Name:         name,
		clients:      loadBankClients(name),
		transactions: make(map[string]*Transaction),
		balances:     loadBalancesFromCommonFile(name),
		mu:           sync.RWMutex{},
	}
	pb.RegisterBankServiceServer(server, bank)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	log.Printf("[%s] Bank server listening on port %d", name, port)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func main() {
	// Ensure data directory exists
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		if err := os.Mkdir("data", 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}
	}
	
	go startBankServer("BankA", 50051)
	go startBankServer("BankB", 50052)

	// Keep main alive
	select {}
}