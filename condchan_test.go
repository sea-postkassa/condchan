package condchan_test

import (
	"gitlab.com/jonas.jasas/condchan"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestSelect(t *testing.T) {
	cc := condchan.New(&sync.Mutex{})
	go func() {
		time.Sleep(time.Millisecond * 10)
		cc.L.Lock()
		cc.Signal()
		cc.Signal()
		cc.L.Unlock()
	}()

	waitFinishChan := make(chan struct{})
	go func() { // Standard cond waiting
		cc.L.Lock()
		cc.Wait()
		cc.L.Unlock()
		close(waitFinishChan)
	}()

	cancelChan := time.After(time.Millisecond * 20)
	cc.L.Lock()
	cc.Select(func(c <-chan struct{}) { // Waiting with select
		select {
		case <-cancelChan:
			t.Fail()
		case <-c:
		}
	})
	cc.L.Unlock()

	<-waitFinishChan
}

func TestWaitCancel(t *testing.T) {
	cc := condchan.New(&sync.Mutex{})

	cancelChan := time.After(time.Millisecond * 10)
	cc.L.Lock()
	cc.Select(func(c <-chan struct{}) { // Waiting with select
		select {
		case <-cancelChan:
		case <-c:
			t.Fail()
		}
	})
	cc.L.Unlock()
}

func TestSelectBroadcast(t *testing.T) {
	cc := condchan.New(&sync.Mutex{})

	cancelChan := time.After(time.Millisecond * 100)
	startChan := make(chan struct{})
	finishChan := make(chan struct{})
	const waiterCnt = 100
	for i := 0; i < waiterCnt; i++ {
		go func() {
			cc.L.Lock()
			startChan <- struct{}{}
			cc.Select(func(c <-chan struct{}) { // Waiting with select
				select {
				case <-cancelChan:
					t.Fail()
				case <-c:
				}
			})
			cc.L.Unlock()
			finishChan <- struct{}{}
		}()
	}

	for i := 0; i < waiterCnt; i++ {
		<-startChan
	}

	cc.L.Lock()
	cc.Broadcast()
	cc.L.Unlock()

	for i := 0; i < waiterCnt; i++ {
		<-finishChan
	}
}

func TestSignBroadRace(t *testing.T) {
	cond := condchan.New(&sync.Mutex{})
	finishChan := make(chan struct{})
	const times = 1000000

	go func() {
		for i := 0; i < times; i++ {
			cond.Signal()
		}
		finishChan <- struct{}{}
	}()

	go func() {
		for i := 0; i < times; i++ {
			cond.Broadcast()
		}
		finishChan <- struct{}{}
	}()

	<-finishChan
	<-finishChan
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Tests below are taken from sync.Cond package ////////////////////////////////////////////////////////////////////////

func TestCondSignal(t *testing.T) {
	var m sync.Mutex
	c := condchan.New(&m)
	n := 2
	running := make(chan bool, n)
	awake := make(chan bool, n)
	for i := 0; i < n; i++ {
		go func() {
			m.Lock()
			running <- true
			c.Wait()
			awake <- true
			m.Unlock()
		}()
	}
	for i := 0; i < n; i++ {
		<-running // Wait for everyone to run.
	}
	for n > 0 {
		select {
		case <-awake:
			t.Fatal("goroutine not asleep")
		default:
		}
		m.Lock()
		c.Signal()
		m.Unlock()
		<-awake // Will deadlock if no goroutine wakes up
		select {
		case <-awake:
			t.Fatal("too many goroutines awake")
		default:
		}
		n--
	}
	c.Signal()
}

func TestCondSignalGenerations(t *testing.T) {
	var m sync.Mutex
	c := condchan.New(&m)
	n := 100
	running := make(chan bool, n)
	awake := make(chan int, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			m.Lock()
			running <- true
			c.Wait()
			awake <- i
			m.Unlock()
		}(i)
		if i > 0 {
			a := <-awake
			if a != i-1 {
				t.Fatalf("wrong goroutine woke up: want %d, got %d", i-1, a)
			}
		}
		<-running
		m.Lock()
		c.Signal()
		m.Unlock()
	}
}

func TestCondBroadcast(t *testing.T) {
	var m sync.Mutex
	c := condchan.New(&m)
	n := 200
	running := make(chan int, n)
	awake := make(chan int, n)
	exit := false
	for i := 0; i < n; i++ {
		go func(g int) {
			m.Lock()
			for !exit {
				running <- g
				c.Wait()
				awake <- g
			}
			m.Unlock()
		}(i)
	}
	for i := 0; i < n; i++ {
		for i := 0; i < n; i++ {
			<-running // Will deadlock unless n are running.
		}
		if i == n-1 {
			m.Lock()
			exit = true
			m.Unlock()
		}
		select {
		case <-awake:
			t.Fatal("goroutine not asleep")
		default:
		}
		m.Lock()
		c.Broadcast()
		m.Unlock()
		seen := make([]bool, n)
		for i := 0; i < n; i++ {
			g := <-awake
			if seen[g] {
				t.Fatal("goroutine woke up twice")
			}
			seen[g] = true
		}
	}
	select {
	case <-running:
		t.Fatal("goroutine did not exit")
	default:
	}
	c.Broadcast()
}

func TestRace(t *testing.T) {
	x := 0
	c := condchan.New(&sync.Mutex{})
	done := make(chan bool)
	go func() {
		c.L.Lock()
		x = 1
		c.Wait()
		if x != 2 {
			t.Error("want 2")
		}
		x = 3
		c.Signal()
		c.L.Unlock()
		done <- true
	}()
	go func() {
		c.L.Lock()
		for {
			if x == 1 {
				x = 2
				c.Signal()
				break
			}
			c.L.Unlock()
			runtime.Gosched()
			c.L.Lock()
		}
		c.L.Unlock()
		done <- true
	}()
	go func() {
		c.L.Lock()
		for {
			if x == 2 {
				c.Wait()
				if x != 3 {
					t.Error("want 3")
				}
				break
			}
			if x == 3 {
				break
			}
			c.L.Unlock()
			runtime.Gosched()
			c.L.Lock()
		}
		c.L.Unlock()
		done <- true
	}()
	<-done
	<-done
	<-done
}

func TestCondSignalStealing(t *testing.T) {
	for iters := 0; iters < 1000; iters++ {
		var m sync.Mutex
		cond := condchan.New(&m)

		// Start a waiter.
		ch := make(chan struct{})
		go func() {
			m.Lock()
			ch <- struct{}{}
			cond.Wait()
			m.Unlock()

			ch <- struct{}{}
		}()

		<-ch
		m.Lock()
		m.Unlock()

		// We know that the waiter is in the cond.Wait() call because we
		// synchronized with it, then acquired/released the mutex it was
		// holding when we synchronized.
		//
		// Start two goroutines that will race: one will broadcast on
		// the cond var, the other will wait on it.
		//
		// The new waiter may or may not get notified, but the first one
		// has to be notified.
		done := false
		go func() {
			cond.Broadcast()
		}()

		go func() {
			m.Lock()
			for !done {
				cond.Wait()
			}
			m.Unlock()
		}()

		// Check that the first waiter does get signaled.
		select {
		case <-ch:
		case <-time.After(2 * time.Second):
			t.Fatalf("First waiter didn't get broadcast.")
		}

		// Release the second waiter in case it didn't get the
		// broadcast.
		m.Lock()
		done = true
		m.Unlock()
		cond.Broadcast()
	}
}

func TestCondCopy(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil || err.(string) != "sync.Cond is copied" {
			t.Fatalf("got %v, expect sync.Cond is copied", err)
		}
	}()
	c := condchan.CondChan{L: &sync.Mutex{}}
	c.Signal()
	var c2 condchan.CondChan
	reflect.ValueOf(&c2).Elem().Set(reflect.ValueOf(&c).Elem()) // c2 := c, hidden from vet
	c2.Signal()
}

func BenchmarkCond1(b *testing.B) {
	benchmarkCond(b, 1)
}

func BenchmarkCond2(b *testing.B) {
	benchmarkCond(b, 2)
}

func BenchmarkCond4(b *testing.B) {
	benchmarkCond(b, 4)
}

func BenchmarkCond8(b *testing.B) {
	benchmarkCond(b, 8)
}

func BenchmarkCond16(b *testing.B) {
	benchmarkCond(b, 16)
}

func BenchmarkCond32(b *testing.B) {
	benchmarkCond(b, 32)
}

func benchmarkCond(b *testing.B, waiters int) {
	c := condchan.New(&sync.Mutex{})
	done := make(chan bool)
	id := 0

	for routine := 0; routine < waiters+1; routine++ {
		go func() {
			for i := 0; i < b.N; i++ {
				c.L.Lock()
				if id == -1 {
					c.L.Unlock()
					break
				}
				id++
				if id == waiters+1 {
					id = 0
					c.Broadcast()
				} else {
					c.Wait()
				}
				c.L.Unlock()
			}
			c.L.Lock()
			id = -1
			c.Broadcast()
			c.L.Unlock()
			done <- true
		}()
	}
	for routine := 0; routine < waiters+1; routine++ {
		<-done
	}
}
