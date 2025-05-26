// storage/users.go
package storage

import (
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/example/gateway/auth"
)

// UserStorage manages user data
type UserStorage struct {
	filePath string
	users    map[string]auth.User
	mu       sync.RWMutex
}

// NewUserStorage creates a new user storage
func NewUserStorage(filePath string) (*UserStorage, error) {
	users, err := loadUsers(filePath)
	if err != nil {
		return nil, err
	}
	
	return &UserStorage{
		filePath: filePath,
		users:    users,
	}, nil
}

// loadUsers loads users from a JSON file
func loadUsers(filePath string) (map[string]auth.User, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var users []auth.User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, err
	}

	userMap := make(map[string]auth.User)
	for _, user := range users {
		userMap[user.Username] = user
	}
	return userMap, nil
}

// GetUser retrieves a user by username
func (s *UserStorage) GetUser(username string) (auth.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	user, exists := s.users[username]
	return user, exists
}

// SaveUser saves a user
func (s *UserStorage) SaveUser(user auth.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.users[user.Username] = user
	
	// Convert map to slice for JSON serialization
	var userSlice []auth.User
	for _, u := range s.users {
		userSlice = append(userSlice, u)
	}
	
	data, err := json.MarshalIndent(userSlice, "", "  ")
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(s.filePath, data, 0644)
}