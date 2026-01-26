package domain

import (
	"fmt"
	"strings"
)

// Utility to get the servicename from a execution client (this utility assumes the service name is the first part of the dnp name)
// e.g hoodi-nethermind.dnp.dappnode.eth -> nethermind
// e.g nethermind-hoodi.dnp.dappnode.eth -> nethermind
// e.g nethermind-hoodi.public.dappnode.eth -> nethermind
// e.g geth.dnp.dappnode.eth -> geth
// e.g reth-gnosis.dnp.dappnode.eth -> reth
// e.g gnosis-reth.dnp.dappnode.eth -> reth
func serviceNameFromExecutionClient(dnpName, network string) string {
	trimmed := strings.TrimSuffix(dnpName, ".dnp.dappnode.eth")
	trimmed = strings.TrimSuffix(trimmed, ".public.dappnode.eth")
	parts := strings.Split(trimmed, "-")

	// If there is only one part, return it
	if len(parts) == 1 {
		return parts[0]
	}
	// If there are multiple parts, find and remove the network part
	for i, part := range parts {
		if part == network {
			parts = append(parts[:i], parts[i+1:]...)
			break
		}
	}
	// Return the remaining parts joined by "-"
	return strings.Join(parts, "-")
}

// Utility to get the container name from service and dnpName. append dnp or public suffix depending on original dnpName
func containerName(serviceName, dnpName string) string {
	return fmt.Sprintf("DAppNodePackage-%s.%s", serviceName, dnpName)
}

// Utility to get the docker volume name from dnpName and compose volume name
// i.e hoodi-nethermind.dnp.dappnode.eth -> hoodi-netherminddnpdappnodeeth_<composeVolumeName>
func composeVolumeName(dnpName, composeVolumeName string) string {
	return fmt.Sprintf("%s_%s", strings.ReplaceAll(dnpName, ".", ""), composeVolumeName)
}
