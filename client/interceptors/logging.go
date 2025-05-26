// client/interceptors/logging.go

package interceptors

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor logs outgoing requests and incoming responses
func LoggingInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Log outgoing request
		startTime := time.Now()
		
		// Extract request info
		clientName, amountStr, transactionID := extractRequestInfo(ctx, req)
		
		log.Printf("[CLIENT REQUEST] Method: %s, Client: %s, Amount: %s, Transaction ID: %s, Payload: %+v", 
			method, 
			clientName, 
			amountStr, 
			transactionID,
			req,
		)
		
		// Make the call
		err := invoker(ctx, method, req, reply, cc, opts...)
		
		// Calculate duration
		duration := time.Since(startTime)
		
		// Log response or error
		if err != nil {
			st, _ := status.FromError(err)
			log.Printf("[CLIENT ERROR] Method: %s, Client: %s, Amount: %s, Transaction ID: %s, Duration: %v, Error: %v", 
				method, 
				clientName, 
				amountStr, 
				transactionID,
				duration, 
				st.Message(),
			)
		} else {
			status := determineResponseStatus(reply)
			
			log.Printf("[CLIENT RESPONSE] Method: %s, Client: %s, Amount: %s, Transaction ID: %s, Duration: %v, Status: %s, Response: %+v", 
				method, 
				clientName, 
				amountStr, 
				transactionID,
				duration, 
				status, 
				reply,
			)
		}
		
		return err
	}
}

// Helper function to extract transaction ID from context
func extractTransactionID(ctx context.Context) string {
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if ids := md.Get("transaction_id"); len(ids) > 0 {
			return ids[0]
		}
	}
	return ""
}

// Helper function to extract relevant information from request
func extractRequestInfo(ctx context.Context, req interface{}) (clientName, amountStr, transactionID string) {
	transactionID = extractTransactionID(ctx)
	
	switch v := req.(type) {
	case *pb.PaymentDetails:
		clientName = v.ClientName
		amountStr = fmt.Sprintf("%.2f", v.Amount)
	case *pb.BalanceRequest:
		clientName = v.ClientName
		amountStr = "N/A (Balance Check)"
	case *pb.AuthRequest:
		clientName = v.Username
		amountStr = "N/A (Authentication)"
	case *pb.ClientDetails:
		clientName = v.Name
		amountStr = "N/A (Registration)"
	default:
		clientName = "unknown"
		amountStr = "unknown"
	}
	
	return
}

// Helper function to determine success status from response
func determineResponseStatus(reply interface{}) string {
	switch v := reply.(type) {
	case *pb.PaymentResponse:
		if v.Success {
			return "SUCCESS"
		}
		return "FAILED: " + v.Message
	case *pb.RegistrationResponse:
		if v.Success {
			return "SUCCESS"
		}
		return "FAILED: " + v.Message
	case *pb.AuthResponse:
		if v.Success {
			return "SUCCESS"
		}
		return "FAILED: " + v.Message
	case *pb.BalanceResponse:
		return fmt.Sprintf("Balance: %.2f %s", v.Balance, v.Currency)
	default:
		return "COMPLETED"
	}
}