package db

import (
	"context"
	"testing"

	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/stretchr/testify/require"
)

func TestNewPool_Ping(t *testing.T) {
	config, err := util.LoadConfig("../../../config.yaml")
	require.NoError(t, err)

	pool, err := NewPool(context.Background(), config.Database.DSN)
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, pool.Ping(context.Background()))
}
