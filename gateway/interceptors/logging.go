package interceptors

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor is a unary server interceptor that logs request/response details
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract request metadata
		clientIP := "unknown"
		if p, ok := peer.FromContext(ctx); ok {
			clientIP = p.Addr.String()
		}

		username := "anonymous"
		transactionID := "none"
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if usernames := md.Get("username"); len(usernames) > 0 {
				username = usernames[0]
			}
			if txIDs := md.Get("transaction_id"); len(txIDs) > 0 {
				transactionID = txIDs[0]
			}
		}

		// Log the incoming request
		startTime := time.Now()
		log.Printf("[REQUEST] Method: %s, Client: %s, IP: %s, Transaction ID: %s, Payload: %+v",
			info.FullMethod,
			username,
			clientIP,
			transactionID,
			req,
		)

		// Process the request
		resp, err := handler(ctx, req)

		// Calculate request duration
		duration := time.Since(startTime)

		// Extract payment details if available
		var amountStr, clientName string

		// Try to extract payment amount and client from different request types
		switch v := req.(type) {
		case *pb.PaymentDetails:
			amountStr = fmt.Sprintf("%.2f", v.Amount)
			clientName = v.ClientName
		case *pb.PaymentRequest:
			amountStr = fmt.Sprintf("%.2f", v.Amount)
			clientName = "Account: " + v.AccountNumber
		case *pb.BalanceRequest:
			amountStr = "N/A (Balance Check)"
			clientName = v.ClientName
		default:
			amountStr = "N/A"
			clientName = username
		}

		// Log based on the response status
		if err != nil {
			st, _ := status.FromError(err)
			log.Printf("[ERROR] Method: %s, Client: %s, Amount: %s, Transaction ID: %s, Duration: %v, Code: %s, Message: %s",
				info.FullMethod,
				clientName,
				amountStr,
				transactionID,
				duration,
				st.Code(),
				st.Message(),
			)

			// Add specific logging for timeout or network errors
			if st.Code() == codes.DeadlineExceeded || st.Code() == codes.Unavailable {
				log.Printf("[RETRY_NEEDED] Transaction ID %s for client %s amount %s failed due to %s: %s",
					transactionID,
					clientName,
					amountStr,
					st.Code(),
					st.Message(),
				)
			}
		} else {
			// Log successful response
			var respStatus string

			// Try to extract success status from different response types
			switch v := resp.(type) {
			case *pb.PaymentResponse:
				if v.Success {
					respStatus = "SUCCESS"
				} else {
					respStatus = "FAILED: " + v.Message
				}
			case *pb.RegistrationResponse:
				if v.Success {
					respStatus = "SUCCESS"
				} else {
					respStatus = "FAILED: " + v.Message
				}
			case *pb.AuthResponse:
				if v.Success {
					respStatus = "SUCCESS"
				} else {
					respStatus = "FAILED: " + v.Message
				}
			default:
				respStatus = "COMPLETED"
			}

			log.Printf("[RESPONSE] Method: %s, Client: %s, Amount: %s, Transaction ID: %s, Duration: %v, Status: %s, Response: %+v",
				info.FullMethod,
				clientName,
				amountStr,
				transactionID,
				duration,
				respStatus,
				resp,
			)
		}

		return resp, err
	}
}