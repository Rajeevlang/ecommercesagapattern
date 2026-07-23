.PHONY: gen clean

# Path configuration
PROTO_DIR = shared/protofiles
OUT_DIR = shared/pb

# Locate all proto files
PROTOS = $(shell find $(PROTO_DIR) -name "*.proto")

# Default action
all: gen

# Target to generate Go code from Protobuf files
gen:
	@echo "Creating output directory: $(OUT_DIR)"
	mkdir -p $(OUT_DIR)
	@echo "Compiling Protobuf files..."
	protoc --proto_path=$(PROTO_DIR) \
	       --go_out=$(OUT_DIR) --go_opt=paths=source_relative \
	       --go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
	       $(PROTOS)
	@echo "Go files successfully generated in $(OUT_DIR)"

# Clean generated Go structures
clean:
	@echo "Removing generated files..."
	rm -rf $(OUT_DIR)/*
	@echo "Cleanup completed."
