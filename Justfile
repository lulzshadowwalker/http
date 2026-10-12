default: 
	@just --list

alias run := tcp

tcp:
	go run cmd/tcp/main.go

udp:
	go run cmd/udp/main.go

test:
	go test ./...
