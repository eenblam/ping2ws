package ping

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Observer struct {
	Target   string
	Broker   Broker
	stopCh   chan struct{}
	updateCh chan Update
	status   bool
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
	}, nil
}

// Start attempts to ping the Observer's Target
// and Publishes the result to the Observer's Broker.
//
// ICMP echo request based on reference in x/net/icmp docs.
//
// Call this method as a goroutine.
func (o *Observer) Start() {
	log.Print("Worker observing ", o.Target)
	conn, listenErr := icmp.ListenPacket("udp4", "0.0.0.0")
	if listenErr != nil {
		log.Fatal(listenErr)
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
			return
		default:
			wb, marshalErr := wm.Marshal(nil)
			if marshalErr != nil {
				log.Printf("Could not marshal messages to %s. Killing goroutine.", o.Target)
				return
			}
			// Send packet
			_, writeErr := conn.WriteTo(wb, addr)
			if writeErr != nil {
				log.Print("Write error: ", writeErr)
				o.Down()
				continue
			}
			// Receive response
			rb := make([]byte, 1500)
			n, peer, readErr := conn.ReadFrom(rb)
			if readErr != nil {
				log.Print("Ignoring read error: %s", readErr)
				continue
			}
			receivedAddr, castOk := peer.(*net.UDPAddr)
			if !castOk {
				log.Print("Couldn't cast to UDP address (ignored): ", peer)
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
				log.Print("Ignoring parse error: ", parseErr)
				continue
			}
			switch rm.Type {
			case ipv4.ICMPTypeEchoReply:
				// deserialize rm.Body to icmp.Echo
				// https://stackoverflow.com/a/38560729
				_, castOk := rm.Body.(*icmp.Echo)
				if castOk {
					//log.Printf("got reflection from %v for seq %d", peer, body.Seq)
					o.Up()
				} else {
					o.Down()
				}
			case ipv4.ICMPTypeDestinationUnreachable:
				o.Down()
			default:
				log.Printf("got %+v; want echo reply", rm)
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
	close(o.stopCh)
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
