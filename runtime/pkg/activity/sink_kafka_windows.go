//go:build windows

package activity

import (
	"fmt"

	"go.uber.org/zap"
)

// NewKafkaSink is not supported on Windows (librdkafka linking issue).
func NewKafkaSink(brokers, topic string, logger *zap.Logger) (Sink, error) {
	return nil, fmt.Errorf("kafka sink is not supported on Windows")
}
