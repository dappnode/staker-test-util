package ports

type BeaconchainPort interface {
	GetIsSyncing() (bool, error)
	GetValidatorLiveness(indexes []string) (map[string]bool, error)
	GetValidatorsIndexes(pubkeys []string) (map[string]string, error)
}
