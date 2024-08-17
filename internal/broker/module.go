package broker

import (
	"context"
	"sync"
	"therealbroker/pkg/broker"
	"time"
)

type Module struct {
	// TODO: Add required fields
	subscriptions map[string][]chan broker.Message
	messages      map[string]map[int]broker.Message
	messageTimes  map[int]time.Time
	mu            sync.RWMutex
	nextID        int
	closed        bool
}

func NewModule() broker.Broker {
	return &Module{
		subscriptions: make(map[string][]chan broker.Message),
		messages:      make(map[string]map[int]broker.Message),
		nextID:        0,
		closed:        false,
	}
}

func (m *Module) Close() error {
	// panic("implement me")
	if m.closed {
		return broker.ErrUnavailable
	}
	m.closed = true
	for _, subs := range m.subscriptions {
		for _, sub := range subs {
			close(sub)
		}
	}
	return nil
}

func (m *Module) Publish(ctx context.Context, subject string, msg broker.Message) (int, error) {
	if m.closed {
		return 0, broker.ErrUnavailable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := m.nextID

	if m.messageTimes == nil {
		m.messageTimes = make(map[int]time.Time)
	}
	m.messageTimes[id] = time.Now()

	// Store the message
	if _, exists := m.messages[subject]; !exists {
		m.messages[subject] = make(map[int]broker.Message)
	}
	m.messages[subject][id] = msg

	// Send the message to all subscribers
	if subs, exists := m.subscriptions[subject]; exists {
		for _, sub := range subs {
			select {
			case sub <- msg:
				// Message sent successfully
			default:
				// Subscriber Channel full or blocked etc handle as needed
			}
		}
	}

	return id, nil
	// panic("implement me")
}

func (m *Module) Subscribe(ctx context.Context, subject string) (<-chan broker.Message, error) {
	// panic("implement me")
	if m.closed {
		return nil, broker.ErrUnavailable
	}
	// 50 Set based on unit test line 94(TestPublishShouldPreserveOrder) change as needed
	ch := make(chan broker.Message, 50)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.subscriptions[subject]; !exists {
		m.subscriptions[subject] = []chan broker.Message{}
	}
	m.subscriptions[subject] = append(m.subscriptions[subject], ch)

	return ch, nil
}

func (m *Module) Fetch(ctx context.Context, subject string, id int) (broker.Message, error) {
	// panic("implement me")
	if m.closed {
		return broker.Message{}, broker.ErrUnavailable
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if msgs, exists := m.messages[subject]; exists {
		if msg, found := msgs[id]; found {
			if creationTime, exists := m.messageTimes[id]; exists {
				expirationTime := creationTime.Add(msg.Expiration)
				if time.Now().After(expirationTime) {
					return broker.Message{}, broker.ErrExpiredID
				}
			}
			return msg, nil
		}
	}
	return broker.Message{}, broker.ErrInvalidID
}
