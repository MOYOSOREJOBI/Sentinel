package kafka

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

func New(brokers []string, group string, topics ...string) (*kgo.Client, error) {
	opts := []kgo.Opt{kgo.SeedBrokers(brokers...)}
	if group != "" {
		opts = append(opts, kgo.ConsumerGroup(group), kgo.ConsumeTopics(topics...))
	}
	return kgo.NewClient(opts...)
}

func ProduceJSON(ctx context.Context, c *kgo.Client, topic, key string, v any, headers ...kgo.RecordHeader) error {
	b, _ := json.Marshal(v)
	rec := &kgo.Record{Topic: topic, Key: []byte(key), Value: b, Headers: headers}
	var err error
	for i := 0; i < 6; i++ {
		err = c.ProduceSync(ctx, rec).FirstErr()
		if err == nil {
			return nil
		}
		if !shouldRetryProduceError(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(i+1) * 300 * time.Millisecond):
		}
	}
	return err
}

func shouldRetryProduceError(err error) bool {
	if err == nil {
		return false
	}
	e := strings.ToUpper(err.Error())
	return strings.Contains(e, "UNKNOWN_TOPIC_OR_PARTITION") ||
		strings.Contains(e, "LEADER_NOT_AVAILABLE") ||
		strings.Contains(e, "NOT_LEADER_FOR_PARTITION")
}
