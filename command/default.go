package command

const (
	DefaultGenesisFileName = "genesis.json"
)

const (
	JSONOutputFlag       = "json"
	GRPCAddressFlag      = "grpc-address"
	JSONRPCFlag          = "jsonrpc"
	TransparentProxyFlag = "trans-proxy"
)

// GRPCAddressFlagLEGACY Legacy flag that needs to be present to preserve backwards
// compatibility with running clients
const (
	GRPCAddressFlagLEGACY = "grpc"
)
