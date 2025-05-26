package payment

import (
	"context"

	"github.com/example/gateway/auth"
	pb "github.com/example/protofiles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Balance represents an account balance
type Balance struct {
	Amount   float32
	Currency string
}

// BalanceService handles balance-related operations
type BalanceService struct {
	balanceStorage BalanceStorage
}

// NewBalanceService creates a new balance service
func NewBalanceService(balanceStorage BalanceStorage) *BalanceService {
	return &BalanceService{
		balanceStorage: balanceStorage,
	}
}

// GetBalance retrieves a client's balance
func (s *BalanceService) GetBalance(ctx context.Context, req *pb.BalanceRequest) (*pb.BalanceResponse, error) {
	// Extract username from metadata
	username, err := auth.ExtractUsername(ctx)
	if err != nil {
		return nil, err
	}

	// Authorization check - ensure user can only view their own balance
	if username != req.ClientName {
		return nil, status.Errorf(codes.PermissionDenied, "cannot view balance for other clients")
	}

	// Get balance
	balance, exists := s.balanceStorage.GetBalance(req.ClientName)
	if !exists {
		return nil, status.Errorf(codes.NotFound, "no balance information found for client")
	}

	return &pb.BalanceResponse{
		Balance:  float64(balance.Amount),
		Currency: balance.Currency,
	}, nil
} 