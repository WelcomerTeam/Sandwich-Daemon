// Idea from: https://betterprogramming.pub/how-to-broadcast-messages-in-go-using-channels-b68f42bdf32e
//
// With grevolt-specific changes
package broadcast

import (
	"context"
)

type BroadcastServer[T any] struct {
	// The source channel
	Source chan T

	// Whether the broadcast server is open or not
	Open bool

	listeners      []chan T
	addListener    chan chan T
	removeListener chan (<-chan T)
	context        context.Context
	cancel         context.CancelFunc
}

// ListenersCount returns the number of listeners
func (s *BroadcastServer[T]) ListenersCount() int {
	return len(s.listeners)
}

func (s *BroadcastServer[T]) Broadcast(val T) {
	s.Source <- val
}

func (s *BroadcastServer[T]) Close() {
	s.Open = false
	s.cancel()
}

// Subscribe returns a new channel that will receive all broadcasts
func (s *BroadcastServer[T]) Subscribe() <-chan T {
	newListener := make(chan T)
	s.addListener <- newListener
	return newListener
}

// CancelSubscription cancels a subscription
//
// All channels returned by Subscribe() should be cancelled eventually
func (s *BroadcastServer[T]) CancelSubscription(channel <-chan T) {
	s.removeListener <- channel
}

func NewBroadcastServer[T any]() BroadcastServer[T] {
	ctx, cancel := context.WithCancel(context.Background())
	service := BroadcastServer[T]{
		Source:         make(chan T),
		context:        ctx,
		cancel:         cancel,
		listeners:      make([]chan T, 0),
		addListener:    make(chan chan T),
		removeListener: make(chan (<-chan T)),
		Open:           true,
	}
	go service.serve()
	return service
}

func (s *BroadcastServer[T]) serve() {
	defer func() {
		for _, listener := range s.listeners {
			if listener != nil {
				close(listener)
			}
		}
	}()

	for {
		select {
		case <-s.context.Done():
			return
		case newListener := <-s.addListener:
			s.listeners = append(s.listeners, newListener)
		case listenerToRemove := <-s.removeListener:
			for i, ch := range s.listeners {
				if ch == listenerToRemove {
					s.listeners[i] = s.listeners[len(s.listeners)-1]
					s.listeners = s.listeners[:len(s.listeners)-1]
					close(ch)
					break
				}
			}
		case val, ok := <-s.Source:
			if !ok {
				return
			}
			for _, listener := range s.listeners {
				if listener != nil {
					select {
					case listener <- val:
					case <-s.context.Done():
						return
					}
				}
			}
		}
	}
}
