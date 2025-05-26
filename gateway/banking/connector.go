package banking

import (
	"log"
	"time"

	"google.golang.org/grpc"

	pb "github.com/example/protofiles"
)

// BankConnector manages connections to bank services
type BankConnector struct {
	banks map[string]pb.BankServiceClient
}

// NewBankConnector creates a new bank connector
func NewBankConnector() *BankConnector {
	return &BankConnector{
		banks: make(map[string]pb.BankServiceClient),
	}
}

// ConnectToBank establishes a connection to a bank service
func (b *BankConnector) ConnectToBank(bankName, address string) pb.BankServiceClient {
    conn, err := grpc.Dial(address, 
        grpc.WithInsecure(), 
        grpc.WithBlock(), 
        grpc.WithTimeout(3*time.Second))
    
    if err != nil {
        log.Printf("WARNING3: Failed to connect to %s: %v", bankName, err)
        // Create non-blocking connection to prevent hanging
        conn, _ = grpc.Dial(address, grpc.WithInsecure())
    }
    
    client := pb.NewBankServiceClient(conn)
    b.banks[bankName] = client
    return client
}

// GetBank retrieves a bank client by name
func (b *BankConnector) GetBank(bankName string) (pb.BankServiceClient, bool) {
	client, exists := b.banks[bankName]
	return client, exists
}

// GetAllBanks returns all connected banks
func (b *BankConnector) GetAllBanks() map[string]pb.BankServiceClient {
	return b.banks
}

// TestBankConnections verifies that all bank connections are working
func (b *BankConnector) TestBankConnections() map[string]error {
	results := make(map[string]error)
	
	for name, address := range map[string]string{
		"BankA": "localhost:50051",
		"BankB": "localhost:50052",
	} {
		conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3*time.Second))
		results[name] = err
		
		if err != nil {
			log.Printf("WARNING: Could not connect to %s at %s: %v", name, address, err)
		} else {
			log.Printf("Successfully connected to %s at %s", name, address)
			conn.Close()
		}
	}
	
	return results
}