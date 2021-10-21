package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/rs/zerolog/log"

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
	addr, privKey, err := wallet.IBCRecoverAccountFromMnemonic(mnemonic, "", "44'/118'/0'/0/0", "umee")
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

func MultiMsgWithdrawCommissionAndDelegate(valAddr string, delAddr string, coin sdktypes.Coin) (msgs []sdktypes.Msg, err error) {

	withdrawMsg := &distrtypes.MsgWithdrawValidatorCommission{
		ValidatorAddress: valAddr,
	}

	if err := withdrawMsg.ValidateBasic(); err != nil {
		return []sdktypes.Msg{}, err
	}
	delegateMsg := &stakingtypes.MsgDelegate{
		DelegatorAddress: delAddr,
		ValidatorAddress: valAddr,
		Amount:           coin,
	}
	if err := delegateMsg.ValidateBasic(); err != nil {
		return []sdktypes.Msg{}, err
	}

	return []sdktypes.Msg{withdrawMsg, delegateMsg}, nil
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
	//st, err := client.RPC.Status(ctx)
	//if err != nil {
	//	panic(fmt.Errorf("get status: %w", err))
	//}
	//startingHeight := st.SyncInfo.LatestBlockHeight + 2
	//log.Info().Msgf("current block height is %d, waiting for the next block to be committed", st.SyncInfo.LatestBlockHeight)

	//if err := rpcclient.WaitForHeight(client.RPC, startingHeight-1, nil); err != nil {
	//	panic(fmt.Errorf("wait for height: %w", err))
	//}

	//targetHeight := startingHeight

	for {
		//st, err := client.RPC.Status(ctx)
		//if err != nil {
		//	panic(fmt.Errorf("get status: %w", err))
		//}
		//if st.SyncInfo.LatestBlockHeight != targetHeight-1 {
		//	log.Warn().Int64("expected", targetHeight-1).Int64("got", st.SyncInfo.LatestBlockHeight).Msg("mismatching block height")
		//	targetHeight = st.SyncInfo.LatestBlockHeight + 1
		//}
		res, err := queryClient.ValidatorCommission(
			ctx,
			&types.QueryValidatorCommissionRequest{ValidatorAddress: cfg.Custom.ValidatorAddr},
		)
		if err != nil {
			println(err)
		}
		delamount := res.GetCommission().Commission.String()
		convertdelamount, err := sdktypes.ParseCoinNormalized(delamount)
		if err != nil {
			println(err)
		}

		accountcoins, err := client.GRPC.GetAllBalances(ctx, d.addr)
		if err != nil {
			println(err)
		}

		println("delamount: ", convertdelamount.String())
		var msgs []sdktypes.Msg
		totalamount := convertdelamount.Add(accountcoins[0])
		msgs, _ = MultiMsgWithdrawCommissionAndDelegate(cfg.Custom.ValidatorAddr, d.addr, totalamount.SubAmount(sdktypes.NewInt(600)))
		fmt.Println(msgs)

		accSeq := d.IncAccSeq()
		txByte, err := tx.Sign(ctx, accSeq, d.AccNum(), d.PrivKey(), msgs...)
		if err != nil {
			panic(fmt.Errorf("sign tx: %w", err))
		}
		resp, err := client.GRPC.BroadcastTx(ctx, txByte)
		if err != nil {
			panic(fmt.Errorf("broadcast tx: %w", err))
		}
		println(resp.TxResponse.RawLog)
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

		time.Sleep(60000 * time.Millisecond)

	}

}
