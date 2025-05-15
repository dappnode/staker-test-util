package domain

type TestConfig struct {
	DnpName string
	Mount   Mount
	Staker  Staker
}

type Mount struct {
	ExecutionContainerName string // The name of the container to mount the NFS volume to
	Path                   string
	Id                     string
}

type Staker struct {
	ValidatorIndexes []string
	Clients          Clients
}

type Clients struct {
	ExecutionDnpName  string
	ConsensusDnpName  string
	Web3SignerDnpName string
	MevBoostDnpName   string
	Relays            []string
	Network           string
	Urls              Urls
}

type Urls struct {
	ExecutionURL   string
	BrainURL       string
	BeaconchainURL string
	DappmanagerURL string
}
