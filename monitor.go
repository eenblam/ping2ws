package ping2ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Monitor provides the main entrypoint into the package.
type Monitor struct {
	Broker    Broker
	Observers []*Observer
	Targets   []string
}

// NewMonitor starts a new Broker and kicks off an Observer for each Target provided.
// Target should be a slice of strings of valid IPv4 addresses.
func NewMonitor(targets []string) *Monitor {
	broker := NewBroker()
	go broker.Start()
	// Create observers
	observers := make([]*Observer, len(targets))
	for i, target := range targets {
		o, oErr := NewObserver(target, broker)
		if oErr != nil {
			log.Printf("Ignoring bad target: %s", target)
			continue
		}
		go o.Start()
		observers[i] = o
	}
	return &Monitor{
		Broker:    *broker,
		Targets:   targets,
		Observers: observers,
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// PingHandler is a pluggable websocket handler
// that subscribes to the Monitor's Broker
// and forwards published messages over the websocket connection.
func (m *Monitor) PingHandler(w http.ResponseWriter, r *http.Request) {
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

// Stop halts goroutines kicked off by the Monitor.
func (m *Monitor) Stop() {
	m.Broker.Stop()
	for _, observer := range m.Observers {
		if observer != nil {
			observer.Stop()
		}
	}
}
