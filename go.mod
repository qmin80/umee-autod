module github.com/nodebreaker0-0/umee-autod

go 1.16

require (
	github.com/cosmos/cosmos-sdk v0.44.2
	github.com/pelletier/go-toml v1.9.3
	github.com/rs/zerolog v1.23.0
	github.com/tendermint/liquidity v1.4.0 // indirect
	github.com/tendermint/tendermint v0.34.13
	google.golang.org/grpc v1.40.0
)

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
