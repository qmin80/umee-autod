package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/rs/zerolog/log"
	rpcclient "github.com/tendermint/tendermint/rpc/client"

	"github.com/nodebreaker0-0/umee-autod/client"
	"github.com/nodebreaker0-0/umee-autod/config"
	"github.com/nodebreaker0-0/umee-autod/tx"
	"github.com/nodebreaker0-0/umee-autod/wallet"
)

type AccountDispenser struct {
	c         *client.Client
	mnemonics []string
	i         int
	addr      string
	privKey   *secp256k1.PrivKey
	accSeq    uint64
	accNum    uint64
}

func NewAccountDispenser(c *client.Client, mnemonics []string) *AccountDispenser {
	return &AccountDispenser{
		c:         c,
		mnemonics: mnemonics,
	}
}

func (d *AccountDispenser) Next() error {
	mnemonic := d.mnemonics[d.i]
	addr, privKey, err := wallet.RecoverAccountFromMnemonic(mnemonic, "", "umee")
	if err != nil {
		return err
	}
	d.addr = addr
	d.privKey = privKey
	fmt.Println(addr)
	acc, err := d.c.GRPC.GetBaseAccountInfo(context.Background(), addr)
	if err != nil {
		return fmt.Errorf("get base account info: %w", err)
	}
	d.accSeq = acc.GetSequence()
	d.accNum = acc.GetAccountNumber()
	d.i++
	if d.i >= len(d.mnemonics) {
		d.i = 0
	}
	return nil
}

func (d *AccountDispenser) Addr() string {
	return d.addr
}

func (d *AccountDispenser) PrivKey() *secp256k1.PrivKey {
	return d.privKey
}

func (d *AccountDispenser) AccSeq() uint64 {
	return d.accSeq
}

func (d *AccountDispenser) AccNum() uint64 {
	return d.accNum
}

func (d *AccountDispenser) IncAccSeq() uint64 {
	r := d.accSeq
	d.accSeq++
	return r
}

func (d *AccountDispenser) DecAccSeq() {
	d.accSeq--
}

func MultiMsgWithdrawCommissionAndDelegate(valAddr sdktypes.ValAddress, delAddr sdktypes.AccAddress, coin sdktypes.Coin) (msgs []sdktypes.Msg, err error) {
	withdrawMsg := distrtypes.NewMsgWithdrawValidatorCommission(valAddr)
	if err := withdrawMsg.ValidateBasic(); err != nil {
		return []sdktypes.Msg{}, err
	}

	delegateMsg := stakingtypes.NewMsgDelegate(delAddr, valAddr, coin)
	if err := delegateMsg.ValidateBasic(); err != nil {
		return []sdktypes.Msg{}, err
	}

	return []sdktypes.Msg{withdrawMsg, delegateMsg}, nil
}

// AccAddressFromBech32 creates an AccAddress from a Bech32 string.
func AccAddressFromBech32(address, prefix string) (addr sdktypes.AccAddress, err error) {
	if len(strings.TrimSpace(address)) == 0 {
		return sdktypes.AccAddress{}, errors.New("empty address string is not allowed")
	}

	bz, err := sdktypes.GetFromBech32(address, prefix)
	if err != nil {
		return nil, err
	}

	err = sdktypes.VerifyAddressFormat(bz)
	if err != nil {
		return nil, err
	}

	return sdktypes.AccAddress(bz), nil
}
func ValAddressFromBech32(address string, prefix string) (addr sdktypes.ValAddress, err error) {
	if len(strings.TrimSpace(address)) == 0 {
		return sdktypes.ValAddress{}, errors.New("empty address string is not allowed")
	}

	bz, err := sdktypes.GetFromBech32(address, prefix)
	if err != nil {
		return nil, err
	}

	err = sdktypes.VerifyAddressFormat(bz)
	if err != nil {
		return nil, err
	}

	return sdktypes.ValAddress(bz), nil
}
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

	chainID, err := client.RPC.GetNetworkChainID(ctx)
	if err != nil {
		panic(err)
	}

	gasLimit := uint64(cfg.Custom.GasLimit)
	fees := sdktypes.NewCoins(sdktypes.NewCoin(cfg.Custom.FeeDenom, sdktypes.NewInt(cfg.Custom.FeeAmount)))
	memo := cfg.Custom.Memo
	tx := tx.NewTransaction(client, chainID, gasLimit, fees, memo)

	d := NewAccountDispenser(client, cfg.Custom.Mnemonics)
	if err := d.Next(); err != nil {
		panic(fmt.Errorf("get next account: %w", err))
	}

	blockTimes := make(map[int64]time.Time)
	st, err := client.RPC.Status(ctx)
	if err != nil {
		panic(fmt.Errorf("get status: %w", err))
	}
	startingHeight := st.SyncInfo.LatestBlockHeight + 2
	log.Info().Msgf("current block height is %d, waiting for the next block to be committed", st.SyncInfo.LatestBlockHeight)

	if err := rpcclient.WaitForHeight(client.RPC, startingHeight-1, nil); err != nil {
		panic(fmt.Errorf("wait for height: %w", err))
	}

	targetHeight := startingHeight

	//started := time.Now()
	sent := 0

	for i := 0; i < 100; i++ {
		st, err := client.RPC.Status(ctx)
		if err != nil {
			panic(fmt.Errorf("get status: %w", err))
		}
		if st.SyncInfo.LatestBlockHeight != targetHeight-1 {
			log.Warn().Int64("expected", targetHeight-1).Int64("got", st.SyncInfo.LatestBlockHeight).Msg("mismatching block height")
			targetHeight = st.SyncInfo.LatestBlockHeight + 1
		}
		delegator, err := AccAddressFromBech32(d.addr, "umee")
		println("deladdress", delegator.String())
		if err != nil {
			panic(err)
		}
		validator, err := ValAddressFromBech32(d.addr, "umee")
		println("valaddress", validator.String())
		if err != nil {
			panic(err)
		}
		// TODO: fix staking amount using queried commission
		staking := sdktypes.NewCoin("uumee", sdktypes.OneInt())
		msgs, err := MultiMsgWithdrawCommissionAndDelegate(validator, delegator, staking)
		fmt.Println(msgs, blockTimes)

		accSeq := d.IncAccSeq()
		txByte, err := tx.Sign(ctx, accSeq, d.AccNum(), d.PrivKey(), msgs...)
		if err != nil {
			panic(fmt.Errorf("sign tx: %w", err))
		}
		resp, err := client.GRPC.BroadcastTx(ctx, txByte)
		if err != nil {
			panic(fmt.Errorf("broadcast tx: %w", err))
		}
		if resp.TxResponse.Code != 0 {
			if resp.TxResponse.Code == 0x14 {
				log.Warn().Msg("mempool is full, stopping")
				d.DecAccSeq()
				continue
			}
			if resp.TxResponse.Code == 0x13 || resp.TxResponse.Code == 0x20 {
				if err := d.Next(); err != nil {
					panic(fmt.Errorf("get next account: %w", err))
				}
				log.Warn().Str("addr", d.Addr()).Uint64("seq", d.AccSeq()).Msgf("received %#v, using next account", resp.TxResponse)
				time.Sleep(500 * time.Millisecond)
				break
			} else {
				panic(fmt.Sprintf("%#v\n", resp.TxResponse))
			}
		}
		sent++
	}

}
