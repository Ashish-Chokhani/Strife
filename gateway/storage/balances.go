package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/example/gateway/payment"
)

// AccountBalance represents a client's account balance
type AccountBalance struct {
    Balance  float32
    Currency string
}

// BalanceData is used for JSON serialization
type BalanceData struct {
    ClientName string  `json:"clientName"`
    Balance    float32 `json:"balance"`
    Currency   string  `json:"currency"`
}

// BalanceStorage manages account balances
type BalanceStorage struct {
	filePath        string
	accountBalances map[string]AccountBalance
	mu              sync.Mutex
}

// NewBalanceStorage creates a new balance storage
func NewBalanceStorage(filePath string) (*BalanceStorage, error) {
	storage := &BalanceStorage{
		filePath:        filePath,
		accountBalances: make(map[string]AccountBalance),
	}
	
	if err := storage.LoadBalances(); err != nil {
		return nil, err
	}
	
	return storage, nil
}

// LoadBalances loads balance data from file
func (s *BalanceStorage) LoadBalances() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    data, err := ioutil.ReadFile(s.filePath)
    if err != nil {
        return err
    }

    var balances []BalanceData
    if err := json.Unmarshal(data, &balances); err != nil {
        return err
    }

    balanceMap := make(map[string]AccountBalance)
    for _, balance := range balances {
        balanceMap[balance.ClientName] = AccountBalance{
            Balance:  balance.Balance,
            Currency: balance.Currency,
        }
    }
    
    s.accountBalances = balanceMap
    return nil
}

// GetBalance retrieves a client's balance - Updated to match payment.BalanceStorage interface
func (s *BalanceStorage) GetBalance(clientName string) (payment.Balance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	accountBalance, exists := s.accountBalances[clientName]
	if !exists {
		return payment.Balance{}, false
	}
	
	return payment.Balance{
		Amount:   accountBalance.Balance,
		Currency: accountBalance.Currency,
	}, exists
}

// UpdateBalance updates a client's balance - Updated to match payment.BalanceStorage interface
func (s *BalanceStorage) UpdateBalance(clientName string, balance payment.Balance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	_, exists := s.accountBalances[clientName]
	if !exists {
		return fmt.Errorf("client not found: %s", clientName)
	}
	
	s.accountBalances[clientName] = AccountBalance{
		Balance:  balance.Amount,
		Currency: balance.Currency,
	}
	
	return nil
}

// SaveBalances persists balances to file
func (s *BalanceStorage) SaveBalances() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var balances []BalanceData
	for clientName, balance := range s.accountBalances {
		balances = append(balances, BalanceData{
			ClientName: clientName,
			Balance:    balance.Balance,
			Currency:   balance.Currency,
		})
	}

	data, err := json.MarshalIndent(balances, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(s.filePath, data, 0644)
}