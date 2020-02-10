package main

import (
	"context"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/smira/go-statsd"
)

func main() {
	topic := "my-topic-" + strconv.Itoa(rand.Int())
	parallelism := 20
	go producer(topic, parallelism)
	go consumer(topic, parallelism)
	blockForever := make(chan bool)
	<-blockForever
}

func newStatsdClient(prefix string) *statsd.Client {
	return statsd.NewClient("127.0.0.1:8125",
		statsd.MaxPacketSize(1400),
		statsd.MetricPrefix(prefix))
}

func consumer(topic string, parallelism int) {
	stat := newStatsdClient("consumer.")
	for x := 0; x < parallelism; x++ {
		go func() {
			client, err := pulsar.NewClient(pulsar.ClientOptions{
				URL: "pulsar://localhost:6650",
			})
			if err != nil {
				log.Fatal("Error connecting the consumer")
			}

			defer client.Close()

			consumer, err := client.Subscribe(pulsar.ConsumerOptions{
				Topic:            topic,
				SubscriptionName: "my-sub",
				Type:             pulsar.Shared,
			})
			if err != nil {
				log.Fatal("Error subscribing")
			}

			defer consumer.Close()
			for {
				msg, recErr := consumer.Receive(context.Background())
				if recErr == nil {
					stat.PrecisionTiming("latency", time.Since(msg.PublishTime()))
					stat.Incr("messages_received", 1)
				} else {
					log.Fatal(recErr)
				}
				consumer.Ack(msg)
			}
		}()
	}
}

func producer(topic string, parallelism int) {
	stat := newStatsdClient("producer.")

	for x := 0; x < parallelism; x++ {
		go func() {
			client, _ := pulsar.NewClient(pulsar.ClientOptions{
				URL: "pulsar://127.0.0.1:6650",
			})

			defer client.Close()

			producer, _ := client.CreateProducer(pulsar.ProducerOptions{
				Topic: topic,
			})

			defer producer.Close()
			for {
				start := time.Now()
				producer.Send(context.Background(), &pulsar.ProducerMessage{
					Payload: []byte("hello"),
				})
				stat.PrecisionTiming("latency", time.Since(start))
				stat.Incr("messages_sent", 1)
			}
		}()
	}
}
