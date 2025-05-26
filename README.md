# Strife

## Introduction
This problem analyzes a distributed payment processing platform that leverages gRPC for inter-service communication. The system employs a three-component architecture consisting of a Gateway Server, multiple Bank Servers, and a Client Application. Key features include Two-Phase Commit transaction protocol, offline payment processing, transaction monitoring, and secure authentication mechanisms. The analysis covers system architecture, protocol implementation, fault tolerance mechanisms, and identifies strengths of the system.

## System Assumptions
1. Registered users can be directly extracted from the file
2. Users receive transaction IDs after completing transactions
3. Usernames and passwords are entered correctly without duplicates during registration
4. Once authenticated, clients don't need to re-authenticate for subsequent operations
5. All operations are case sensitive
6. No new client registrations occur while the gateway is offline
7. Authentication persists even when a client goes offline
8. Users cannot change their credentials
9. There are no compensating transactions to reverse partial commits
10. Clients only retry the same payment until successful, and don't attempt to retry old unsuccessful payments after switching to other payments
11. System is designed to handle inter-bank transfers 

## 1. System Architecture Analysis

### 1.1 Core Components
The system comprises three primary components:
- **Gateway Server**: Acts as the central coordination point, handling client authentication, request routing, and transaction orchestration
- **Bank Servers**: Multiple independent services managing account balances and processing transactions
- **Client Application**: User-facing interface that enables payment operations and maintains offline capabilities

### 1.2 Communication Model
The system implements a gRPC-based communication model with:
- **Clearly defined service contracts**: Protocol Buffer service definitions establishing API boundaries
- **Request-response patterns**: Synchronous communication for most operations
- **Streaming capabilities**: Potential for monitoring transaction status updates (implied in the service definitions)

### 1.3 Key Architectural Patterns
The architecture implements several established patterns:
- **Facade Pattern**: Gateway server abstracts the complexities of the banking network
- **Two-Phase Commit**: Ensures transaction atomicity across distributed banking services
- **Circuit Breaker**: Implied through retry mechanisms and offline processing capabilities
- **Command Pattern**: Payment operations encapsulated as commands that can be queued for later execution

## 2. Protocol Implementation Analysis

### 2.1 Service Definition Analysis

#### 2.1.1 Gateway Service
```protobuf
service PaymentGateway {
    rpc Authenticate(AuthRequest) returns (AuthResponse);
    rpc MakePayment(PaymentDetails) returns (PaymentResponse);
    rpc GetBalance(BalanceRequest) returns (BalanceResponse);
    rpc GetTransactionStatus(TransactionStatusRequest) returns (TransactionStatusResponse);
    rpc ListActiveTransactions(ListTransactionsRequest) returns (ListTransactionsResponse);
}
```

**Strengths:**
- Comprehensive API covering all user-facing operations
- Clear separation between authentication, transaction, and monitoring operations
- Support for both simple and complex payment scenarios

#### 2.1.2 Bank Service
```protobuf
service BankService {
  rpc ProcessPayment(PaymentRequest) returns (PaymentResponse);
  rpc PrepareDebit(TwoPhaseRequest) returns (TwoPhaseResponse);
  rpc PrepareCredit(TwoPhaseRequest) returns (TwoPhaseResponse);
  rpc CommitDebit(TwoPhaseRequest) returns (TwoPhaseResponse);
  rpc CommitCredit(TwoPhaseRequest) returns (TwoPhaseResponse);
  rpc Abort(TwoPhaseRequest) returns (TwoPhaseResponse);
  rpc GetBankName(TwoPhaseRequest) returns (TwoPhaseResponse);
}
```

**Strengths:**
- Complete implementation of Two-Phase Commit protocol operations
- Separation of debit and credit operations for greater control
- Support for both simple and complex payment scenarios

#### 2.1.3 Authentication Service
```protobuf
service AuthService {
  rpc Authenticate(AuthRequest) returns (AuthResponse);
}
```

**Strengths:**
- Simple, focused service following the single responsibility principle
- Clear separation from payment processing concerns

### 2.2 Two-Phase Commit Protocol Analysis

#### 2.2.1 Protocol Implementation
The system implements a classic Two-Phase Commit (2PC) protocol with properly defined transaction states:
```go
// Transaction states in payment package
const (
    TransactionPreparing // Initial state
    TransactionPrepared  // After successful prepare phase
    TransactionCommitting // During commit phase
    TransactionCommitted // Successfully completed
    TransactionAborting  // During abort phase
    TransactionAborted   // Failed and rolled back
)
```

