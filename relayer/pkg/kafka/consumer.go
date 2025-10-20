package kafka

import (
	"fmt"
	"log"

	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaConsumer struct {
	ConfigMap *ckafka.ConfigMap
	Topics    []string
	consumer  *ckafka.Consumer
}

func NewKafkaConsumer(configMap *ckafka.ConfigMap, topics []string) *KafkaConsumer {
	return &KafkaConsumer{
		ConfigMap: configMap,
		Topics:    topics,
	}
}

func (c *KafkaConsumer) Consume(msgChan chan *ckafka.Message) error {
	var err error
	c.consumer, err = ckafka.NewConsumer(c.ConfigMap)
	if err != nil {
		return fmt.Errorf("error creating kafka consumer: %w", err)
	}
	err = c.consumer.SubscribeTopics(c.Topics, nil)
	if err != nil {
		return fmt.Errorf("error subscribing to topics: %w", err)
	}
	for {
		msg, err := c.consumer.ReadMessage(-1)
		if err == nil {
			log.Printf("Message on %s: %s", msg.TopicPartition, string(msg.Value))
			msgChan <- msg
		} else {
			log.Printf("Consumer error: %v\n", err)
		}
	}
}

func (c *KafkaConsumer) CommitMessage(msg *ckafka.Message) error {
	if c.consumer == nil {
		return fmt.Errorf("consumer not initialized")
	}
	_, err := c.consumer.CommitMessage(msg)
	return err
}
