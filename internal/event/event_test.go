package event

import (
	"testing"
	"time"
)

func TestEventBus_Subscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var received bool
	bus.Subscribe(EventLoginSuccess, func(event Event) {
		received = true
	})

	bus.Publish(Event{
		Type:      EventLoginSuccess,
		Timestamp: time.Now().UnixMilli(),
		Data:      nil,
	})

	time.Sleep(10 * time.Millisecond)
	if !received {
		t.Error("Event handler was not called")
	}
}

func TestEventBus_MultipleHandlers(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	count := 0
	handler := func(event Event) {
		count++
	}

	bus.Subscribe(EventDownloadCompleted, handler)
	bus.Subscribe(EventDownloadCompleted, handler)

	bus.Publish(Event{
		Type:      EventDownloadCompleted,
		Timestamp: time.Now().UnixMilli(),
		Data:      nil,
	})

	time.Sleep(10 * time.Millisecond)
	if count != 2 {
		t.Errorf("Expected 2 handlers to be called, got %d", count)
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	count := 0
	handler := func(event Event) {
		count++
	}

	bus.Subscribe(EventUploadCompleted, handler)
	bus.Unsubscribe(EventUploadCompleted, handler)

	bus.Publish(Event{
		Type:      EventUploadCompleted,
		Timestamp: time.Now().UnixMilli(),
		Data:      nil,
	})

	time.Sleep(10 * time.Millisecond)
	if count != 0 {
		t.Error("Unsubscribed handler should not be called")
	}
}
