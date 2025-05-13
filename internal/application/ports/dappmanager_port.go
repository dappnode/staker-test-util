package ports

import "clients-test/internal/application/domain"

type DappmanagerPort interface {
	Ping() error
	PackageInstall(dnpName, versionOrIpfsHash string) error
	GetStakerConfig(network domain.Network) (domain.StakerConfigGetMinimal, error)
	SetStakerConfig(network domain.Network, executionDnpName, consensusDnpName, mevBoostDnpName, web3signerDnpName *string, relays []string) error
	RemoveNonCorePackages() error
}
