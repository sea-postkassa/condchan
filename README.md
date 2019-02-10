# CondChan cancellable [sync.Cond](https://golang.org/pkg/sync/#Cond)  

CondChan is a [sync.Cond](https://golang.org/pkg/sync/#Cond) with the ability to [wait](https://golang.org/pkg/sync/#Cond.Wait) in [select](https://tour.golang.org/concurrency/5) statement. 

* Adds waiting in [select](https://tour.golang.org/concurrency/5) statement feature
* Implements all [sync.Cond](https://golang.org/pkg/sync/#Cond) interface
* Passes all [sync.Cond](https://golang.org/pkg/sync/#Cond) tests
* Implemented using channels
* Just ~37% slower comparing to [sync.Cond](https://golang.org/pkg/sync/#Cond) 

[![go report card](https://goreportcard.com/badge/gitlab.com/jonas.jasas/condchan)](https://goreportcard.com/report/gitlab.com/jonas.jasas/condchan)
[![pipeline status](https://gitlab.com/jonas.jasas/condchan/badges/master/pipeline.svg)](https://gitlab.com/jonas.jasas/condchan/commits/master)
[![coverage report](https://gitlab.com/jonas.jasas/condchan/badges/master/coverage.svg)](https://gitlab.com/jonas.jasas/condchan/commits/master)
[![godoc](https://godoc.org/gitlab.com/jonas.jasas/condchan?status.svg)](http://godoc.org/gitlab.com/jonas.jasas/condchan)


## Installation
Simple install the package to your [$GOPATH](https://github.com/golang/go/wiki/GOPATH "GOPATH") with the [go tool](https://golang.org/cmd/go/ "go command") from shell:
```bash
$ go get gitlab.com/jonas.jasas/condchan
```
Make sure [Git is installed](https://git-scm.com/downloads) on your machine and in your system's `PATH`.


## Usage examples

### Timeout example

In this example CondChan is created and endlessly waiting on it.
*Select* method is used for the waiting.
Method accepts *func* argument.
Signaling channel is provided inside *func* that should be used in select statement together with the other channels that can interrupt waiting.   

```go
func timeoutExample()  {
	cc := condchan.New(&sync.Mutex{})
	timeoutChan := time.After(time.Second)

	cc.L.Lock()
	// Passing func that gets channel c that signals when
	// Signal or Broadcast is called on CondChan
	cc.Select(func(c <-chan struct{}) { // Waiting with select
		select {
		case <-c:   // Never ending wait
		case <-timeoutChan:
			fmt.Println("Hooray! Just escaped from the eternal wait.")
		}
	})
	cc.L.Unlock()
}
```

Output:
```text
Hooray! Just escaped from the eternal wait.
```

### Broadcast example

This example is using *Wait* and *Select* methods for simultaneous wait on same CondChan.
Waiter "Patience" is waiting 3s and waiter "Impatience" is waiting 1s while job is done in 2s.
Waiter "Impatience" will be cancelled with *timeoutChan*.
Waiter "Patience" will receive signal on channel *c*.

```go
func broadcastExample()  {
	cc := condchan.New(&sync.Mutex{})

	var jobResult string
	go func() {
		time.Sleep(time.Second*2)  // Imitating long job
		cc.L.Lock()
		jobResult = "CANDY"
		cc.L.Unlock()
		cc.Broadcast()  // Letting know all waiters that job is done
	}()

	go waiter(cc, "Patience", time.Second*3, &jobResult)
	go waiter(cc, "Impatience", time.Second*1, &jobResult)

	cc.L.Lock()
	cc.Wait()   // Waiting like on sync.Cond
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
```

Output:
```text
2019/01/27 22:48:02 Impatience: I can't wait much longer
2019/01/27 22:48:03 Patience: I received what I been waiting for - CANDY
```