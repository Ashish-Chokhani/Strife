package interceptors

import (
	"context"

	"google.golang.org/grpc"
)

// ChainInterceptors combines multiple interceptors into a single interceptor
func ChainInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        buildChain := func(current grpc.UnaryServerInterceptor, next grpc.UnaryHandler) grpc.UnaryHandler {
            return func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
                return current(currentCtx, currentReq, info, next)
            }
        }
        
        chain := handler
        // Apply interceptors in reverse order
        for i := len(interceptors) - 1; i >= 0; i-- {
            chain = buildChain(interceptors[i], chain)
        }
        
        return chain(ctx, req)
    }
}