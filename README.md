# ping2ws
ping2ws provides a WebSocket handler that streams uptime observations for a number of targets to the front-end.

The actual observation architecture supports a few observation strategies:
- ICMP Echo requests (requires enabling unprivileged ping)
- Attempting TCP connections on a specific port

Internally, ping2ws uses a pub-sub model to distribute notifications.

## Planned Features
- Improve pub-sub model to include topic-based subscription
- Add/remove targets via gRPC
