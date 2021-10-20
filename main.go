package main

import (
	"context"
	"fmt"

	"github.com/nodebreaker0-0/umee-autod/client"
	"github.com/nodebreaker0-0/umee-autod/config"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg, err := config.Read(config.DefaultConfigPath)
	if err != nil {
		return err
	}
	client, err := client.NewClient(IBCchain.Rpc, IBCchain.Grpc)
	if err != nil {
		return err
	}
	defer client.Stop() // nolint: errcheck
	defer client.GRPC.Close()
	grpcclient := client.GRPC
	coins, err := grpcclient.GetAllBalances(ctx, IBCchain.DstAddress)
	if err != nil {
		return err
	}
	fmt.Println(IBCchain.ChainId, " | ", coins)
}
