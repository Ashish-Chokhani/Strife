# Directories
PROTO_DIR      = protofiles
BANK_DIR       = bank
GATEWAY_DIR    = gateway
CLIENT_DIR     = client

# Proto files
PROTO_FILES    = $(PROTO_DIR)/bank.proto $(PROTO_DIR)/common.proto $(PROTO_DIR)/payment_gateway.proto
PROTO_OUT_DIR  = $(PROTO_DIR)

# gRPC flags
GO_FLAGS       = --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
                 --go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative

# Include protofiles directory in the search path
PROTO_PATH     = -I=$(PROTO_DIR)

.PHONY: proto bank gateway client clean all

# Generate Go code from proto files and place them in protofiles directory
proto:
	protoc $(PROTO_PATH) $(GO_FLAGS) $(PROTO_FILES)

# Run multiple bank servers
bank:
	go run $(BANK_DIR)/bank_server.go

# Run payment gateway server
gateway:
	go run $(GATEWAY_DIR)/server.go

# Run client
client:
	go run $(CLIENT_DIR)/client.go

# Start everything
all: proto bank gateway client

# Clean up generated proto files
clean:
	find $(PROTO_DIR) -name "*.pb.go" -delete