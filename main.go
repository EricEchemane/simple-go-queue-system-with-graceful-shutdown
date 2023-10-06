package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

func main() {

	server := &http.Server{Addr: ":8080"}

	topic := "quickstart-events"
	kafkaWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   topic,
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		random_message := rand.Int()
		fmt.Fprint(w, random_message)

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			log.Printf("processing: %d\n", random_message)
			err := kafkaWriter.WriteMessages(ctx, kafka.Message{Value: []byte(fmt.Sprint(random_message))})
			if err != nil {
				log.Println("failed to write message to Kafka:", err)
			}
		}()
	})

	go server.ListenAndServe()
	log.Println("Running on port 8080")

	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   topic,
	})

	go func() {
		for {
			message, err := kafkaReader.ReadMessage(context.Background())
			if err != nil {
				log.Println("failed to read message from Kafka:", err)
				break
			}
			fmt.Printf("Received message: %+v\n", message)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
	log.Println("Shutting down gracefully...")

	kafkaWriter.Close()
	kafkaReader.Close()
}
