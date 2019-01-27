package main

import (
	"fmt"
	"gitlab.com/jonas.jasas/condchan"
	"log"
	"sync"
	"time"
)

func timeoutExample() {
	cc := condchan.New(&sync.Mutex{})
	timeoutChan := time.After(time.Second)

	go func() {}()

	cc.L.Lock()
	// Passing func that gets channel c that signals when
	// Signal or Broadcast is called on CondChan
	cc.Select(func(c <-chan struct{}) { // Waiting with select
		select {
		case <-c: // Never ending wait
		case <-timeoutChan:
			fmt.Println("Hooray! Just escaped from eternal wait.")
		}
	})
	cc.L.Unlock()
}

func broadcastExample() {
	cc := condchan.New(&sync.Mutex{})

	var jobResult string
	go func() {
		time.Sleep(time.Second * 2) // Imitating long job
		cc.L.Lock()
		jobResult = "CANDY"
		cc.L.Unlock()
		cc.Broadcast() // Letting know waiter that job is done
	}()

	go waiter(cc, "Patience", time.Second*3, &jobResult)
	go waiter(cc, "Impatience", time.Second*1, &jobResult)

	cc.L.Lock()
	cc.Wait() // Waiting like on sync.Cond
	cc.L.Unlock()
	time.Sleep(time.Second)
}

func waiter(cc *condchan.CondChan, name string, wait time.Duration, jobResult *string) {
	timeoutChan := time.After(wait)
	cc.L.Lock()
	cc.Select(func(c <-chan struct{}) { // Waiting with select
		select {
		case <-c:
			log.Printf("%s: I received what I been waiting for - %s", name, *jobResult)
		case <-timeoutChan:
			log.Printf("%s: I can't wait much longer", name)
		}
	})
	cc.L.Unlock()
}

func main() {
	//timeoutExample()
	broadcastExample()
}
