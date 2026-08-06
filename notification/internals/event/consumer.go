package event

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/ports"
	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	brokers      []string
	templatesDir string
	svc          ports.NotificationService
	wg           sync.WaitGroup
	readers      []*kafka.Reader
	stopChan     chan struct{}
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

func NewKafkaConsumer(brokers []string, templatesDir string, svc ports.NotificationService) *KafkaConsumer {
	return &KafkaConsumer{
		brokers:      brokers,
		templatesDir: templatesDir,
		svc:          svc,
		stopChan:     make(chan struct{}),
	}
}

func (c *KafkaConsumer) Start(ctx context.Context) error {
	log.Printf("Initializing Kafka consumer on brokers: %v, templates dir: %s\n", c.brokers, c.templatesDir)

	// Create readers for topics
	completedReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.brokers,
		GroupID:  "notification-group",
		Topic:    "events.order.completed",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	c.readers = append(c.readers, completedReader)

	failedReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.brokers,
		GroupID:  "notification-group",
		Topic:    "events.order.failed",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	c.readers = append(c.readers, failedReader)

	c.wg.Add(2)
	go c.consumeCompletedLoop(completedReader)
	go c.consumeFailedLoop(failedReader)

	return nil
}

func (c *KafkaConsumer) Stop() {
	close(c.stopChan)
	for _, r := range c.readers {
		_ = r.Close()
	}
	c.wg.Wait()
	log.Println("Kafka consumers stopped.")
}

func (c *KafkaConsumer) consumeCompletedLoop(r *kafka.Reader) {
	defer c.wg.Done()
	log.Println("Listening for events.order.completed events...")

	for {
		select {
		case <-c.stopChan:
			return
		default:
			// Fetch the message without committing the offset automatically
			m, err := r.FetchMessage(context.Background())
			if err != nil {
				// if reader is closed, break
				if strings.Contains(err.Error(), "closed") {
					return
				}
				log.Printf("Error reading message from events.order.completed: %v\n", err)
				time.Sleep(1 * time.Second)
				continue
			}

			log.Printf("Received events.order.completed message: %s\n", string(m.Value))
			var event OrderCompletedEvent
			if err := json.Unmarshal(m.Value, &event); err != nil {
				log.Printf("Error deserializing OrderCompletedEvent: %v\n", err)
				// Commit bad messages so we don't block the queue forever (poison pill handling)
				_ = r.CommitMessages(context.Background(), m)
				continue
			}

			if event.UserEmail == "" {
				log.Printf("Warning: UserEmail is empty in OrderCompletedEvent for order %s. Skipping.\n", event.OrderID)
				_ = r.CommitMessages(context.Background(), m)
				continue
			}

			// Render templates
			body, err := c.renderTemplate("email_order_confirmation.html", event)
			if err != nil {
				log.Printf("Failed to render confirmation email template: %v\n", err)
				_ = r.CommitMessages(context.Background(), m)
				continue
			}

			idempotencyKey := fmt.Sprintf("order-completed-%s", event.OrderID)
			_, err = c.svc.SendEmail(context.Background(), event.UserID, event.UserEmail, "Order Confirmed! - #"+event.OrderID, body, "order_confirmation", idempotencyKey)
			if err != nil {
				log.Printf("Failed to send order confirmation email: %v\n", err)
				// DO NOT commit if the database itself failed (network issues). Retry backoff.
				if !strings.Contains(err.Error(), "failed to check idempotency") && !strings.Contains(err.Error(), "failed to log notification") {
					_ = r.CommitMessages(context.Background(), m)
				} else {
					// Back off for a bit before fetching the same message again
					time.Sleep(2 * time.Second)
				}
			} else {
				log.Printf("Successfully processed confirmation email for order: %s\n", event.OrderID)
				// Commit offset on successful save/send
				_ = r.CommitMessages(context.Background(), m)
			}
		}
	}
}

func (c *KafkaConsumer) consumeFailedLoop(r *kafka.Reader) {
	defer c.wg.Done()
	log.Println("Listening for events.order.failed events...")

	for {
		select {
		case <-c.stopChan:
			return
		default:
			m, err := r.FetchMessage(context.Background())
			if err != nil {
				if strings.Contains(err.Error(), "closed") {
					return
				}
				log.Printf("Error reading message from events.order.failed: %v\n", err)
				time.Sleep(1 * time.Second)
				continue
			}

			log.Printf("Received events.order.failed message: %s\n", string(m.Value))
			var event OrderFailedEvent
			if err := json.Unmarshal(m.Value, &event); err != nil {
				log.Printf("Error deserializing OrderFailedEvent: %v\n", err)
				_ = r.CommitMessages(context.Background(), m)
				continue
			}

			if event.UserEmail == "" {
				log.Printf("Warning: UserEmail is empty in OrderFailedEvent for order %s. Skipping.\n", event.OrderID)
				_ = r.CommitMessages(context.Background(), m)
				continue
			}

			body, err := c.renderTemplate("email_order_failed.html", event)
			if err != nil {
				log.Printf("Failed to render failed email template: %v\n", err)
				_ = r.CommitMessages(context.Background(), m)
				continue
			}

			idempotencyKey := fmt.Sprintf("order-failed-%s", event.OrderID)
			_, err = c.svc.SendEmail(context.Background(), event.UserID, event.UserEmail, "Problem processing your order - #"+event.OrderID, body, "order_failed", idempotencyKey)
			if err != nil {
				log.Printf("Failed to send order failed email: %v\n", err)
				if !strings.Contains(err.Error(), "failed to check idempotency") && !strings.Contains(err.Error(), "failed to log notification") {
					_ = r.CommitMessages(context.Background(), m)
				} else {
					time.Sleep(2 * time.Second)
				}
			} else {
				log.Printf("Successfully processed failure email for order: %s\n", event.OrderID)
				_ = r.CommitMessages(context.Background(), m)
			}
		}
	}
}

func (c *KafkaConsumer) renderTemplate(filename string, data interface{}) (string, error) {
	path := filepath.Join(c.templatesDir, filename)
	tmpl, err := template.New(filename).Funcs(template.FuncMap{
		"formatPrice": func(cents int64) string {
			return fmt.Sprintf("%.2f", float64(cents)/100.0)
		},
	}).ParseFiles(path)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
