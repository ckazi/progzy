FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod ./
RUN go mod download && go mod verify

COPY . .

RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o proxy-server .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/proxy-server .

EXPOSE 8080 8081

CMD ["./proxy-server"]
