install-std:
	go build -o $(GOPATH)/bin/rexplorer .

install:
	go build -tags 'debug dev' -o $(GOPATH)/bin/rexplorer .
