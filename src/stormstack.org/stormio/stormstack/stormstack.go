package stormstack

import (
	"fmt"
	log "github.com/cihub/seelog"
	"launchpad.net/goose/client"
	goosehttp "launchpad.net/goose/http"
	"net/http"
	"net/url"
	"stormstack.org/stormio/persistence"
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
	if dr.AgentId != "" {
		var resp persistence.StormAgent
		headers := make(http.Header)
		headers.Add("V-Auth-Token", dr.ControlTokenId)
		u := fmt.Sprintf("%s/agents/%s", dr.ControlProvider.StormtrackerURL, dr.AgentId)
		requestData := &goosehttp.RequestData{ReqHeaders: headers, RespValue: &resp, ExpectedStatus: []int{http.StatusNoContent}}
		err = nclient.SendRequest(client.DELETE, "", u, requestData)
		if err != nil {
			log.Errorf("[areq %s][res %s] Error in deregistering storm agent %v", dr.Id, dr.ResourceId, err)
		}
	}
	//dr.AgentId = ""
	return
}

type DomainAgent struct {
	AgentId string `json:"agentId"`
}

func DomainDeleteAgent(arq *persistence.AssetRequest) (err error) {
	if arq.AgentId == "" {
		log.Debugf("[areq %s][res %s] AgentID not present in assetRequest. Hence skipping deleting the agent in StormLight", arq.Id, arq.ResourceId)
		return nil
	}
	if arq.ControlProvider.StormlightURL == "" {
		log.Debugf("[areq %s][res %s] Stormlight URL missing in the assetRequest", arq.Id, arq.ResourceId)
		return fmt.Errorf("DomainDeleteAgent: missing stormlight URL in the ControlProvider")
	}
	if arq.ControlProvider.DefaultDomainId == "" {
		log.Debugf("[areq %s][res %s] Default DomainID missing in the assetRequest", arq.Id, arq.ResourceId)
		return fmt.Errorf("DomainDeleteAgent: missing domainID in the ControlProvider")
	}
	var resp DomainAgent
	headers := make(http.Header)
	headers.Add("V-Auth-Token", arq.ControlTokenId)
	requestData := &goosehttp.RequestData{ReqHeaders: headers, RespValue: &resp, ExpectedStatus: []int{http.StatusAccepted, http.StatusNoContent}}
	u := fmt.Sprintf("%s/domains/%s/agents/%s", arq.ControlProvider.StormlightURL, arq.ControlProvider.DefaultDomainId, arq.AgentId)
	err = nclient.SendRequest(client.DELETE, "", u, requestData)
	if err != nil {
		log.Errorf("[areq %s][res %s] Error in deleting storm agent %v with StormLight", arq.Id, arq.ResourceId, err)
		return err
	}
	log.Debugf("[areq %s][res %s] Deleted Agent ID %s with stormlight", arq.Id, arq.ResourceId, arq.AgentId)
	return nil
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
	headers := make(http.Header)
	headers.Add("V-Auth-Token", arq.ControlTokenId)
	requestData := &goosehttp.RequestData{ReqHeaders: headers, ReqValue: req, RespValue: &resp, ExpectedStatus: []int{http.StatusAccepted, http.StatusOK}}
	u := fmt.Sprintf("%s/domains/%s/agents", arq.ControlProvider.StormlightURL, arq.ControlProvider.DefaultDomainId)
	err = nclient.SendRequest(client.POST, "", u, requestData)
	if err != nil {
		log.Debugf("[areq %s][res %s] Error in registering storm agent %v", arq.Id, arq.ResourceId, err)
		return err
	}
	log.Debugf("[areq %s][res %s] Registered Agent with stormlight %v. Response is %#v", arq.Id, arq.ResourceId, arq.AgentId, resp)
	return nil
}

func RegisterStormAgent(arq *persistence.AssetRequest, entityId string) (err error) {
	arq.ServerId = entityId
	var req persistence.StormAgent
	req.SerialKey = entityId
	req.Stoken = arq.ControlTokenId
	req.StormBolt = arq.ControlProvider.Bolt
	req.Id = arq.AgentId

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
		go DeRegisterStormAgent(arq)
		return err
	}
	return nil
}
