package event

import (
	"context"
	"reflect"
	"sync"
	"time"
)

type EventType string

const (
	EventLoginSuccess       EventType = "login_success"
	EventLoginFailure       EventType = "login_failure"
	EventTokenRefreshed     EventType = "token_refreshed"
	EventTokenRefreshFailed EventType = "token_refresh_failed"
	EventDownloadStarted    EventType = "download_started"
	EventDownloadProgress   EventType = "download_progress"
	EventDownloadCompleted  EventType = "download_completed"
	EventDownloadFailed     EventType = "download_failed"
	EventUploadStarted      EventType = "upload_started"
	EventUploadProgress     EventType = "upload_progress"
	EventUploadCompleted    EventType = "upload_completed"
	EventUploadFailed       EventType = "upload_failed"
	EventFileCreated        EventType = "file_created"
	EventFileDeleted        EventType = "file_deleted"
	EventShareCreated       EventType = "share_created"
	EventShareDeleted       EventType = "share_deleted"
	EventError              EventType = "error"
)

type Event struct {
	Type      EventType
	Timestamp int64
	Data      map[string]interface{}
	Error     error
}

type EventHandler func(event Event)

type handlerWrapper struct {
	id      int
	handler EventHandler
}

type EventBus struct {
	handlers      map[EventType][]handlerWrapper
	nextHandlerID int
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewEventBus() *EventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventBus{
		handlers:      make(map[EventType][]handlerWrapper),
		nextHandlerID: 1,
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) int {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handlerWrapper{
		id:      eb.nextHandlerID,
		handler: handler,
	})
	id := eb.nextHandlerID
	eb.nextHandlerID++
	return id
}

func (eb *EventBus) UnsubscribeByID(eventType EventType, id int) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	handlers := eb.handlers[eventType]
	for i, h := range handlers {
		if h.id == id {
			eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

func (eb *EventBus) Unsubscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	handlers := eb.handlers[eventType]
	for i, h := range handlers {
		if reflect.ValueOf(h.handler).Pointer() == reflect.ValueOf(handler).Pointer() {
			eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	event.Timestamp = CurrentTimestamp()
	wrappers := eb.handlers[event.Type]
	for _, wrapper := range wrappers {
		go wrapper.handler(event)
	}
}

func (eb *EventBus) Close() {
	eb.cancel()
}

func CurrentTimestamp() int64 {
	return time.Now().UnixMilli()
}
