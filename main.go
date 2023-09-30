package main

import (
	"context"
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
	item := rand.Int()
	go worker(item)
	time.Sleep(time.Second * 10) // simulate some process
	fmt.Fprintf(w, "%d", item)
}

func main() {
	q = make(chan int)
	go func() {
		for {
			select {
			case item := <-q:
				log.Printf("done: %d processed\n", item)
			case <-time.After(time.Millisecond):
			}
		}
	}()

	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/", index)
	go server.ListenAndServe()
	log.Println("Running on port 8080")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
	log.Println("Shutting down gracefully...")

	log.Println("Waiting for other workers to complete")
	wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		log.Printf("Error encountered while stopping HTTP listener: %s\n", err)
	}
}
