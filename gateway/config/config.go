package config

import (
	"time"
)

// TLSConfig holds TLS certificate paths
type TLSConfig struct {
	ServerCertPath string
	ServerKeyPath  string
	CACertPath     string
}

// BankConnection represents a connection to a bank
type BankConnection struct {
	Name    string
	Address string
}


// Config holds the application configuration
type Config struct {
    ServerAddress    string
    UsersFilePath    string
    BalancesFilePath string
    BankClientsFilePath string // Add this to store the path to bank_clients.json
    TLSConfig        TLSConfig
    BankConnections  []BankConnection
    TransactionTTL   time.Duration
    CleanupInterval  time.Duration
    PaymentTimeout   time.Duration
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
    return &Config{
        ServerAddress:      ":50053",
        UsersFilePath:      "data/users.json",
        BalancesFilePath:   "data/balances.json",
        BankClientsFilePath: "data/bank_clients.json", // Add this path
        TLSConfig: TLSConfig{
            ServerCertPath: "certs/server.crt",
            ServerKeyPath:  "certs/server.key",
            CACertPath:     "certs/ca.crt",
        },
        BankConnections: []BankConnection{
            {Name: "BankA", Address: "localhost:50051"},
            {Name: "BankB", Address: "localhost:50052"},
        },
        TransactionTTL:  24 * time.Hour,
        CleanupInterval: 1 * time.Minute,
        PaymentTimeout:  5 * time.Second,
    }
}


func LoadConfig(configPath string) (*Config, error) {
	return DefaultConfig(), nil
}