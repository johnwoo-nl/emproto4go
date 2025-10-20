package internal

import (
	"github.com/johnwoo-nl/emproto4go/types"
)

type EventWatcher struct {
	// Communicator that created this watcher. Needed to remove from the watchers list when Stop() is called.
	communicator *Communicator

	// Optional, if nil the watcher is registered for all EVSEs.
	evse *Evse

	// Which event types to watch for.
	eventTypes []types.EmEventType

	// Channel on which events are sent (send-only for internal use).
	channel chan<- types.EmEvent

	// Whether the watcher has been stopped (Stop() has been called or the watcher stopped itself due to
	// an issue sending events to the channel).
	stopped bool
}

func (watcher *EventWatcher) Stop() {
	watcher.stopped = true
	close(watcher.channel)
	watcher.communicator.RemoveWatcher(watcher)
}

func (watcher *EventWatcher) IsStopped() bool {
	return watcher.stopped
}

func (watcher *EventWatcher) Evse() types.EmEvse {
	return watcher.evse
}

func (watcher *EventWatcher) EventTypes() []types.EmEventType {
	return watcher.eventTypes
}

func (watcher *EventWatcher) Notify(event types.EmEvent) {
	if watcher.stopped {
		watcher.communicator.RemoveWatcher(watcher)
		return
	}

	// Check if event matches the watcher EVSE.
	if watcher.evse != nil && event.Evse.Serial() != watcher.evse.Serial() {
		return
	}

	// Check if event matches the watcher event types.
	if len(watcher.eventTypes) > 0 {
		found := false
		for _, t := range watcher.eventTypes {
			if t == event.Type {
				found = true
				break
			}
		}
		if !found {
			return
		}
	}

	// Send event to channel (non-blocking).
	select {
	case watcher.channel <- event:
		// sent successfully
	default:
		// channel full, not ready or closed
		watcher.Stop()
	}
}
