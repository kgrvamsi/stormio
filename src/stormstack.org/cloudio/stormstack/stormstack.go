package stormstack

import (
	"stormstack.org/cloudio/persistence"
	"fmt"
	log "github.com/cihub/seelog"
	"launchpad.net/goose/client"
	goosehttp "launchpad.net/goose/http"
	"net/http"
	"net/url"
)

var (
	nclient = client.NewPublicClient("")
)

func BuildStormData(arq *persistence.AssetRequest) (stormdata string) {
	log.Debugf("stormtracker URL is %#v", arq)
	if arq.ControlProvider.StormtrackerURL == "" {
		log.Debugf("[areq %s] stormtracker URL is nil", arq.Id)
		return ""
	}
	u, err := url.Parse(arq.ControlProvider.StormtrackerURL)
	if err != nil {
		log.Debugf("[areq %s] Failed to parse the stormtracker URL %v", arq.Id, arq.ControlProvider.StormtrackerURL)
		return ""
	}
	stormdata = u.Scheme + "://" + arq.ControlTokenId + "@" + u.Host + "/" + u.Path
	log.Debugf("[areq %s] stormdata is %v", arq.Id, stormdata)
	return stormdata
}

func DeRegisterStormAgent(dr *persistence.AssetRequest) (err error) {
	var resp persistence.StormAgent
	u := fmt.Sprintf("%s/agents/%s", dr.ControlProvider.StormtrackerURL, dr.AgentId)
	requestData := &goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusNoContent}}
	err = nclient.SendRequest(client.DELETE, "", u, requestData)
	if err != nil {
		log.Debugf("[areq %s][res %s] Error in deregistering storm agent %v", dr.Id, dr.ResourceId, err)
		return err
	}
	// XXX - Add code to delete agent from domain
	return
}

type DomainAgent struct {
	AgentId string `json:"agentId"`
}

func DomainAddAgent(arq *persistence.AssetRequest) (err error) {
	var req DomainAgent
	req.AgentId = arq.AgentId
	if arq.ControlProvider.StormlightURL == "" {
		log.Debugf("[areq %s][res %s] Stormlight URL missing in the assetRequest", arq.Id, arq.ResourceId)
		return fmt.Errorf("DomainAddAgent: missing stormlight URL in the ControlProvider")
	}
	if arq.ControlProvider.DefaultDomainId == "" {
		log.Debugf("[areq %s][res %s] Default DomainID missing in the assetRequest", arq.Id, arq.ResourceId)
		return fmt.Errorf("DomainAddAgent: missing domainID in the ControlProvider")
	}

	log.Debugf("Registering storm agent with Stormlight %#v", req)
	var resp DomainAgent
	requestData := &goosehttp.RequestData{ReqValue: req, RespValue: &resp, ExpectedStatus: []int{http.StatusAccepted, http.StatusOK}}
	u := fmt.Sprintf("%s/domains/%s/agents", arq.ControlProvider.StormlightURL, arq.ControlProvider.DefaultDomainId)
	err = nclient.SendRequest(client.POST, "", u, requestData)
	if err != nil {
		log.Debugf("[areq %s][res %s] Error in registering storm agent %v", arq.Id, arq.ResourceId, err)
		return err
	}
	log.Debugf("[areq %s][res %s] Registered Agent with stormlight %v", arq.Id, arq.ResourceId, arq.AgentId)
	return nil
}

func RegisterStormAgent(arq *persistence.AssetRequest, entityId string) (err error) {
	if arq.AgentId != "" {
		log.Debugf("[areq %s][res %s] Agent ID already exists agent Id is %v", arq.Id, arq.ResourceId, arq.AgentId)
		return nil
	}
	arq.ServerId = entityId
	var req persistence.StormAgent
	req.SerialKey = entityId
	req.Stoken = arq.ControlTokenId
	req.StormBolt = arq.ControlProvider.Bolt

	log.Debugf("Registering storm agent with tracker %#v", req)
	var resp persistence.StormAgent
	requestData := &goosehttp.RequestData{ReqValue: req, RespValue: &resp, ExpectedStatus: []int{http.StatusAccepted, http.StatusOK}}
	u := fmt.Sprintf("%s/agents", arq.ControlProvider.StormtrackerURL)
	err = nclient.SendRequest(client.POST, "", u, requestData)
	if err != nil {
		log.Debugf("[areq %s][res %s] Error in registering storm agent %v", arq.Id, arq.ResourceId, err)
		return err
	}
	arq.AgentId = resp.Id
	arq.SerialKey = entityId
	log.Debugf("[areq %s][res %s] Registered Agent with stormtracker %v", arq.Id, arq.ResourceId, arq.AgentId)
	// Register the agent in Stormlight default domain
	err = DomainAddAgent(arq)
	if err != nil {
		log.Debugf("[areq %s][res %s] Error in Adding agent %v to default domain %v", arq.Id, arq.ResourceId, arq.AgentId, arq.ControlProvider.DefaultDomainId)
		return err
	}
	return nil
}
