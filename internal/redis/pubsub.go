package redis

import (
	"context"
	"log"
)

const (
	// Pub/Sub channels
	ChannelUSDRate    = "broadcast:usd_rate"
	ChannelMarketData = "broadcast:market_data"
)

// Publish publishes a message to a channel
func (c *Client) Publish(ctx context.Context, channel string, message []byte) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to a channel and calls handler for each message
func (c *Client) Subscribe(ctx context.Context, channel string, handler func([]byte)) error {
	pubsub := c.rdb.Subscribe(ctx, channel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			if msg != nil {
				handler([]byte(msg.Payload))
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// SubscribeMultiple subscribes to multiple channels
func (c *Client) SubscribeMultiple(ctx context.Context, channels []string, handler func(channel string, data []byte)) error {
	pubsub := c.rdb.Subscribe(ctx, channels...)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			if msg != nil {
				handler(msg.Channel, []byte(msg.Payload))
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// PublishUSDRate publishes USD rate update
func (c *Client) PublishUSDRate(ctx context.Context, data []byte) error {
	log.Printf("Publishing USD rate to Redis channel: %s", ChannelUSDRate)
	return c.Publish(ctx, ChannelUSDRate, data)
}

// PublishMarketData publishes market data update
func (c *Client) PublishMarketData(ctx context.Context, data []byte) error {
	log.Printf("Publishing market data to Redis channel: %s", ChannelMarketData)
	return c.Publish(ctx, ChannelMarketData, data)
}
