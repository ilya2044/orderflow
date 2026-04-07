package kafka

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type MessageHandler func(message *sarama.ConsumerMessage) error

type Consumer struct {
	group   sarama.ConsumerGroup
	topics  []string
	handler MessageHandler
	logger  *zap.Logger
}

type consumerGroupHandler struct {
	handler MessageHandler
	logger  *zap.Logger
}

func NewConsumer(brokers []string, groupID string, topics []string, handler MessageHandler, logger *zap.Logger) (*Consumer, error) {
	cfg := sarama.NewConfig()
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	cfg.Consumer.Group.Session.Timeout = 20 * time.Second
	cfg.Consumer.Group.Heartbeat.Interval = 6 * time.Second
	cfg.Net.DialTimeout = 10 * time.Second

	group, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		group:   group,
		topics:  topics,
		handler: handler,
		logger:  logger,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	h := &consumerGroupHandler{handler: c.handler, logger: c.logger}
	for {
		if err := c.group.Consume(ctx, c.topics, h); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.logger.Error("consumer group error", zap.Error(err))
			return err
		}
		if ctx.Err() != nil {
			return nil
		}
	}
}

func (c *Consumer) Close() error {
	return c.group.Close()
}

func (c *Consumer) Errors() <-chan error {
	return c.group.Errors()
}

func (h *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.handler(msg); err != nil {
			h.logger.Error("failed to handle message",
				zap.String("topic", msg.Topic),
				zap.Int32("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.Error(err),
			)
		}
		session.MarkMessage(msg, "")
	}
	return nil
}
