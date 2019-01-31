package main

import (
	"fmt"
	"gitlab.com/jonas.jasas/condchan"
	"sync"
	"time"
)

func main() {
	fmt.Println("Timeout example")
	cc := condchan.New(&sync.Mutex{})
	timeoutChan := time.After(time.Second)

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
