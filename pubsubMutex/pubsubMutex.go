package pubsubmutex

import (
	"sync"
	"time"

	"github.com/cskr/pubsub"
)

type PubSubMutex struct {
	pubsub    *pubsub.PubSub
	capacity  int
	isRunning bool
	guard     sync.RWMutex
}

func New(capacity int) *PubSubMutex {
	p := PubSubMutex{
		pubsub:    nil,
		capacity:  capacity,
		isRunning: false,
		guard:     sync.RWMutex{},
	}
	return &p
}

func (p *PubSubMutex) Start() {
	p.guard.Lock()
	defer p.guard.Unlock()
	p.shutdown()
	p.pubsub = pubsub.New(p.capacity)
	p.isRunning = true
}

func (p *PubSubMutex) Use(callback func(*pubsub.PubSub)) {
	p.guard.RLock()
	defer p.guard.RUnlock()
	if callback != nil && p.pubsub != nil && p.isRunning {
		callback(p.pubsub)
	}
}

func (p *PubSubMutex) Shutdown() {
	p.guard.Lock()
	defer p.guard.Unlock()
	p.shutdown()
}
func (p *PubSubMutex) shutdown() {
	if p.pubsub != nil && p.isRunning {
		p.pubsub.Shutdown()
		p.pubsub = nil
	}
	p.isRunning = false
}

func (p *PubSubMutex) Sub(subTopic string) (result <-chan interface{}) {
	p.Use(func(instance *pubsub.PubSub) {
		result = instance.Sub(subTopic)
	})
	return
}

func (p *PubSubMutex) SubAsync(subTopic string) (result <-chan interface{}) {
	r := make(chan interface{})
	result = r
	go p.Use(func(instance *pubsub.PubSub) {
		instance.AddSub(r, subTopic)
	})
	return
}

func (p *PubSubMutex) SendReceive(sendTopic string, receiveTopic string,
	sendMsg interface{},
	timeoutMs int) (result interface{}) {
	curChan := make(chan interface{})
	go p.Use(func(instance *pubsub.PubSub) {
		instance.AddSubOnceEach(curChan, receiveTopic)
		instance.TryPub(sendMsg, sendTopic)
	})
	select {
	case r, ok := <-curChan:
		if ok {
			result = r
		}
	case <-time.After(time.Millisecond * time.Duration(timeoutMs)):
		go p.Use(func(instance *pubsub.PubSub) {
			instance.Unsub(curChan, receiveTopic)
		})
	}
	return
}
