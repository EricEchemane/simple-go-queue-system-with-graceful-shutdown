package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var q chan int
var wg sync.WaitGroup

func worker(item int) {
	defer wg.Done()
	delay := rand.Intn(7)
	log.Printf("ongoing: %d received processing in %ds\n", item, delay)
	time.Sleep(time.Second * time.Duration(delay))
	q <- item
}

func index(w http.ResponseWriter, req *http.Request) {
	wg.Add(1)
	go worker(rand.Int())
	fmt.Fprintf(w, "hello\n")
}

func main() {
	q = make(chan int)
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case item := <-q:
				log.Printf("done: %d processed\n", item)
			case <-time.After(time.Millisecond):
			}
		}
	}()

	http.HandleFunc("/", index)
	go http.ListenAndServe(":8080", nil)
	log.Println("Running on port 8080")

	<-shutdown
	log.Println("Shutting down gracefully...")

	log.Println("Waiting for other workers to complete")
	wg.Wait()
}
