package rabbitmq

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/streadway/amqp"
)

type Option func(*RabbitMQ)

type RabbitMQ struct {
	conn              *amqp.Connection
	channel           *amqp.Channel
	queue             amqp.Queue
	comsumerChMapping map[string]chan amqp.Delivery
	quitConsumeCh     chan struct{}
	quitReconnectCh   chan struct{}

	url       string
	queueName string

	shouldRecovers  bool
	recoverDataPath string
}

func NewRabbitMQ(url, queueName string, options ...Option) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	queue, err := channel.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	r := &RabbitMQ{
		conn:    conn,
		channel: channel,
		queue:   queue,

		url:       url,
		queueName: queueName,

		comsumerChMapping: make(map[string]chan amqp.Delivery),
		quitConsumeCh:     make(chan struct{}),
		quitReconnectCh:   make(chan struct{}),
	}

	for _, option := range options {
		option(r)
	}

	return r, nil
}

func WithRecoverData(recoverPath string) Option {
	return func(r *RabbitMQ) {
		r.shouldRecovers = true
		r.recoverDataPath = filepath.Join(recoverPath, r.queueName)

		if _, err := os.Stat(filepath.Join(recoverPath, r.queueName)); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Join(recoverPath, r.queueName), 0755); err != nil {
				panic(err)
			}
		}

		if err := r.RecoverData(); err != nil {
			panic(err)
		}
	}
}

func WithReconnect() Option {
	return func(r *RabbitMQ) {
		go r.Reconnect()
	}
}

func (r *RabbitMQ) RecoverData() error {
	if _, err := os.Stat(r.recoverDataPath); os.IsNotExist(err) {
		return nil
	}

	if err := filepath.Walk(r.recoverDataPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			defer file.Close()
			data, err := io.ReadAll(file)
			if err != nil {
				return err
			}

			if err := r.PublishRaw(data); err != nil {
				return err
			}

			if err := os.Remove(path); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (r *RabbitMQ) PublishRaw(message []byte) error {
	err := r.channel.Publish(
		"",
		r.queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message,
		},
	)
	if err != nil {
		if err := r.saveMessage(message); err != nil {
			slog.Error("Failed to save message", slog.Any("channel", r.queueName))
		}
	}

	return err
}

func (r *RabbitMQ) Publish(payload interface{}) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	err = r.channel.Publish(
		"",
		r.queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message,
		},
	)
	if err != nil {
		if err := r.saveMessage(message); err != nil {
			slog.Error("Failed to save message", slog.Any("channel", r.queueName))
		}
	}

	return err
}

func (r *RabbitMQ) Consume(comsumerName string) (<-chan amqp.Delivery, error) {
	msgs, err := r.channel.Consume(
		r.queue.Name,
		comsumerName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	newMsgChan := make(chan amqp.Delivery)
	if _, ok := r.comsumerChMapping[comsumerName]; ok {
		return nil, fmt.Errorf("comsumer %s already exists", comsumerName)
	}

	r.comsumerChMapping[comsumerName] = newMsgChan
	go func() {
	Loop:
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					for {
						msgs, err = r.channel.Consume(
							r.queue.Name,
							comsumerName,
							true,
							false,
							false,
							false,
							nil,
						)
						if err != nil {
							time.Sleep(5 * time.Second)
							slog.Warn("fail consume message", slog.Any("channel", r.queueName))
							continue
						}

						continue Loop
					}
				}

				newMsgChan <- msg

			case <-r.quitConsumeCh:
				return
			}
		}
	}()

	return newMsgChan, nil
}

func (r *RabbitMQ) Cancel(comsumerName string) error {
	if ch, ok := r.comsumerChMapping[comsumerName]; ok {
		close(ch)
		delete(r.comsumerChMapping, comsumerName)
	}

	err := r.channel.Cancel(comsumerName, true)
	if err != nil {
		return err
	}

	return nil
}

func (r *RabbitMQ) CloseConnection() error {
	for _, ch := range r.comsumerChMapping {
		close(ch)
		delete(r.comsumerChMapping, r.queueName)
	}

	close(r.quitReconnectCh)
	if err := r.channel.Close(); err != nil {
		return err
	}

	if err := r.conn.Close(); err != nil {
		return err
	}

	return nil
}

func (r *RabbitMQ) StopConsume() {
	close(r.quitConsumeCh)
}

func (r *RabbitMQ) IsClosed() bool {
	if r.conn.IsClosed() {
		slog.Warn("disconnect rabbitmq", slog.Any("channel", r.queueName))
	}

	return r.conn.IsClosed()
}

func (r *RabbitMQ) Reconnect() {
	for {
		if r.IsClosed() {
			select {
			case <-r.quitReconnectCh:
				return
			default:
				time.Sleep(5 * time.Second)
				conn, err := amqp.Dial(r.url)
				if err != nil {
					slog.Debug("failed to connect to RabbitMQ", slog.Any("channel", r.queueName))
					continue
				}

				channel, err := conn.Channel()
				if err != nil {
					continue
				}

				queue, err := channel.QueueDeclare(
					r.queueName,
					false,
					false,
					false,
					false,
					nil,
				)
				if err != nil {
					continue
				}

				r.conn = conn
				r.channel = channel
				r.queue = queue
				if r.shouldRecovers {
					if err := r.RecoverData(); err != nil {
						return
					}
				}

				slog.Info("reconnect", slog.Any("channel", r.queueName))
			}
		} else {
			time.Sleep(5 * time.Second)
		}
	}
}

func (r *RabbitMQ) saveMessage(message []byte) error {
	recoverPath := filepath.Join(
		fmt.Sprintf("%s/%d.json", r.recoverDataPath, time.Now().UnixNano()),
	)
	file, err := os.OpenFile(recoverPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error(
			"failed to write recover data",
			slog.Any("channel", r.queueName),
			slog.Any("err", err),
		)
	}

	defer file.Close()

	if _, err := file.Write(message); err != nil {
		slog.Error(
			"Failed to write recover data",
			slog.Any("channel", r.queueName),
			slog.Any("err", err),
		)
	}

	return nil
}
