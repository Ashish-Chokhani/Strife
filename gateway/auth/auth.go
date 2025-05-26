package auth

import (
	"context"
	"sync"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UserStorage defines the interface for user data operations
type UserStorage interface {
	GetUser(username string) (User, bool)
	SaveUser(user User) error
}

// Service handles authentication and authorization
type Service struct {
	userStorage        UserStorage
	authenticatedUsers map[string]bool
	mu                 sync.RWMutex
}

// NewService creates a new auth service
func NewService(userStorage UserStorage) *Service {
	return &Service{
		userStorage:        userStorage,
		authenticatedUsers: make(map[string]bool),
	}
}

// Authenticate authenticates a user
func (s *Service) Authenticate(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
    user, exists := s.userStorage.GetUser(req.Username)
    if !exists || user.Password != req.Password {
        return &pb.AuthResponse{
            Success: false,
            Message: "Invalid credentials",
        }, nil
    }

    s.mu.Lock()
    s.authenticatedUsers[req.Username] = true
    s.mu.Unlock()

    return &pb.AuthResponse{
        Success: true,
        Message: "Authentication successful",
        Roles:   user.Roles,
    }, nil
}

// IsAuthenticated checks if a user is authenticated
func (s *Service) IsAuthenticated(username string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authenticatedUsers[username]
}

// HasRole checks if a user has a specific role
func (s *Service) HasRole(username string, requiredRoles []string) bool {
	user, exists := s.userStorage.GetUser(username)
	if !exists {
		return false
	}

	for _, requiredRole := range requiredRoles {
		for _, userRole := range user.Roles {
			if userRole == requiredRole {
				return true
			}
		}
	}
	return false
}

// ExtractUsername extracts username from context metadata
func ExtractUsername(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "missing context metadata")
	}

	usernames := md.Get("username")
	if len(usernames) == 0 {
		return "", status.Errorf(codes.Unauthenticated, "username required")
	}

	return usernames[0], nil
}

