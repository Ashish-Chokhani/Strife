package banking

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"io/ioutil"
	"time"
	"encoding/json"

	"github.com/example/gateway/config"
	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// BankService defines the interface for interacting with bank services
type BankService interface {
	GetBankClient(clientName string) (pb.BankServiceClient, error)
	GetAccountNumber(clientName string) (string, error)
	GetBankForClient(clientName string) (BankClient, error)
	// RegisterClient(clientName, bankName, accountNumber string) error
}

// BankClient defines the interface for a bank participating in 2PC
type BankClient interface {
	PrepareDebit(ctx context.Context, txID string, clientName string, amount float32) (bool, error)
	PrepareCredit(ctx context.Context, txID string, clientName string, amount float32) (bool, error)
	CommitDebit(ctx context.Context, txID string, clientName string, amount float32) (bool, error)
	CommitCredit(ctx context.Context, txID string, clientName string, amount float32) (bool, error)
	Abort(ctx context.Context, txID string) (bool, error)
	GetName() string
}

// Bank represents a bank and its connection info
type Bank struct {
	Name      string
	Address   string
	Port      int
	Clients   map[string]string // clientName -> accountNumber
	client    pb.BankServiceClient
	connected bool
	mu        sync.RWMutex
}

// DefaultBankService is the implementation of BankService
type DefaultBankService struct {
    banks          map[string]*Bank // bankName -> Bank
    bankClientsPath string
    mu             sync.RWMutex
}

// NewBankService creates a new banking service from configuration
func NewBankService(bankConnections []config.BankConnection, bankClientsPath string) (*DefaultBankService, error) {
    service := &DefaultBankService{
        banks: make(map[string]*Bank),
        bankClientsPath: bankClientsPath, // Store the path
    }

    log.Printf("Initializing connections to %d banks", len(bankConnections))

    // Initialize banks from configuration
    for _, conn := range bankConnections {
        // Extract host and port from address
        var host string
        var port int
        _, err := fmt.Sscanf(conn.Address, "%s:%d", &host, &port)
        if err != nil {
            // Default to localhost if format doesn't match
            host = conn.Address
            port = 0 // Will be parsed from address later
        }

        service.banks[conn.Name] = &Bank{
            Name:      conn.Name,
            Address:   host,
            Port:      port,
            Clients:   make(map[string]string),
            connected: false,
        }
        log.Printf("Added bank %s at %s", conn.Name, conn.Address)
    }

    // Load client mappings from the data file
    service.loadClientMappings()

    return service, nil
}

// loadClientMappings loads the mappings of clients to banks from data files
func (s *DefaultBankService) loadClientMappings() {
    // Read from bank_clients.json using the path from the service
    data, err := ioutil.ReadFile(s.bankClientsPath)
    if err != nil {
        log.Printf("Error reading bank clients file %s: %v", s.bankClientsPath, err)
        return
    }
    
    var clientMappings map[string]map[string]string
    if err := json.Unmarshal(data, &clientMappings); err != nil {
        log.Printf("Error parsing bank clients file: %v", err)
        return
    }

    for bankName, clients := range clientMappings {
        bank, exists := s.banks[bankName]
        if !exists {
            log.Printf("Warning: Bank %s not found in configuration", bankName)
            continue
        }

        bank.mu.Lock()
        for accountNumber, clientName := range clients {
            bank.Clients[clientName] = accountNumber
            log.Printf("Loaded client %s with account %s at bank %s", 
                clientName, accountNumber, bankName)
        }
        bank.mu.Unlock()
    }
}

// GetBankClient returns a gRPC client for the bank that serves this client
func (s *DefaultBankService) GetBankClient(clientName string) (pb.BankServiceClient, error) {
	bankName, err := s.getBankNameForClient(clientName)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	bank, exists := s.banks[bankName]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("bank %s not found", bankName)
	}

	// Connect to bank if not already connected
	bank.mu.Lock()
	defer bank.mu.Unlock()

	if !bank.connected || bank.client == nil {
		address := bank.Address
		conn, err := grpc.Dial(address, 
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(), 
			grpc.WithTimeout(3*time.Second))
		
		if err != nil {
			log.Printf("WARNING1: Failed to connect to %s: %v", bankName, err)
			// Create non-blocking connection to prevent hanging
			conn, _ = grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
			return nil, fmt.Errorf("failed to connect to bank %s: %w", bankName, err)
		}
		bank.client = pb.NewBankServiceClient(conn)
		bank.connected = true
		log.Printf("Successfully connected to bank %s at %s", bankName, address)
	}

	return bank.client, nil
}

