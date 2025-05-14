package domain

type StakerClients struct {
	ExecutionDnpName  string
	ConsensusDnpName  string
	Web3SignerDnpName string
	MevBoostDnpName   string
	Relays            []string
	Network           string
}

type TestConfig struct {
	ValidatorIndexes       []string
	DnpName                string
	MountPath              string
	MountId                string
	ExecutionContainerName string
	StakerClients          StakerClients
}
