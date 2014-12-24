package persistence

import (
	"stormstack.org/stormio/util"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/nu7hatch/gouuid"
	"labix.org/v2/mgo/bson"
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

type MongoSuite struct {
	id string
}

var _ = Suite(&MongoSuite{})

func (ms *MongoSuite) SetUpSuite(c *C) {
	util.LoadProperties()
	defer log.Flush()
	seelogconfig := util.GetString("default", "log-conf")
	log.LoggerFromConfigAsFile(seelogconfig)
}

func (ms *MongoSuite) TestCreate(c *C) {
	conn, err := DefaultSession()
	if err != nil {
		c.Errorf(err.Error())
		return
	}

	defer conn.Close()
	assetRequest := &AssetRequest{}
	u4, _ := uuid.NewV4()
	ms.id = u4.String()
	u5, _ := uuid.NewV4()
	assetRequest.Id = u4.String()
	assetRequest.AssetProviderId = "self"
	assetRequest.ResourceId = u4.String()
	assetRequest.ReceivedOn = time.Now().String()
	assetRequest.Status = RequestRetry
	module := ModuleStatus{Name: "Sample", Id: u5.String(), Installed: true, Configured: false}
	assetRequest.Modules = append(assetRequest.Modules, module)
	conn.Create(assetRequest)
	fmt.Println("Test Create success")
}

func (ms *MongoSuite) TestFind(c *C) {
	fmt.Printf("Id %s\n", ms.id)
	conn, err := DefaultSession()
	if err != nil {
		c.Errorf(err.Error())
		return
	}
	defer conn.Close()
	asset, _ := conn.Find(bson.M{"resourceid": ms.id})

	fmt.Println(asset)
	fmt.Println("Test Find success")
}

func (ms *MongoSuite) TestList(c *C) {
	conn, err := DefaultSession()
	if err != nil {
		c.Errorf(err.Error())
		return
	}
	defer conn.Close()
	assets, _ := conn.FindAll(nil)
	for _, asset := range assets {
		fmt.Printf("%v", asset)
	}
	fmt.Println("Test List success")
}

func (ms *MongoSuite) TestMultiConditions(c *C) {
	conn, err := DefaultSession()
	if err != nil {
		c.Errorf(err.Error())
		return
	}
	defer conn.Close()
	assetReqs, err := conn.FindAll(bson.M{"status": bson.M{"$in": []string{RequestRetry, RequestRetryModuleInstall, RequestRetryModuleConfig, RequestMarkDeletion}}})
	fmt.Printf("Asset request size:%d", len(assetReqs))
	for _, asset := range assetReqs {
		fmt.Println(asset)
	}

	fmt.Println("Test Multi condition success")
}

func (ms *MongoSuite) TestUpdate(c *C) {
	fmt.Printf("Id %s\n", ms.id)
	conn, err := DefaultSession()
	if err != nil {
		c.Errorf(err.Error())
		return
	}
	defer conn.Close()
	asset, _ := conn.Find(bson.M{"_id": ms.id})
	asset.Modules[0].Name = "Renamed from Sample"
	conn.Update(asset)
	fmt.Println(asset)
	fmt.Println("Test Update success")
}
