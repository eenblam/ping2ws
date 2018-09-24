.PHONY: ssl example

ssl:
	openssl genrsa -out example/server.key 2048
	openssl req -new -x509 -sha256 -key example/server.key -out example/server.crt -days 3650

example:
	cd example && go run cmd/ping/main.go

example-tcp:
	cd example && go run cmd/tcp/main.go

linux:
	# Allow unprivileged ICMP
	# Some distros may need "1   0" instead?
	sysctl -w net.ipv4.ping_group_range="0   2147483647"
