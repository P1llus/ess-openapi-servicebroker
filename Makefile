all: test build

build: 
	go build -o servicebroker -v

test:
	go test -v ./...
	go vet -v ./...
	golint ./...

clean: 
	go clean
	rm -f servicebroker

run:
	go build -o servicebroker -v
	./servicebroker

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o servicebroker -v
docker-build:
	docker run --rm -it -v "$(GOPATH)":/go -w /go/src/github.com/P1llus/ess-openapi-servicebroker golang:latest go build -o "servicebroker" -v