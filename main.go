package main

import (
	"context"

	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/nodebreaker0-0/umee-autod/client"
	"github.com/nodebreaker0-0/umee-autod/config"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Read(config.DefaultConfigPath)
	if err != nil {
		println(err)

	}
	client, err := client.NewClient(cfg.RPC.Address, cfg.GRPC.Address)
	if err != nil {
		println(err)
	}
	defer client.Stop() // nolint: errcheck

	queryClient := types.NewQueryClient(client.GRPC)

	res, err := queryClient.ValidatorCommission(
		ctx,
		&types.QueryValidatorCommissionRequest{ValidatorAddress: cfg.Custom.ValidatorAddr},
	)
	if err != nil {
		println(err)
	}
	a := res.GetCommission().Commission.String()
	println(a)

}
