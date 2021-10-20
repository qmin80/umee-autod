package codec

import (
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/cosmos-sdk/simapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
)

// Codec is the application-wide Amino codec and is initialized upon package loading.
var (
	AppCodec       codec.Codec
	AminoCodec     *codec.LegacyAmino
	EncodingConfig simappparams.EncodingConfig
)

// SetCodec sets encoding config.
func SetCodec() {
	EncodingConfig = simapp.MakeTestEncodingConfig()
	AppCodec = EncodingConfig.Marshaler
	AminoCodec = EncodingConfig.Amino
}
