package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
)

var (
	seededRNG = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type SmartLock struct {
	realLock    sync.Mutex
	queueLength int64
	lockID      string
	cont        bool
	acquired    time.Time
	activeSpan  opentracing.Span
}

func NewSmartLock(cont bool) *SmartLock {
	return &SmartLock{
		realLock: sync.Mutex{},
		lockID:   fmt.Sprintf("smart_lock-%v", seededRNG.Int63()),
		cont:     cont,
	}
}

func (sl *SmartLock) Lock(activeSpan opentracing.Span) {
	if sl.cont {
		activeSpan.SetTag("c:", sl.lockID)
	}
	atomic.AddInt64(&sl.queueLength, 1)
	sl.realLock.Lock()
	sl.activeSpan = activeSpan
	atomic.AddInt64(&sl.queueLength, -1)
	sl.acquired = time.Now()
}

func (sl *SmartLock) Unlock() {
	released := time.Now()
	sl.activeSpan.SetTag("weight", int(released.Sub(sl.acquired).Seconds()*1000.0+1))
	sl.realLock.Unlock()
}

func (sl *SmartLock) QueueLength() float64 {
	return float64(sl.queueLength)
}
