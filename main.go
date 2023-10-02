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

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

var wg sync.WaitGroup
var redisClient *redis.Client

func worker(item int) {
	defer wg.Done()
	delay := rand.Intn(7)
	log.Printf("ongoing: %d received processing in %ds\n", item, delay)
	time.Sleep(time.Second * time.Duration(delay))

	key := fmt.Sprintf("queue:%d", item)
	err := redisClient.Set(context.Background(), key, item, 0).Err()
	if err != nil {
		log.Printf("Failed to push item to Redis queue: %v", err)
	}
}

func index(w http.ResponseWriter, req *http.Request) {
	wg.Add(1)
	item := rand.Int()
	go worker(item)
	time.Sleep(time.Second * 10) // simulate some process
	fmt.Fprintf(w, "%d", item)
}

func main() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	pong, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Printf("Connected to Redis: %s", pong)

	go func() {
		for range time.Tick(time.Millisecond) {
			// Use GET to retrieve and process items from the Redis queue
			key, err := redisClient.Keys(context.Background(), "queue:*").Result()
			if err != nil {
				log.Printf("Failed to fetch keys from Redis: %v", err)
				continue
			}

			for _, k := range key {
				item, err := redisClient.Get(context.Background(), k).Result()
				if err != nil {
					log.Printf("Failed to get item from Redis: %v", err)
					continue
				}

				log.Printf("done: %s processed\n", item)

				// delete after processing
				redisClient.Del(context.Background(), k)
			}
		}
	}()

	limiter := rate.NewLimiter(rate.Limit(1000), 5)
	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		index(w, r)
	})
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

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error encountered while stopping HTTP listener: %s\n", err)
	}
}
