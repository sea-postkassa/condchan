# CondChan cancellable [sync.Cond](https://golang.org/pkg/sync/#Cond)  

CondChan is a [sync.Cond](https://golang.org/pkg/sync/#Cond) with the ability to [wait](https://golang.org/pkg/sync/#Cond.Wait) in [select](https://tour.golang.org/concurrency/5) statement. 

## Facts

* Adds waiting in [select](https://tour.golang.org/concurrency/5) statement feature
* Implements all [sync.Cond](https://golang.org/pkg/sync/#Cond) interface
* Passes all [sync.Cond](https://golang.org/pkg/sync/#Cond) tests
* Implemented using channels
* Just ~36% slower comparing to [sync.Cond](https://golang.org/pkg/sync/#Cond) 


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
Signaling channel is provided inside *func* that should be used in select statement together with other channels that can interrupt waiting.   

```go
func timeoutExample()  {
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
```

### Broadcast example

```go
func broadcastExample()  {
	cc := condchan.New(&sync.Mutex{})

	var jobResult string
	go func() {
		time.Sleep(time.Second*2)  // Imitating long job
		cc.L.Lock()
		jobResult = "CANDY"
		cc.L.Unlock()
		cc.Broadcast()	// Letting know waiter that job is done
	}()

	go waiter(cc, "Patience", time.Second*3, &jobResult)
	go waiter(cc, "Impatience", time.Second*1, &jobResult)

	cc.L.Lock()
	cc.Wait()	// Waiting like on sync.Cond
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