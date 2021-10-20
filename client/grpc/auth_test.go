package grpc_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBaseAccountInfo(t *testing.T) {
	address := "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v"
	resp, err := c.GetBaseAccountInfo(context.Background(), address)
	require.NoError(t, err)

	t.Log(resp.GetAccountNumber())
	t.Log(resp.GetSequence())
}
