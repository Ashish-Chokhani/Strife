package auth

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthService defines the authentication interface needed by the interceptor
type AuthService interface {
	IsAuthenticated(username string) bool
	HasRole(username string, requiredRoles []string) bool
}

// AuthInterceptor handles authentication and authorization
type AuthInterceptor struct {
	authService AuthService
}

// NewAuthInterceptor creates a new auth interceptor
func NewAuthInterceptor(authService AuthService) grpc.UnaryServerInterceptor {
	interceptor := &AuthInterceptor{
		authService: authService,
	}
	return interceptor.Unary
}

// Unary is the unary server interceptor function
func (i *AuthInterceptor) Unary(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	method := info.FullMethod
	log.Printf("Method called: %s", method)

	// Check if this is a public method (no auth required)
	if PublicMethods[method] {
		return handler(ctx, req)
	}

	// Extract metadata for authentication
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Printf("Missing metadata")
		return nil, status.Errorf(codes.Unauthenticated, "missing context metadata")
	}

	// Get username from metadata
	usernames := md.Get("username")
	if len(usernames) == 0 {
		log.Printf("No username in metadata")
		return nil, status.Errorf(codes.Unauthenticated, "username required")
	}

	username := usernames[0]

	// Check if user is authenticated
	// if !i.authService.IsAuthenticated(username) {
	// 	log.Printf("User %s is not authenticated", username)
	// 	return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	// }

	// Authorization check - get required roles for this method
	requiredRoles, hasPermissions := MethodPermissions[method]

	// If method has no specific permissions, allow access
	if !hasPermissions {
		log.Printf("Method has no specific permissions: %s", method)
		return handler(ctx, req)
	}

	// Check if user has required role
	if !i.authService.HasRole(username, requiredRoles) {
		log.Printf("User %s lacks required role(s) for %s", username, method)
		return nil, status.Errorf(codes.PermissionDenied, "insufficient permissions")
	}

	log.Printf("User %s authorized for method %s", username, method)
	return handler(ctx, req)
}