package provision

import (
	"stormstack.org/stormio/persistence"
	log "github.com/cihub/seelog"
	"launchpad.net/goose/errors"
	"launchpad.net/goose/neutron"
)

type Network struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Cidr      string `json:"cidr"`
	Vlan      int32  `json:"vlan"`
	QuantumId string `json:"quantumId"`
}

type Intranet struct {
	Network string `json:"network"`
	Netmask string `json:"netmask"`
	Gateway string `json:"gateway"`
}

type ProviderNetwork struct {
	Id            string            `json:"id"`
	Network       Network           `json:"network"`
	Intranets     []Intranet        `json:"intranets"`
	AssetProvider persistence.AssetProvider `json:"assetProvider"`
}

const (
	NETWORK_TYPE      = "vlan"
	NETWORK_NAME      = "physnet1"
	IP_VERSION_4      = 4
	IP_VERSION_6      = 6
	DEFAULT_ROUTER_ID = ""
)

func (svc *ServiceProvision) CreateProviderNetwork(pn *ProviderNetwork) error {
	network := &neutron.Network{Name: "Network-" + pn.Id, TenantId: pn.AssetProvider.Tenant, PhysicalNetwork: NETWORK_NAME,
		NetworkType: NETWORK_TYPE, SegmentationId: pn.Network.Vlan}
	log.Debugf("Creating network")
	network, err := svc.neutron.CreateNetwork(network)
	if err != nil {
		log.Errorf("Faile to create network error:%v", err)
		return errors.Newf(err, "failed to create network")
	}
	ipRange := make([]neutron.IPRange, 1, 1)
	ipRange[0] = neutron.IPRange{Start: pn.Network.Start, End: pn.Network.End}
	subnet := &neutron.Subnet{IpVersion: IP_VERSION_4, TenantId: pn.AssetProvider.Tenant, NetworkId: network.Id,
		EnableDhcp: true, Cidr: pn.Network.Cidr, AllocationPools: ipRange}
	subnet, err = svc.neutron.CreateSubnet(subnet)
	if err != nil {
		log.Errorf("Faile to create subnet error:%v", err)
		return errors.Newf(err, "failed to create subnet")
	}
	_, err = svc.neutron.AddRouterToSubnet(DEFAULT_ROUTER_ID, subnet.Id)
	if err != nil {
		return errors.Newf(err, "Failed to attach Router to Subnet")
	}
	log.Debug("Provider Network created successfully")
	return nil
}
