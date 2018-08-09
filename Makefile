STANDARD_REDIS_ADDR = :6379
STANDARD_REDIS_DB = 0
TESTNET_REDIS_ADDR = :6379
TESTNET_REDIS_DB = 1

install-std:
	go build -o $(GOPATH)/bin/rexplorer .

install:
	go build -tags 'debug dev' -o $(GOPATH)/bin/rexplorer .

integration-tests: integration-test-sumcoins

integration-test-sumcoins:
	go run tests/integration/sumcoins/main.go --network testnet --db-address "$(TESTNET_REDIS_ADDR)" --db-slot "$(TESTNET_REDIS_DB)"
	go run tests/integration/sumcoins/main.go --network standard --db-address "$(STANDARD_REDIS_ADDR)" --db-slot "$(STANDARD_REDIS_DB)"
