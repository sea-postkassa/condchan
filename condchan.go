package condchan

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type CondChan struct {
	noCopy noCopy

	L   sync.Locker
	ch  chan struct{}
	chL sync.RWMutex

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
	this.L.Unlock()

	this.chL.RLock()
	ch := this.ch
	this.chL.RUnlock()

	fn(ch)

	this.L.Lock()
}

func (this *CondChan) Wait() {
	//log.Print("WI")
	this.checker.check()
	this.L.Unlock()

	this.chL.RLock()
	ch := this.ch
	this.chL.RUnlock()
	<-ch

	this.L.Lock()
	//log.Print("WO")

}

func (this *CondChan) Signal() {
	this.checker.check()

	this.chL.Lock()
	select {
	case this.ch <- struct{}{}:
	default:
	}
	this.chL.Unlock()
}

func (this *CondChan) Broadcast() {
	this.checker.check()

	this.chL.Lock()
	close(this.ch)
	this.ch = make(chan struct{})
	this.chL.Unlock()

	//this.chL.Lock()
	//for more := true; more; {
	//	select {
	//	case this.ch <- struct{}{}:
	//	default:
	//		more = false
	//	}
	//}
	//this.chL.Unlock()
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
