all: test build

build: 
	go build -o ess-openapi-servicebroker -v

test:
	go test ./...
	go vet ./...
	golint ./...

clean: 
	go clean
	rm -f ess-openapi-servicebroker

run:
	go build -o ess-openapi-servicebroker -v
	./ess-openapi-servicebroker

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ess-openapi-servicebroker -v
docker-build:
	docker run --rm -it -v "$(GOPATH)":/go -w /go/src/github.com/P1llus/ess-openapi-servicebroker golang:latest go build -o "ess-openapi-servicebroker" -v