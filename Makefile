STANDARD_REDIS_ADDR ?= :6379
STANDARD_REDIS_DB ?= 0
STANDARD_ENCODING_TYPE ?= msgp
TESTNET_REDIS_ADDR ?= :6379
TESTNET_REDIS_DB ?= 1
TESTNET_ENCODING_TYPE ?= msgp

version = $(shell git describe | cut -d '-' -f 1)
commit = $(shell git rev-parse --short HEAD)
ifeq ($(commit), $(shell git rev-list -n 1 $(version) | cut -c1-7))
fullversion = $(version)
else
fullversion = $(version)-$(commit)
endif

stdbindir = $(GOPATH)/bin
ldflagsversion = -X main.rawVersion=$(fullversion)

testpkgs = . ./pkg/types ./pkg/encoding

install-std: test
	go build -ldflags "$(ldflagsversion) -s -w" -o $(stdbindir)/rexplorer .

install: test
	go build -race -tags "debug dev" -ldflags "$(ldflagsversion)" -o $(stdbindir)/rexplorer .

test: ineffassign
	go test -race -tags "debug testing" $(testpkgs)

ineffassign:
	ineffassign $(testpkgs)

integration-tests: integration-test-sumcoins

integration-test-sumcoins:
	go run tests/integration/sumcoins/main.go \
		--db-address "$(TESTNET_REDIS_ADDR)" --db-slot "$(TESTNET_REDIS_DB)" \
		--encoding "$(TESTNET_ENCODING_TYPE)"
	go run tests/integration/sumcoins/main.go \
		--db-address "$(STANDARD_REDIS_ADDR)" --db-slot "$(STANDARD_REDIS_DB)" \
		--encoding "$(STANDARD_ENCODING_TYPE)"

generate-messagepack:
	go generate pkg/types/types.go
	go generate types.go
