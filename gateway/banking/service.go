package banking

import (
	"log"

	"github.com/example/gateway/config"
)

// InitializeBankService sets up the banking service with the provided configuration
func InitializeBankService(cfg *config.Config) (BankService, error) {
    // Create a new banking service
    service, err := NewBankService(cfg.BankConnections, cfg.BankClientsFilePath)
    if err != nil {
        log.Printf("Error initializing bank service: %v", err)
        return nil, err
    }

    log.Printf("Banking service initialized with %d bank connections", len(cfg.BankConnections))
    return service, nil
}