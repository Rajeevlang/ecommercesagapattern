package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Rajeevlang/ecommercesagapattern/order/internals/domain"
	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	brokers []string
}

type OrderItem struct {
	ProductID  string `json:"product_id"`
	Quantity   int32  `json:"quantity"`
	PriceCents int64  `json:"price_cents"`
}

type OrderCompletedEvent struct {
	OrderID          string      `json:"order_id"`
	UserID           string      `json:"user_id"`
	UserEmail        string      `json:"user_email"`
	TotalAmountCents int64       `json:"total_amount_cents"`
	Items            []OrderItem `json:"items"`
}

type OrderFailedEvent struct {
	OrderID   string `json:"order_id"`
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	Reason    string `json:"reason"`
}

func NewKafkaPublisher(brokers []string) *KafkaPublisher {
	return &KafkaPublisher{brokers: brokers}
}

func (p *KafkaPublisher) PublishOrderCompleted(ctx context.Context, order *domain.Order, email string) error {
	var items []OrderItem
	for _, item := range order.Items {
		items = append(items, OrderItem{
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: item.PriceCents,
		})
	}

	event := OrderCompletedEvent{
		OrderID:          order.ID,
		UserID:           order.UserID,
		UserEmail:        email,
		TotalAmountCents: order.TotalAmountCents,
		Items:            items,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal order completed event: %w", err)
	}

	w := &kafka.Writer{
		Addr:     kafka.TCP(p.brokers...),
		Topic:    "events.order.completed",
		Balancer: &kafka.LeastBytes{},
	}
	defer w.Close()

	err = w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(order.ID),
		Value: payload,
	})
	if err != nil {
		log.Printf("Warning: Failed to publish events.order.completed to Kafka (Kafka might be offline): %v\n", err)
		// We print a warning and return nil or error depending on how strict we want to be.
		// For our distributed Saga, notification is a side effect, so we shouldn't fail order creation
		// if message broker is down during testing, but we still log it.
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	log.Printf("Published events.order.completed event to Kafka: %s\n", string(payload))
	return nil
}

func (p *KafkaPublisher) PublishOrderFailed(ctx context.Context, order *domain.Order, email string, reason string) error {
	event := OrderFailedEvent{
		OrderID:   order.ID,
		UserID:    order.UserID,
		UserEmail: email,
		Reason:    reason,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal order failed event: %w", err)
	}

	w := &kafka.Writer{
		Addr:     kafka.TCP(p.brokers...),
		Topic:    "events.order.failed",
		Balancer: &kafka.LeastBytes{},
	}
	defer w.Close()

	err = w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(order.ID),
		Value: payload,
	})
	if err != nil {
		log.Printf("Warning: Failed to publish events.order.failed to Kafka (Kafka might be offline): %v\n", err)
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	log.Printf("Published events.order.failed event to Kafka: %s\n", string(payload))
	return nil
}
