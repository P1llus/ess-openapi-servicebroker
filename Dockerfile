# First stage
FROM golang:1.14.7 as builder

LABEL maintainer="Marius Iversen <marius@chasenet.org>"

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Second stage
FROM alpine:latest  

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8000

CMD ["./main"] 