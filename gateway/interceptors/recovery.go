package interceptors

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryInterceptor prevents panics from crashing the server
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC] Method: %s, Recovered from panic: %v", info.FullMethod, r)
				// Return an error so the client knows something went wrong
				status.Errorf(codes.Internal, "Internal server error")
				// You might want to notify admins here
			}
		}()
		
		return handler(ctx, req)
	}
}