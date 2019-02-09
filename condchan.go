// CondChan is a sync.Cond with the ability to wait in select statement.
package condchan

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// CondChan implements a condition variable, a rendezvous point for goroutines waiting for or announcing the occurrence
// of an event.
//
// Each Cond has an associated Locker L (often a *Mutex or *RWMutex),
// which must be held when changing the condition and when calling the Wait method.
//
// A Cond must not be copied after first use.
type CondChan struct {
	L   sync.Locker
	ch  chan struct{}
	chL sync.RWMutex

	noCopy  noCopy
	checker copyChecker
}

type selectFn func(<-chan struct{})

// New returns a new CondChan with Locker l.
func New(l sync.Locker) *CondChan {
	return &CondChan{
		L:   l,
		ch:  make(chan struct{}),
		chL: sync.RWMutex{},
	}
}

// Select atomically unlocks cc.L and executes fn.
// After later resuming execution, Wait locks cc.L before returning.
//
// fn is executed passing channel in to it.
// Passed channel will signal by emitting struct{} or by closing.
// Inside fn should be select statement using passed channel together with the other channels that signals execution continuation.
func (cc *CondChan) Select(fn selectFn) {
	cc.checker.check()

	cc.chL.RLock()
	ch := cc.ch
	cc.chL.RUnlock()

	cc.L.Unlock()
	fn(ch)
	cc.L.Lock()
}

// Wait atomically unlocks cc.L and suspends execution of the calling goroutine.
// After later resuming execution, Wait locks cc.L before returning.
// Unlike in other systems, Wait cannot return unless awoken by Broadcast or Signal.
func (cc *CondChan) Wait() {
	cc.checker.check()

	cc.chL.RLock()
	ch := cc.ch
	cc.chL.RUnlock()

	cc.L.Unlock()
	<-ch
	cc.L.Lock()
}

// Signal wakes one goroutine waiting on cc, if there is any.
// It is allowed but not required for the caller to hold cc.L during the call.
func (cc *CondChan) Signal() {
	cc.checker.check()

	cc.chL.RLock()
	select {
	case cc.ch <- struct{}{}:
	default:
	}
	cc.chL.RUnlock()
}

// Broadcast wakes all goroutines waiting on cc.
// It is allowed but not required for the caller to hold cc.L during the call.
func (cc *CondChan) Broadcast() {
	cc.checker.check()

	cc.chL.Lock()
	close(cc.ch)
	cc.ch = make(chan struct{})
	cc.chL.Unlock()
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Below code is borrowed from sync.cond ///////////////////////////////////////////////////////////////////////////////

// copyChecker holds back pointer to itself to detect object copying.
type copyChecker uintptr

func (c *copyChecker) check() {
	if uintptr(*c) != uintptr(unsafe.Pointer(c)) &&
		!atomic.CompareAndSwapUintptr((*uintptr)(c), 0, uintptr(unsafe.Pointer(c))) &&
		uintptr(*c) != uintptr(unsafe.Pointer(c)) {
		panic("sync.Cond is copied")
	}
}

// noCopy may be embedded into structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
