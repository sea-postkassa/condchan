package condchan

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type CondChan struct {
	L   sync.Locker
	ch  chan struct{}
	chL sync.RWMutex

	noCopy  noCopy
	checker copyChecker
}

type selectFn func(<-chan struct{})

func New(l sync.Locker) *CondChan {
	return &CondChan{
		L:   l,
		ch:  make(chan struct{}),
		chL: sync.RWMutex{},
	}
}

func (this *CondChan) Select(fn selectFn) {
	this.checker.check()

	this.chL.RLock()
	ch := this.ch
	this.chL.RUnlock()

	this.L.Unlock()
	fn(ch)
	this.L.Lock()
}

func (this *CondChan) Wait() {
	this.checker.check()

	this.chL.RLock()
	ch := this.ch
	this.chL.RUnlock()

	this.L.Unlock()
	<-ch
	this.L.Lock()
}

func (this *CondChan) Signal() {
	this.checker.check()

	this.chL.RLock()
	ch := this.ch
	this.chL.RUnlock()

	select {
	case ch <- struct{}{}:
	default:
	}
}

func (this *CondChan) Broadcast() {
	this.checker.check()

	this.chL.Lock()
	close(this.ch)
	this.ch = make(chan struct{})
	this.chL.Unlock()
}

// Code borrowed from sync.cond ////////////////////////////////////////////////////////////////////////////////////////

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
