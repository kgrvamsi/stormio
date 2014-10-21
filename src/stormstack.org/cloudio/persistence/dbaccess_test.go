package persistence

import (
	"fmt"
	"github.com/nu7hatch/gouuid"
	"testing"
	"time"
)

func DisableTestCreate(t *testing.T) {
	assetDS := &AssetDS{}
	err := assetDS.InitDatabase("mysql", "cloudio", "cloudio", "password", "localhost", "3306")
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	assetDS.GetConnection()
	defer assetDS.Close()
	assetRequest := &AssetRequest{}
	u4, _ := uuid.NewV4()
	assetRequest.Id = u4.String()
	assetRequest.AssetProviderId = "self"
	assetRequest.ResourceId = "No callback"
	assetRequest.ReceivedOn = time.Now().String()
	assetDS.Create(assetDS.conn, assetRequest)
	fmt.Println("Test Create success")
}

func DisableTestUpate(t *testing.T) {
	assetDS := &AssetDS{}
	err := assetDS.InitDatabase("mysql", "cloudio", "cloudio", "password", "localhost", "3306")
	if err != nil {
		t.Errorf(err.Error())
	}
	assetDS.GetConnection()
	defer assetDS.Close()
	assetRequest := &AssetRequest{}
	u4, _ := uuid.NewV4()
	assetRequest.Id = u4.String()
	assetRequest.AssetProviderId = "self"
	assetRequest.ResourceId = "No callback"
	assetRequest.ReceivedOn = time.Now().String()
	assetDS.Create(assetDS.conn, assetRequest)
	assetRequest.ResourceId = "yes callback"
	assetDS.Update(assetDS.conn, assetRequest)
	fmt.Println("Test Update success")
}

func DisableTestRemove(t *testing.T) {
	assetDS := &AssetDS{}
	err := assetDS.InitDatabase("mysql", "cloudio", "cloudio", "password", "localhost", "3306")
	if err != nil {
		t.Errorf(err.Error())
	}
	assetDS.GetConnection()
	defer assetDS.Close()
	assetRequest := &AssetRequest{}
	u4, _ := uuid.NewV4()
	assetRequest.Id = u4.String()
	assetRequest.AssetProviderId = "self"
	assetRequest.ResourceId = "No callback"
	assetRequest.ReceivedOn = time.Now().String()
	assetDS.Create(assetDS.conn, assetRequest)
	assetDS.Remove(assetDS.conn, assetRequest.Id)
	fmt.Println("Test Delete success")
}

func DisableTestFind(t *testing.T) {
	assetDS := &AssetDS{}
	err := assetDS.InitDatabase("mysql", "cloudio", "cloudio", "password", "localhost", "3306")
	if err != nil {
		t.Errorf(err.Error())
	}
	assetDS.GetConnection()
	defer assetDS.Close()
	assetRequest := &AssetRequest{}
	u4, _ := uuid.NewV4()
	assetRequest.Id = u4.String()
	assetRequest.AssetProviderId = "self"
	assetRequest.ResourceId = "No callback"
	assetRequest.ReceivedOn = time.Now().String()
	assetDS.Create(assetDS.conn, assetRequest)
	newAs, err := assetDS.Find(assetDS.conn, assetRequest.Id)
	fmt.Println(newAs.ResourceId)
}
