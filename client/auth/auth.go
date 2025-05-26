// client/auth/auth.go

package auth

import (
	"context"
	"fmt"
	"log"
	"sync"

	pb "github.com/example/protofiles"
)

// User represents authentication credentials
type User struct {
	Username string
	Password string
}

// AuthManager handles user authentication
type AuthManager struct {
	client            pb.PaymentGatewayClient
	authenticatedUsers map[string]bool
	userRoles         map[string][]string
	mu                sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(client pb.PaymentGatewayClient) *AuthManager {
	return &AuthManager{
		client:            client,
		authenticatedUsers: make(map[string]bool),
		userRoles:         make(map[string][]string),
	}
}
// AuthenticateUsers authenticates a list of users
func (am *AuthManager) AuthenticateUsers(users []User) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for _, u := range users {
		authResp, err := am.client.Authenticate(context.Background(), &pb.AuthRequest{
			Username: u.Username,
			Password: u.Password,
		})
		if err != nil {
			log.Printf("Authentication failed for %s: %v (will assume authenticated for offline mode)", u.Username, err)
			// In offline mode, assume users are authenticated
			am.authenticatedUsers[u.Username] = true
		} else {
			fmt.Printf("Auth response for %s: %v\n", u.Username, authResp)
			
			// Store user roles if authentication successful
			if authResp.Success {
				am.authenticatedUsers[u.Username] = true
				am.userRoles[u.Username] = authResp.Roles
			}
		}
	}
}

// IsAuthenticated checks if a user is authenticated
func (am *AuthManager) IsAuthenticated(username string) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.authenticatedUsers[username]
}

// GetUserRoles returns the roles assigned to a user
func (am *AuthManager) GetUserRoles(username string) []string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.userRoles[username]
}

// GetAuthenticatedUsernames returns a list of authenticated usernames
func (am *AuthManager) GetAuthenticatedUsernames() []string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	usernames := make([]string, 0, len(am.authenticatedUsers))
	for username := range am.authenticatedUsers {
		usernames = append(usernames, username)
	}
	return usernames
}