The protocol follows a standard workflow:
1. **Prepare Phase**: Source bank verifies fund availability; destination bank prepares to receive
2. **Decision Phase**: Gateway decides to commit or abort based on prepare results
3. **Commit/Abort Phase**: All participants either commit or roll back changes

#### 2.2.2 Protocol Strength Analysis
- **Atomicity**: Ensures all-or-nothing transaction semantics across banks
- **Transaction Tracking**: Maintains detailed state for monitoring and recovery
- **Clear Status Transitions**: Well-defined state machine for transaction lifecycle

## 3. Fault Tolerance and Recovery Analysis

### 3.1 Server-Side Fault Tolerance

#### 3.1.1 Gateway Server Resilience
The gateway implements several fault tolerance mechanisms:
```go
stopChan := make(chan os.Signal, 1)
signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

userStorage, err := storage.NewUserStorage(cfg.UsersFilePath)
balanceStorage, err := storage.NewBalanceStorage(cfg.BalancesFilePath)

transactionMgr := payment.NewTransactionManager(cfg.TransactionTTL, cfg.CleanupInterval)
```

**Strengths:**
- Graceful shutdown handling via signal capturing
- Persistent storage for user and balance data
- Transaction TTL management preventing resource leaks

#### 3.1.2 Bank Server Resilience
Bank servers appear to have independent operation capabilities:
- Independent state management for client records and balances
- Protocol-level coordination via 2PC for consistency

### 3.2 Client-Side Fault Tolerance

#### 3.2.1 Offline Processing Capabilities
The client implements sophisticated offline payment handling:
```go
// Check if payment already queued
if payment.TransactionID == transactionID || 
   (payment.ClientName == clientName && payment.RequestId == requestID && requestID != "") {
    log.Printf("[OFFLINE] Payment with transaction ID %s or request ID %s already queued", 
        transactionID, requestID)
    return
}
```

**Strengths:**
- Request ID-based idempotency ensures payment uniqueness
- Payment queuing for network outages
- Persistent storage of pending transactions

#### 3.2.2 Network Resilience
The client implements network resilience through:
```go
interceptorChain := grpc.WithChainUnaryInterceptor(
    interceptors.LoggingInterceptor(),
    interceptors.RetryInterceptor(),
)
```

**Strengths:**
- Automatic retry of failed RPC calls
- Logging for operation traceability
- Interceptor chain for modular middleware approach

## 4. Security Implementation Analysis

### 4.1 Authentication and Authorization
The system implements token-based authentication:
```go
interceptorChain := interceptors.ChainInterceptors(
    interceptors.RecoveryInterceptor(),
    interceptors.LoggingInterceptor(),
    auth.NewAuthInterceptor(server.authService),
)
```

**Strengths:**
- Dedicated authentication service
- Auth interceptor for enforcing authentication on protected operations
- Persistent authentication across sessions (per assumptions)

### 4.2 Communication Security
All gRPC communications use TLS:
```go
tlsCredentials, err := config.SetupTLSCredentials(cfg.TLSConfig)
```

**Strengths:**
- Encrypted communications between all system components
- Credential configuration through configuration files

## 5. Transaction Management Analysis

### 5.1 Idempotency Implementation
The system implements request idempotency through unique request IDs:
```go
requestID := app.requestCounter.GetNextRequestID(clientName)
```

And verifies them during processing:
```go
// Check if this payment is already in the queue
for _, payment := range m.pendingPayments {
    if payment.TransactionID == transactionID || 
       (payment.ClientName == clientName && payment.RequestId == requestID && requestID != "") {
        log.Printf("[OFFLINE] Payment with transaction ID %s or request ID %s already queued", 
            transactionID, requestID)
        return
    }
}
```

### 5.2 Idempotency Approach with Correctness Proof
The payment system implements idempotency through request IDs to ensure that duplicate payment requests don't result in duplicate transactions. This approach is formally proven below.

#### 5.2.1 Formal Model
**Definitions**
- Payment Request: A tuple R = (clientName, amount, receiverName, currency, requestID)
- Payment Processing Function: P(R) → (result, sideEffects)
- Payment Database: DB = {transaction₁, transaction₂, ..., transactionₙ}
- Idempotency: A property where P(R) executed multiple times produces the same observable result as if executed exactly once

**System Properties**
For the payment processing system to be idempotent, it satisfies:
- Uniqueness: Each valid payment request has a unique identifier (requestID)
- Determinism: For any given input R, the function P must produce the same output when successfully executed
- State Awareness: The system can detect if a request has been processed previously

