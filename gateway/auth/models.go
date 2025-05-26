package auth

// User represents an authenticated user
type User struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}


// Method to role mapping
var MethodPermissions = map[string][]string{
    // "/proto.PaymentGateway/RegisterClient":    {"admin"},
    "/proto.PaymentGateway/MakePayment":       {"user", "admin"},
    "/proto.PaymentGateway/GetPaymentHistory": {"admin"},
    "/proto.PaymentGateway/GetBalance":        {"user", "admin"},
}

// Public methods that don't require authentication
var PublicMethods = map[string]bool{
    "/proto.PaymentGateway/Authenticate": true,
    // Remove RegisterClient from here since it requires admin privileges
}