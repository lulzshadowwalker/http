default: 
	@just --list

alias run := http

http:
	go run cmd/http/main.go

tcp:
	go run cmd/tcp/main.go

udp:
	go run cmd/udp/main.go

test:
	go test ./...
