package dbtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTx_SelectOne(t *testing.T) {
	tx := Tx(t)

	var n int
	err := tx.QueryRow(context.Background(), "SELECT 1").Scan(&n)
	require.NoError(t, err)
	require.Equal(t, 1, n)
}
