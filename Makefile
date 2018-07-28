.PHONY: ssl example

ssl:
	openssl genrsa -out example/server.key 2048
	openssl req -new -x509 -sha256 -key example/server.key -out example/server.crt -days 3650

example:
	cd example && go run cmd/ping/main.go