**Algorithm**
The system implements idempotency through the following algorithm:
```
function ProcessPayment(R):
    if DB.contains(transaction with R.requestID):
        return DB.get(transaction with R.requestID)
    else:
        result = executePayment(R)
        DB.store(R.requestID, result)
        return result
```

#### 5.2.2 Proof by Cases
**Case 1: First Execution of Request R**
When P(R) is called for the first time:
- No transaction with R.requestID exists in the database
- The system executes the payment operation
- The result is stored in the database with key R.requestID
- The function returns the result of the payment operation

**Case 2: Subsequent Executions of Request R**
When P(R) is called again with the same request:
- A transaction with R.requestID already exists in the database
- The system retrieves the stored result without re-executing the payment operation
- The function returns the same result as in Case 1

**Case 3: Concurrent Executions**
With concurrent calls to P(R) with the same request:
- Database transaction isolation ensures that only one execution creates a record
- All other concurrent executions will either:
  - Wait for the first execution to complete and then retrieve the result, or
  - Fail with a unique constraint violation and retry by retrieving the result

#### 5.2.3 Implementation Evidence
The system implements this approach in multiple places:

**Client-Side Request ID Generation:**
```go
requestID := app.requestCounter.GetNextRequestID(clientName)
```

**Request ID Checking in Payment Processing:**
```go
// Check if this payment is already in the queue
for _, payment := range m.pendingPayments {
    if payment.TransactionID == transactionID || 
       (payment.ClientName == clientName && payment.RequestId == requestID && requestID != "") {
        log.Printf("[OFFLINE] Payment with transaction ID %s or request ID %s already queued", 
            transactionID, requestID)
        return
    }
}
```

**Request ID Inclusion in gRPC Messages:**
```protobuf
message PaymentDetails {
  string client_name = 1;
  float amount = 2;
  string receiver_name = 3;
  string currency = 4;
  string request_id = 5;  // For idempotency
}
```

This implementation ensures that even with network partitions, server restarts, or client retries, each payment request is processed exactly once, maintaining consistency in the financial system.

### 5.3 Transaction Monitoring
The system provides transaction monitoring capabilities:
```protobuf
rpc GetTransactionStatus(TransactionStatusRequest) returns (TransactionStatusResponse);
rpc ListActiveTransactions(ListTransactionsRequest) returns (ListTransactionsResponse);
```

**Strengths:**
- Real-time transaction status tracking
- Support for listing active transactions
- Detailed state tracking for 2PC operations

## 6. Scalability Analysis

### 6.1 Gateway Scalability
The gateway server has several design elements supporting horizontal scaling:
- Stateless authentication through tokens
- Shared-nothing architecture allowing independent operation
- Interceptor chain for modular cross-cutting concerns

### 6.2 Bank Service Scalability
The bank services have independent operation capabilities:
- Direct port assignment
- Independent state maintenance
- 2PC for consistency across instances

## 7. Conclusion
The distributed payment processing system implements a robust architecture using gRPC for inter-service communication. The Two-Phase Commit protocol ensures transaction atomicity across distributed banks, while offline processing capabilities provide resilience against network failures. The system has strong security foundations with TLS and authentication interceptors.

Key strengths include the idempotency implementation (with formal correctness proof), transaction monitoring capabilities, and offline payment processing. The system demonstrates a well-designed distributed architecture with excellent fault tolerance and security features, while maintaining good scalability characteristics for handling growing transaction volumes.


## How to Run

### Prerequisites
- [Go](https://golang.org/dl/) (version 1.18+)
- `protoc` (Protocol Buffers compiler) with Go plugins installed:
  ```bash
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  ```
- Make sure `$GOPATH/bin` is in your `PATH`, as `protoc` needs access to the installed plugins.

---

### Step-by-Step Instructions

1. **Initialize Go Modules** (if not already initialized):
   ```bash
   go mod init github.com/example
   go mod tidy
   ```

2. **Generate gRPC Code from Protobufs:**
   ```bash
   make proto
   ```

3. **Run Bank Servers:**
   Open one or more terminals and start bank servers:
   ```bash
   make bank
   ```

4. **Run Gateway Server:**
   In a new terminal:
   ```bash
   make gateway
   ```

5. **Run the Client Application:**
   In another terminal:
   ```bash
   make client
   ```

---

### Cleaning Up
To delete all generated `.pb.go` files:
```bash
make clean
```

---

### TLS and Configuration Notes
- TLS certificates are located in the `certs/` directory.
- Configuration and state files (e.g., users, balances, pending payments) are in the `data/` directory.
- You may modify paths and parameters in the code or configuration files (`client/config`, `gateway/config`) as needed.
