package domain

// Network represents the network type
type Network string

const (
	Mainnet Network = "mainnet"
	Hoodi   Network = "hoodi"
	Gnosis  Network = "gnosis"
	Lukso   Network = "lukso"
)

// stakerItemMinimal represents the minimal staker item info needed
type stakerItemMinimal struct {
	DnpName    string `json:"dnpName"`
	IsSelected bool   `json:"isSelected"`
}

// StakerConfigGetMinimal represents the minimal staker config info needed
type StakerConfigGetMinimal struct {
	ExecutionClients []stakerItemMinimal `json:"executionClients"`
	ConsensusClients []stakerItemMinimal `json:"consensusClients"`
	Web3Signer       stakerItemMinimal   `json:"web3Signer"`
	MevBoost         *stakerItemMinimal  `json:"mevBoost,omitempty"`
}

// StakerConfigSetRequest represents the request body for setting staker config
type StakerConfigSetRequest struct {
	Network           Network  `json:"network"`
	ExecutionDnpName  *string  `json:"executionDnpName"`
	ConsensusDnpName  *string  `json:"consensusDnpName"`
	MevBoostDnpName   *string  `json:"mevBoostDnpName"`
	Relays            []string `json:"relays"`
	Web3SignerDnpName *string  `json:"web3signerDnpName"`
}
