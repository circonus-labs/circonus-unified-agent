package kafka

import (
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
)

// ReadConfig for kafka clients meaning to read from Kafka.
type ReadConfig struct {
	Config
}

// SetConfig on the sarama.Config object from the ReadConfig struct.
func (k *ReadConfig) SetConfig(config *sarama.Config) error {
	config.Consumer.Return.Errors = true

	return k.Config.SetConfig(config)
}

// WriteConfig for kafka clients meaning to write to kafka
type WriteConfig struct {
	Config

	RequiredAcks     int  `toml:"required_acks"`
	MaxRetry         int  `toml:"max_retry"`
	MaxMessageBytes  int  `toml:"max_message_bytes"`
	IdempotentWrites bool `toml:"idempotent_writes"`
}

// SetConfig on the sarama.Config object from the WriteConfig struct.
func (k *WriteConfig) SetConfig(config *sarama.Config) error {
	config.Producer.Return.Successes = true
	config.Producer.Idempotent = k.IdempotentWrites
	config.Producer.Retry.Max = k.MaxRetry
	if k.MaxMessageBytes > 0 {
		config.Producer.MaxMessageBytes = k.MaxMessageBytes
	}
	config.Producer.RequiredAcks = sarama.RequiredAcks(k.RequiredAcks)
	return k.Config.SetConfig(config)
}

// Config common to all Kafka clients.
type Config struct {
	tls.ClientConfig
	Version  string `toml:"version"`
	ClientID string `toml:"client_id"`
	SASLAuth
	CompressionCodec int `toml:"compression_codec"`
}

// SetConfig on the sarama.Config object from the Config struct.
func (k *Config) SetConfig(config *sarama.Config) error {
	if k.Version != "" {
		version, err := sarama.ParseKafkaVersion(k.Version)
		if err != nil {
			return fmt.Errorf("parse kafka ver (%s): %w", k.Version, err)
		}

		config.Version = version
	}

	if k.ClientID != "" {
		config.ClientID = k.ClientID
	} else {
		config.ClientID = "Circonus"
	}

	config.Producer.Compression = sarama.CompressionCodec(k.CompressionCodec)

	tlsConfig, err := k.ClientConfig.TLSConfig()
	if err != nil {
		return fmt.Errorf("TLSConfig: %w", err)
	}

	if tlsConfig != nil {
		config.Net.TLS.Config = tlsConfig
		config.Net.TLS.Enable = true
	}

	if err := k.SetSASLConfig(config); err != nil {
		return err
	}

	return nil
}
