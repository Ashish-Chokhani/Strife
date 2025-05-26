// client/interceptors/retry.go

package interceptors

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetryInterceptor implements retry logic for the client
func RetryInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Extract client info
		clientName, isPayment, amount, receiverName, requestId := extractRetryInfo(req)
		
		maxRetries := 3
		var lastErr error
		
		// Handle transaction ID for payment requests
		if isPayment && method == "/proto.PaymentGateway/MakePayment" {
			metadata := TransactionMetadata{
                ClientName:   clientName,
                ReceiverName: receiverName,
                Amount:       amount,
                RequestId:    requestId,
            }
            ctx = ensureTransactionID(ctx, metadata)
		}
		
		for attempt := 1; attempt <= maxRetries; attempt++ {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				if attempt > 1 {
					log.Printf("[CLIENT RETRY SUCCESS] Method: %s, Client: %s, succeeded on attempt %d", 
						method, 
						clientName, 
						attempt,
					)
				}
				return nil
			}
			
			lastErr = err
			st, _ := status.FromError(err)
			
			// Only retry on specific error codes
			if !isRetryableError(st.Code()) || attempt == maxRetries {
				if attempt > 1 {
					log.Printf("[CLIENT RETRY FAILED] Method: %s, Client: %s, all %d attempts failed", 
						method, 
						clientName, 
						attempt,
					)
				}
				return lastErr
			}
			
			log.Printf("[CLIENT RETRY ATTEMPT] Method: %s, Client: %s, attempt %d/%d, error: %v", 
				method, 
				clientName, 
				attempt, 
				maxRetries, 
				st.Message(),
			)
			
			// Exponential backoff
			backoffTime := time.Duration(attempt*200) * time.Millisecond
			time.Sleep(backoffTime)
		}
		
		return lastErr
	}
}

// Helper function to extract client information for retry logging
func extractRetryInfo(req interface{}) (clientName string, isPayment bool, amount string, receiverName string, requestID string) {
    switch v := req.(type) {
    case *pb.PaymentDetails:
        clientName = v.ClientName
        isPayment = true
        amount = fmt.Sprintf("%.2f", v.Amount)
        receiverName = v.ReceiverName
        requestID = v.RequestId // Extract the request ID
    case *pb.BalanceRequest:
        clientName = v.ClientName
    case *pb.AuthRequest:
        clientName = v.Username
    case *pb.ClientDetails:
        clientName = v.Name
    default:
        clientName = "unknown"
    }
    
    return
}

// Determine if an error code is retryable
func isRetryableError(code codes.Code) bool {
	return code == codes.Unavailable || 
		   code == codes.DeadlineExceeded || 
		   code == codes.ResourceExhausted
}