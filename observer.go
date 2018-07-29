package ping2ws

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Observer struct {
	sync.Mutex
	Target   string
	Broker   Broker
	stopCh   chan struct{}
	updateCh chan Update
	status   bool
	running  bool
}

func NewObserver(target string, broker *Broker) (*Observer, error) {
	targetIP := net.ParseIP(target)
	if targetIP == nil {
		return nil, fmt.Errorf("Could not parse target IPv4 address: %s", target)
	}
	return &Observer{
		Target:   target,
		Broker:   *broker,
		stopCh:   make(chan struct{}),
		updateCh: make(chan Update),
		status:   false,
		running:  true,
	}, nil
}

// Start attempts to ping the Observer's Target
// and Publishes the result to the Observer's Broker.
//
// ICMP echo request based on reference in x/net/icmp docs.
//
// Call this method as a goroutine.
func (o *Observer) Start() {
	o.log("started")
	conn, listenErr := icmp.ListenPacket("udp4", "0.0.0.0")
	if listenErr != nil {
		o.log("Could not observe target: %s", listenErr)
		return
	}
	defer conn.Close()

	// Construct base packet, which we'll reuse
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0, //???
		// Body: Messagebody (interface)
		Body: &icmp.Echo{
			ID:  os.Getpid() & 0xffff,
			Seq: 1,
		},
	}
	target := net.ParseIP(o.Target)
	addr := &net.UDPAddr{IP: target}
	for {
		select {
		case <-o.stopCh:
			o.log("stopped")
			return
		default:
			wb, marshalErr := wm.Marshal(nil)
			if marshalErr != nil {
				o.log("Could not marshal messages. Killing observer.")
				return
			}
			// Send packet
			_, writeErr := conn.WriteTo(wb, addr)
			if writeErr != nil {
				o.Down()
				continue
			}
			// Receive response
			rb := make([]byte, 1500)
			n, peer, readErr := conn.ReadFrom(rb)
			if readErr != nil {
				o.log("Ignoring read error: %s", readErr)
				continue
			}
			receivedAddr, castOk := peer.(*net.UDPAddr)
			if !castOk {
				o.log("Couldn't cast to UDP address (ignored): ", peer)
				continue
			}
			if receivedAddr.IP.String() != o.Target {
				// Response to someone else's ping.
				if !o.status {
					// If we got someone else's reply BEFORE we get a positive response,
					// report outage again.
					//TODO This overreports more and more as number of targets increases.
					// Can address this, but requires tracking even more state.
					o.Down()
				}
				continue
			}
			rm, parseErr := icmp.ParseMessage(ipv4.ICMPTypeEcho.Protocol(), rb[:n])
			if parseErr != nil {
				o.log("Ignoring parse error: ", parseErr)
				continue
			}
			switch rm.Type {
			case ipv4.ICMPTypeEchoReply:
				// deserialize rm.Body to icmp.Echo
				// https://stackoverflow.com/a/38560729
				_, castOk := rm.Body.(*icmp.Echo)
				if castOk {
					o.Up()
				} else {
					o.Down()
				}
			case ipv4.ICMPTypeDestinationUnreachable:
				o.Down()
			default:
				o.log("got %+v; want echo reply", rm)
				o.Down()
			}
			wm.Body.(*icmp.Echo).Seq++
			//TODO This could probably be a Ticker
			//TODO Interval could be configurable
			time.Sleep(1 * time.Second)
		}
	}
}

// Stop signals that the Observer's Start() method should exit.
// Calling this twice will cause a panic, so don't do that.
func (o *Observer) Stop() {
	o.Lock()
	if o.running {
		o.running = false
		o.log("received stop")
		close(o.stopCh)
	}
	o.Unlock()
}

// Down publishes a negative status for the observed resource.
func (o *Observer) Down() {
	o.status = false
	u := &Update{Target: o.Target, Up: false}
	o.Broker.Publish(u)
}

// Up publishes a positive status for the observed resource.
func (o *Observer) Up() {
	o.status = true
	u := &Update{Target: o.Target, Up: true}
	o.Broker.Publish(u)
}

// log is an internal logger for the observer.
func (o *Observer) log(s string, args ...interface{}) {
	preface := fmt.Sprintf("Observer:%s ", o.Target)
	if len(args) > 0 {
		log.Print(preface + s)
	} else {
		log.Printf(preface+s, args...)
	}
}
