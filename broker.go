package ping2ws

// Courtesy of github.com/icza || stackoverflow.com/users/1705598/icza
// See https://stackoverflow.com/a/49877632

type Broker struct {
	// stopCh is used to signal the broker to halt
	stopCh chan struct{}
	// publishCh receives messages to the broker, which are then forwarded to subscribers
	publishCh chan interface{}
	// subCh receives channels from new subscribers
	subCh chan chan interface{}
	// unsubCh receives channels from subscribers when they're ready to unsubscribe
	unsubCh chan chan interface{}
}

func NewBroker() *Broker {
	return &Broker{
		stopCh:    make(chan struct{}),
		publishCh: make(chan interface{}, 1),
		subCh:     make(chan chan interface{}, 1),
		unsubCh:   make(chan chan interface{}, 1),
	}
}

func (b *Broker) Start() {
	// Track subscriptions in a map
	subs := map[chan interface{}]struct{}{}
	for {
		select {
		case <-b.stopCh:
			return
		case msgCh := <-b.subCh:
			// Add a new subscriber
			subs[msgCh] = struct{}{}
		case msgCh := <-b.unsubCh:
			// Remove an existing subscriber if found
			delete(subs, msgCh)
		case msg := <-b.publishCh:
			// Someone published a message to broker. Forward to subscribers.
			for msgCh := range subs {
				// msgCh should be buffered. Use non-blocking send to protect broker regardless.
				// See https://gobyexample.com/non-blocking-channel-operations
				// We're dealing with pings, so it's no big deal if packets get dropped here.
				select {
				case msgCh <- msg:
				default:
				}
			}
		}
	}
}

// Stop closes the broker's stop channel.
// The channel closure is received in Start, causing it to exit.
func (b *Broker) Stop() {
	close(b.stopCh)
}

// Subscribe creates a new subscription channel,
// passes it to the broker, and returns it to the subscriber.
func (b *Broker) Subscribe() chan interface{} {
	msgCh := make(chan interface{}, 5)
	b.subCh <- msgCh
	return msgCh
}

// Unsubscribe removes a channel from the broker's collection of subscribers,
// if present.
func (b *Broker) Unsubscribe(msgCh chan interface{}) {
	b.unsubCh <- msgCh
}

// Publish accepts a msg to be forwarded to all subscribers.
func (b *Broker) Publish(msg interface{}) {
	b.publishCh <- msg
}
