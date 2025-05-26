package interceptors

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/example/protofiles"
)

// RetryInterceptor logs and potentially handles retry logic for failed transactions
func RetryInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // Only handle payment processing methods
        if info.FullMethod == "/proto.PaymentGateway/MakePayment" {
            var resp interface{}
            var err error
            maxRetries := 3
            
            // Extract client and amount for logging
            paymentReq, ok := req.(*pb.PaymentDetails)
            if !ok {
                log.Printf("[ERROR] Expected PaymentDetails but got %T", req)
                return handler(ctx, req)
            }
            
            clientName := paymentReq.ClientName
            amount := paymentReq.Amount
            
            for attempt := 1; attempt <= maxRetries; attempt++ {
                // Process the request
                resp, err = handler(ctx, req)
                
                // If successful or not a retryable error, return immediately
                if err == nil {
                    if attempt > 1 {
                        log.Printf("[RETRY_SUCCESS] Transaction for client %s amount %.2f succeeded on attempt %d", 
                            clientName, 
                            amount, 
                            attempt,
                        )
                    }
                    return resp, err
                }
                
                // Check if error is retryable (timeout or unavailable)
                st, _ := status.FromError(err)
                if st.Code() != codes.DeadlineExceeded && st.Code() != codes.Unavailable {
                    // Not retryable
                    return resp, err
                }
                
                // Log retry attempt
                log.Printf("[RETRY_ATTEMPT] Transaction for client %s amount %.2f failed (attempt %d/%d): %s", 
                    clientName, 
                    amount, 
                    attempt, 
                    maxRetries, 
                    st.Message(),
                )
                
                // Wait before retrying (exponential backoff)
                if attempt < maxRetries {
                    backoffTime := time.Duration(attempt*100) * time.Millisecond
                    time.Sleep(backoffTime)
                }
            }
            
            log.Printf("[RETRY_EXHAUSTED] All retry attempts failed for client %s amount %.2f", 
                clientName, 
                amount,
            )
            
            return resp, err
        }
        
        // For non-payment methods, just pass through
        return handler(ctx, req)
    }
}