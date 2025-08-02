//go:build unit
package config_test

import (
	"testing"

	"github.com/muratdemir0/gopulse-messages/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	t.Run("given a valid config file, it should load the config", func(t *testing.T) {
		cfg, err := config.Load("../../testdata/dev.yaml")
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		assert.Equal(t, "test", cfg.App.Name)
		assert.Equal(t, 8080, cfg.App.Port)
		assert.Equal(t, "http://localhost:8080", cfg.Webhook.Host)
		assert.Equal(t, "/webhook/gopulse-messages", cfg.Webhook.Path)
	})

	t.Run("given a non-existent config file, it should return an error", func(t *testing.T) {
		cfg, err := config.Load("../../testdata/nonexistent.yaml")
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})
}
