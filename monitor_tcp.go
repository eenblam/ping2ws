package ping2ws

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// MonitorTCP provides the main entrypoint into the package.
type MonitorTCP struct {
	Broker    Broker
	Observers []Observer
	Targets   []string
}

// NewMonitorTCP starts a new Broker and kicks off an Observer for each Target provided.
// Target should be a slice of strings of valid IPv4 addresses and ports.
func NewMonitorTCP(targets []string) *MonitorTCP {
	broker := NewBroker()
	go broker.Start()
	// Create observers
	observers := make([]Observer, len(targets))
	for i, target := range targets {
		// address string, timeout time.Duration, broker *Broker
		timeout := 100 * time.Millisecond
		o, oErr := NewTCPObserverFromString(target, timeout, broker)
		if oErr != nil {
			log.Printf("Ignoring bad target: %s", target)
			continue
		}
		go o.Start()
		observers[i] = o
	}
	return &MonitorTCP{
		Broker:    *broker,
		Targets:   targets,
		Observers: observers,
	}
}

// HandleWS is a pluggable websocket handler
// that subscribes to the MonitorTCP's Broker
// and forwards published messages over the websocket connection.
func (m *MonitorTCP) HandleWS(w http.ResponseWriter, r *http.Request) {
	if !websocket.IsWebSocketUpgrade(r) {
		log.Println("No upgrade requested")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Request websocket upgrade"))
		return
	}
	conn, upgradeErr := upgrader.Upgrade(w, r, nil)
	if upgradeErr != nil {
		log.Print("Could not upgrade connection: ", upgradeErr)
		return
	}
	defer conn.Close()
	// Get pings
	sub := m.Broker.Subscribe()
	defer m.Broker.Unsubscribe(sub)
	for {
		// On receive, send on conn
		select {
		case update := <-sub:
			// Publish to websocket connection
			conn.WriteJSON(update)
		default:
		}
	}
	log.Print("Exit")
}

// Stop halts goroutines kicked off by the MonitorTCP.
func (m *MonitorTCP) Stop() {
	m.Broker.Stop()
	for _, observer := range m.Observers {
		if observer != nil {
			observer.Stop()
		}
	}
}
