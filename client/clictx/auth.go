package clictx

import (
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/nodebreaker0-0/umee-autod/config"
)

// GetAccount checks account type and returns account interface.
func (c *Client) GetAccount(address string) (sdkclient.Account, error) {
	accAddr, err := config.AccAddressFromBech32(address, "umee")
	if err != nil {
		return nil, err
	}

	ar := authtypes.AccountRetriever{}

	acc, _, err := ar.GetAccountWithHeight(c.Context, accAddr)
	if err != nil {
		return nil, err
	}

	return acc, nil
}
