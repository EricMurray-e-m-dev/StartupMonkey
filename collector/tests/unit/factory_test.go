package unit

import (
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
	"github.com/stretchr/testify/assert"
)

func TestNewAdapter_Postgres(t *testing.T) {
	a, err := adapter.NewAdapter("postgres", "postgres://test@localhost/test", "postgres")

	assert.NoError(t, err)
	assert.NotNil(t, a)
}

func TestNewAdapter_UnsupportedType(t *testing.T) {
	a, err := adapter.NewAdapter("unsupported-db", "conn-string", "id")

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "unsupported")
}
