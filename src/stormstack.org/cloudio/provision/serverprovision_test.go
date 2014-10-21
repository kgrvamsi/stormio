package provision

import (
	"stormstack.org/cloudio/persistence"
	"stormstack.org/cloudio/util"
	"fmt"
	log "github.com/cihub/seelog"
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type HPSuite struct{}

var _ = Suite(&HPSuite{})

func (s *HPSuite) SetUpSuite(c *C) {
	util.LoadProperties()
	defer log.Flush()
	seelogconfig := util.GetString("default", "log-conf")
	log.LoggerFromConfigAsFile(seelogconfig)
}

func _TestFloatingIP(t *testing.T) {
	assetProvider := &persistence.AssetProvider{Id: persistence.NewUUID(), Tenant: "vsc1.hp.intercloud.net", Username: "vscAdmin", Password: "Pr0t3ctth1s",
		EndPointURL: "https://region-b.geo-1.identity.hpcloudsvc.com:35357/v2.0", RegionName: "region-b.geo-1"}
	NewServiceProvision(assetProvider)
}

func (hp *HPSuite) _TestServerCreateHP(c *C) {
	assetProvider := &persistence.AssetProvider{Id: persistence.NewUUID(), Tenant: "vsc1.hp.intercloud.net", Username: "mmedina", Password: "pr0t3ctth1s",
		EndPointURL: "https://region-a.geo-1.identity.hpcloudsvc.com:35357/v2.0/", RegionName: "region-b.geo-1"}
	assetModel := &persistence.AssetModel{Name: "kvm", Flavor: "standard.xsmall", Image: "ClearPath_Cloudnode_3.8.0_20131208", Id: persistence.NewUUID()}
	assetReq := persistence.AssetRequest{HostName: "Testing Host", ResourceId: persistence.NewUUID(),
		ReceivedOn: time.Now().String(), Provider: assetProvider, Model: assetModel}
	nsp := NewServiceProvision(assetProvider)
	eId, fip, err := nsp.ProvisionInstance(&assetReq)
	if err != nil {
		c.Error(err)
	}
	assetReq.IpAddress = fip
	assetReq.ServerId = eId
	fmt.Printf("%s, %s", assetReq.IpAddress, assetReq.ServerId)
	log.Debugf("Deprovisiong instance...")
	nsp.DeprovisionInstance(&assetReq)
}

func _TestServerCreate(t *testing.T) {
	assetProvider := &persistence.AssetProvider{Id: persistence.NewUUID(), Tenant: "vhub1.dev.intercloud.net", Username: "vscAdmin", Password: "pr0t3ctth1s",
		EndPointURL: "http://vhub1.dev.intercloud.net:5000/v2.0", RegionName: "RegionOne"}
	assetModel := &persistence.AssetModel{Name: "kvm", Flavor: "c1.medium", Image: "cloudnode-x86-3.8.0-20130729-0", Id: persistence.NewUUID()}
	assetReq := persistence.AssetRequest{HostName: "Testing Host", ResourceId: persistence.NewUUID(),
		ReceivedOn: time.Now().String(), Provider: assetProvider, Model: assetModel}
	nsp := NewServiceProvision(assetProvider)
	sd, eId, err := nsp.ProvisionInstance(&assetReq)
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println("%s %v", eId, sd)
}
