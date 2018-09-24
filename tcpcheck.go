package ping2ws

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// TCPObserver is an Observer the reports the availability of a port
// for TCP connections.
type TCPObserver struct {
	sync.Mutex
	Target   string
	Broker   Broker
	stopCh   chan struct{}
	updateCh chan Update
	running  bool
	timeout  time.Duration
}

// NewTCPObserver configures a new observer from an IP address and port number.
func NewTCPObserver(address string, port uint, timeout time.Duration, broker *Broker) (Observer, error) {
	targetIP := net.ParseIP(address)
	if targetIP.To4() == nil {
		return nil, fmt.Errorf("Invalid IPv4 address: %s", address)
	}
	target := fmt.Sprintf("%s:%d", address, port)
	return NewTCPObserverFromString(target, timeout, broker)
}

// NewTCPObserverFromString configures a new observer from a host string of the form "ip:port"
func NewTCPObserverFromString(host string, timeout time.Duration, broker *Broker) (Observer, error) {
	return &TCPObserver{
		Target:   host,
		Broker:   *broker,
		stopCh:   make(chan struct{}),
		updateCh: make(chan Update),
		running:  true,
		timeout:  timeout,
	}, nil
}

// Start attempts to ping the TCPObserver's Target
// and Publishes the result to the TCPObserver's Broker.
//
// Call this method as a goroutine.
func (o *TCPObserver) Start() {
	o.log("started")
	ticker := time.NewTicker(o.timeout)
	defer ticker.Stop()
	for _ = range ticker.C {
		select {
		case <-o.stopCh:
			o.log("stopped")
			return
		default:
			conn, dialErr := net.DialTimeout("tcp", o.Target, o.timeout)
			if dialErr != nil {
				o.Down()
				continue
			}
			conn.Close()
			o.Up()
		}
	}
}

// Stop signals that the TCPObserver's Start() method should exit.
func (o *TCPObserver) Stop() {
	o.Lock()
	if o.running {
		o.running = false
		o.log("received stop")
		close(o.stopCh)
	}
	o.Unlock()
}

// Down publishes a negative status for the observed resource.
func (o *TCPObserver) Down() {
	u := &Update{Target: o.Target, Up: false}
	o.Broker.Publish(u)
}

// Up publishes a positive status for the observed resource.
func (o *TCPObserver) Up() {
	u := &Update{Target: o.Target, Up: true}
	o.Broker.Publish(u)
}

// log is an internal logger for the observer.
func (o *TCPObserver) log(s string, args ...interface{}) {
	preface := fmt.Sprintf("Observer:%s ", o.Target)
	if len(args) > 0 {
		log.Printf(preface+s, args...)
	} else {
		log.Print(preface + s)
	}
}
