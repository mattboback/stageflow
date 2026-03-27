package jobstatus

import (
	"sync"
)

type watcherBroker struct {
	mu       sync.RWMutex
	watchers map[string]map[*watchSubscription]struct{}
}

type watchSubscription struct {
	jobID     string
	updates   chan Change
	broker    *watcherBroker
	closeOnce sync.Once
}

func newWatcherBroker() *watcherBroker {
	return &watcherBroker{watchers: make(map[string]map[*watchSubscription]struct{})}
}

func (b *watcherBroker) Subscribe(jobID string) Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := &watchSubscription{
		jobID:   jobID,
		updates: make(chan Change, 16),
		broker:  b,
	}

	if b.watchers[jobID] == nil {
		b.watchers[jobID] = make(map[*watchSubscription]struct{})
	}

	b.watchers[jobID][sub] = struct{}{}

	return sub
}

func (b *watcherBroker) Publish(change Change) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	watchers := b.watchers[change.JobID]
	if len(watchers) == 0 {
		return
	}

	broadcast := change
	broadcast.Snapshot = cloneJobRecord(change.Snapshot)

	for sub := range watchers {
		sendChangeWithEviction(sub.updates, broadcast)
	}
}

func (b *watcherBroker) unsubscribe(sub *watchSubscription) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, ok := b.watchers[sub.jobID]; ok {
		delete(subs, sub)
		if len(subs) == 0 {
			delete(b.watchers, sub.jobID)
		}
	}

	close(sub.updates)
}

func (s *watchSubscription) Updates() <-chan Change {
	return s.updates
}

func (s *watchSubscription) Close() error {
	s.closeOnce.Do(func() {
		s.broker.unsubscribe(s)
	})

	return nil
}

func sendChangeWithEviction(ch chan Change, change Change) {
	select {
	case ch <- change:
		return
	default:
	}

	select {
	case <-ch:
	default:
	}

	select {
	case ch <- change:
	default:
	}
}