// GetAccountNumber returns the account number for a client
func (s *DefaultBankService) GetAccountNumber(clientName string) (string, error) {
	bankName, err := s.getBankNameForClient(clientName)
	if err != nil {
		return "", err
	}

	s.mu.RLock()
	bank, exists := s.banks[bankName]
	s.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("bank %s not found", bankName)
	}

	bank.mu.RLock()
	accountNumber, exists := bank.Clients[clientName]
	bank.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("account not found for client %s", clientName)
	}

	return accountNumber, nil
}

// GetBankForClient returns a BankClient for the bank that serves this client
func (s *DefaultBankService) GetBankForClient(clientName string) (BankClient, error) {
	bankName, err := s.getBankNameForClient(clientName)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	bank, exists := s.banks[bankName]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("bank %s not found", bankName)
	}

	// Connect to bank if not already connected
	bank.mu.Lock()
	defer bank.mu.Unlock()

	if !bank.connected || bank.client == nil {
		address := fmt.Sprintf("%s", bank.Address)
		conn, err := grpc.Dial(address, 
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(), 
			grpc.WithTimeout(3*time.Second))
			
		if err != nil {
			log.Printf("WARNING2: Failed to connect to %s: %v", bankName, err)
			return nil, fmt.Errorf("failed to connect to bank %s: %w", bankName, err)
		}
		bank.client = pb.NewBankServiceClient(conn)
		bank.connected = true
		log.Printf("Successfully connected to bank %s for client operations", bankName)
	}

	return &GrpcBankClient{
		client: bank.client,
		name:   bank.Name,
	}, nil
}

// getBankNameForClient returns the name of the bank for a client
func (s *DefaultBankService) getBankNameForClient(clientName string) (string, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // Iterate through all banks to find which one has this client
    for bankName, bank := range s.banks {
        bank.mu.RLock()
        _, exists := bank.Clients[clientName]
        bank.mu.RUnlock()
        
        if exists {
            return bankName, nil
        }
    }
    
    return "", fmt.Errorf("no bank found for client %s", clientName)
}

// GrpcBankClient is an implementation of BankClient using gRPC
type GrpcBankClient struct {
	client pb.BankServiceClient
	name   string
}

// PrepareDebit prepares a debit transaction
func (c *GrpcBankClient) PrepareDebit(ctx context.Context, txID string, clientName string, amount float32) (bool, error) {
	resp, err := c.client.PrepareDebit(ctx, &pb.TwoPhaseRequest{
		TransactionId: txID,
		ClientName:    clientName,
		Amount:        amount,
	})
	if err != nil {
		return false, err
	}
	if !resp.Success {
		return false, errors.New(resp.Message)
	}
	return resp.Success, nil
}

// PrepareCredit prepares a credit transaction
func (c *GrpcBankClient) PrepareCredit(ctx context.Context, txID string, clientName string, amount float32) (bool, error) {
	resp, err := c.client.PrepareCredit(ctx, &pb.TwoPhaseRequest{
		TransactionId: txID,
		ClientName:    clientName,
		Amount:        amount,
	})
	if err != nil {
		return false, err
	}
	if !resp.Success {
		return false, errors.New(resp.Message)
	}
	return resp.Success, nil
}

// CommitDebit commits a debit transaction
func (c *GrpcBankClient) CommitDebit(ctx context.Context, txID string, clientName string, amount float32) (bool, error) {
	resp, err := c.client.CommitDebit(ctx, &pb.TwoPhaseRequest{
		TransactionId: txID,
		ClientName:    clientName,
		Amount:        amount,
	})
	if err != nil {
		return false, err
	}
	if !resp.Success {
		return false, errors.New(resp.Message)
	}
	return resp.Success, nil
}

// CommitCredit commits a credit transaction
func (c *GrpcBankClient) CommitCredit(ctx context.Context, txID string, clientName string, amount float32) (bool, error) {
	resp, err := c.client.CommitCredit(ctx, &pb.TwoPhaseRequest{
		TransactionId: txID,
		ClientName:    clientName,
		Amount:        amount,
	})
	if err != nil {
		return false, err
	}
	if !resp.Success {
		return false, errors.New(resp.Message)
	}
	return resp.Success, nil
}

// Abort aborts a transaction
func (c *GrpcBankClient) Abort(ctx context.Context, txID string) (bool, error) {
	resp, err := c.client.Abort(ctx, &pb.TwoPhaseRequest{
		TransactionId: txID,
	})
	if err != nil {
		return false, err
	}
	if !resp.Success {
		return false, errors.New(resp.Message)
	}
	return resp.Success, nil
}

// GetName returns the name of the bank
func (c *GrpcBankClient) GetName() string {
	return c.name
}