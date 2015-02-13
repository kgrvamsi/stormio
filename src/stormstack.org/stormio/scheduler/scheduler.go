package scheduler

import (
	"fmt"
	log "github.com/cihub/seelog"
	"labix.org/v2/mgo/bson"
	"launchpad.net/goose/client"
	"launchpad.net/goose/errors"
	goosehttp "launchpad.net/goose/http"
	"launchpad.net/goose/identity"
	"net/http"
	pkgurl "net/url"
	"stormstack.org/stormio/cache"
	"stormstack.org/stormio/persistence"
	"stormstack.org/stormio/provision"
	"stormstack.org/stormio/stormstack"
	"stormstack.org/stormio/util"
	"time"
)

type Provisioner struct {
	DelNotification chan *persistence.AssetRequest
	CRequest        chan *persistence.AssetRequest
	CRemediation    chan *persistence.AssetRequest
	CNotification   chan string
	Client          client.Client
}

const (
	MaxBuffer = 50
)

/*
 *
 * Provisioner keep holds of the channel using for communicating the progress
 * in various steps
 *
 */
func NewProvisioner() (pro *Provisioner) {
	crequest := make(chan *persistence.AssetRequest, MaxBuffer)
	cremediation := make(chan *persistence.AssetRequest, MaxBuffer)
	dnotifies := make(chan *persistence.AssetRequest, MaxBuffer)
	cnotifies := make(chan string, MaxBuffer)
	prov := &Provisioner{dnotifies, crequest, cremediation, cnotifies, client.NewPublicClient("")}
	prov.StartProvisioner()
	pro = prov
	return
}

func (prov *Provisioner) StartProvisioner() {
	go func() {
		RateLimit := time.Duration(util.GetInt("server", "rate-limit"))
		throttle := time.Tick(time.Minute / RateLimit)
		for arReq := range prov.CRequest {
			log.Debugf("[areq %s] Server creation request received from Vertex", arReq.Id)
			go func(assetReq *persistence.AssetRequest) {
				conn, err := persistence.DefaultSession()
				defer conn.Close()
				if err != nil {
					log.Errorf("[areq %s] Error in getting persistent session :%v", assetReq.Id, err)
					return
				}
				if assetReq.Provider == (persistence.AssetProvider{}) {
					assetReq.SerialKey = persistence.NewUUID()
					time.Sleep(time.Duration(4) * time.Second)
					err = stormstack.RegisterStormAgent(assetReq, assetReq.SerialKey)
					if err != nil {
						log.Errorf("[areq %s] Failed to register Agent. Error is %v", assetReq.Id, err)
						assetReq.Status = persistence.RequestRetry
						conn.Update(assetReq)
						return
					}
					prov.updateAndNotify(conn, assetReq)
				} else {

					if err := prov.createServer(conn, assetReq); err == nil {
						////shouldn't notify Vertex if fip is nil
						if assetReq.IpAddress == "" {
							log.Debugf("[areq %s] Floating IP is not allocated", assetReq.Id)
							return
						}
						//// code ends
						log.Debugf("[areq %s] Notifying VertexPlatform to create an Asset, ServerId:%s", assetReq.Id, assetReq.ServerId)
						prov.updateAndNotify(conn, assetReq)
					}
				}
			}(arReq)
			<-throttle //Rate limit
		}
	}()

	go func() {
		for remReq := range prov.CRemediation {
			go func(assetReq *persistence.AssetRequest) {
				conn, err := persistence.DefaultSession()
				defer conn.Close()
				if err != nil {
					log.Errorf("[areq %s][res %s] Error in getting persistence session :%v", assetReq.Id, assetReq.ResourceId, err)
					return
				}
				if err := prov.terminateFailedResource(assetReq, true); err == nil {
					if err := prov.createServer(conn, assetReq); err == nil {
						log.Debugf("[areq %s][res %s] Notifying VertexPlatform to create an Asset, ServerId:%s", assetReq.Id, assetReq.ResourceId, assetReq.ServerId)
						prov.updateAndNotify(conn, assetReq)
					}
				}
			}(remReq)
		}
	}()

	go func() {
		for resourceId := range prov.CNotification {
			log.Debugf("[res %s] VCG is activated notification received", resourceId)
			go prov.notifyActivation(resourceId)
		}
	}()

	go func() {
		for delReq := range prov.DelNotification {
			log.Debugf("[res %s] Delete notification recevied", delReq.ServerId)
			go stormstack.DomainDeleteAgent(delReq)
			go stormstack.DeRegisterStormAgent(delReq)
			go prov.notifyDeActivation(delReq)
			go prov.notifyDettachAsset(delReq)
		}
	}()

	go prov.RescheduleOldRequests()
}

func (prov *Provisioner) createServer(conn *persistence.Connection, ar *persistence.AssetRequest) (err error) {
	log.Debugf("[areq %s] Creating a VCG", ar.Id)

	serviceProvision, err := cache.GetProvider(&ar.Provider)
	if err != nil {
		log.Criticalf("[%s][%s]No service provision instance, can't proceed with server creation", ar.Id, ar.ResourceId)
		return fmt.Errorf("No valid asset provider credentials")
	}
	created := false
	ar.Status = persistence.RequestBuild
	conn.Update(ar)

	for i := 0; i < 5; i++ {
		entityId := ""
		entityId, fip, err := serviceProvision.ProvisionInstance(ar)
		if err == nil && fip != "" {
			ar.ServerId = entityId
			ar.IpAddress = fip
			created = true
			break
		} else {
			if entityId != "" {
				ar.ServerId = entityId
			}
			perr := err.(*provision.ProvisionError)
			switch perr.Code {
			case provision.ErrorServerCreate, provision.ErrorSettingHostName, provision.ErrorAssociateIP, provision.ErrorStormRegister:
				if len(entityId) > 0 {
					serviceProvision.DeprovisionInstance(ar)
				}
			case provision.ErrorFindFlavor, provision.ErrorFindImage:
				log.Debugf("Image / Flavor not found %v", perr)

			}
		}
		log.Debugf("[arq %s] Provisioning instance failed for the Asset Request. Retrying in 10 seconds", ar.Id)
		time.Sleep(10 * time.Second)
	}
	if created {
		ar.Status = persistence.RequestHalfFilled
	} else {
		//Reschedule it
		log.Debugf("[areq %s] Rescheduling the Asset create request in 5min", ar.Id)
		ar.Status = persistence.RequestRetry
	}
	conn.Update(ar)
	return
}

func (prov *Provisioner) updateAndNotify(conn *persistence.Connection, arRes *persistence.AssetRequest) {
	err := prov.notifyAttachAsset(arRes)
	if err != nil {
		if errors.IsNotFound(err) {
			arRes.Status = persistence.RequestMarkDeletion
		} else {
			arRes.Status = persistence.RequestNotifyFail
		}
	}
	conn.Update(arRes)
}

/*
 * Send this information to VertexPlatform
 */

type NotifyAsset struct {
	Id        string `json:"id"`
	Resource  string `json:"resource"`
	Instance  string `json:"instance"`
	IsActive  bool   `json:"isActive"`
	IpAddress string `json:"ipAddress"`
	AgentId   string `json:"agent"`
	SerialKey string `json:"serialKey"`
}

type NotifyResponse struct {
	Asset NotifyAsset `json:"asset"`
}

type NotifyAgent struct {
	SerialKey string `json:"serialKey"`
}
type ServiceAgent struct {
	ServiceAgent NotifyAgent `json:"serviceAgent"`
}

var (
	nclient = client.NewPublicClient("")
)

type usgTokenReq struct {
	Identification string `json:"identification"`
	Password       string `json:"password"`
}

type usgTokenResp struct {
	Token string `json:"token"`
}

func getNotifierToken() string {
	username := util.GetString("usg", "username")
	password := util.GetString("usg", "password")
	authurl := util.GetString("usg", "authurl")

	var req usgTokenReq
	req.Identification = username
	req.Password = password

	var resp usgTokenResp

	reqData := &goosehttp.RequestData{ReqValue: req, RespValue: &resp,
		ExpectedStatus: []int{http.StatusOK}}

	err := nclient.SendRequest(client.POST, "", authurl, reqData)
	if err != nil {
		log.Errorf("Cannot get token from USG. Error[%v]", err)
		return ""
	}
	return resp.Token

}
func notifyServiceAgent(asset *persistence.AssetRequest) error {

	token := getNotifierToken()
	if token == "" {
		return fmt.Errorf("Not able to fetch token")
	}
	headers := make(http.Header)
	tokenstr := fmt.Sprintf("Bearer %s", token)
	headers.Add("Authorization", tokenstr)

	var agentReq ServiceAgent
	agentReq.ServiceAgent.SerialKey = asset.SerialKey
	time.Sleep(time.Duration(2) * time.Second)

	reqData := &goosehttp.RequestData{ReqHeaders: headers, ReqValue: agentReq,
		ExpectedStatus: []int{http.StatusOK}}
	purl, err := pkgurl.Parse(asset.Notify.Url)
	if err != nil {
		log.Errorf("[areq %s][res %s] cannot parse the url %s", asset.Id, asset.ResourceId, asset.Notify.Url)
		return err
	}
	url := fmt.Sprintf("%s://%s/serviceAgents/%s", purl.Scheme, purl.Host, asset.AgentId)
	log.Debugf("[areq %s][res %s] Updating Caller [%s] with Agent details", asset.Id, asset.ResourceId, url)
	err = nclient.SendRequest(client.PUT, "", url, reqData)
	if err != nil {
		log.Errorf("[areq %s][res %s] Caller Error on agent update. Error[%v]", asset.Id, asset.ResourceId, err)
		return err
	}
	return nil
}

func (prov *Provisioner) notifyAttachAsset(arRes *persistence.AssetRequest) error {

	if arRes.Provider == (persistence.AssetProvider{}) {
		return notifyServiceAgent(arRes)
	}

	var req NotifyResponse
	req.Asset.Id = arRes.Id
	req.Asset.Resource = arRes.ResourceId
	req.Asset.Instance = arRes.ServerId
	req.Asset.IpAddress = arRes.IpAddress
	req.Asset.IsActive = true
	req.Asset.AgentId = arRes.AgentId

	headers := make(http.Header)
	headers.Add("V-Auth-Token", arRes.Notify.Token)
	var resp persistence.AssetRequest
	reqData := &goosehttp.RequestData{ReqHeaders: headers, ReqValue: req, RespValue: &resp,
		ExpectedStatus: []int{http.StatusOK}}
	url := fmt.Sprintf("%s", arRes.Notify.Url)
	log.Debugf("[areq %s][res %s] Updating Caller [%s] with Asset details", arRes.Id, arRes.ResourceId, url)
	err := prov.Client.SendRequest(client.POST, "", url, reqData)
	if err != nil {
		log.Errorf("[areq %s][res %s] Caller Error on Asset attach. Error[%v]", arRes.Id, arRes.ResourceId, err)
		return err
	}
	log.Debugf("[areq %s][res %s] Update to Caller on Asset attach acknowledged", arRes.Id, arRes.ResourceId)
	return nil
}

func (prov *Provisioner) notifyDettachAsset(ar *persistence.AssetRequest) error {
	headers := make(http.Header)
	headers.Add("V-Auth-Token", ar.Notify.Token)
	reqData := &goosehttp.RequestData{ReqHeaders: headers, ExpectedStatus: []int{http.StatusOK, http.StatusNoContent}}
	url := fmt.Sprintf("%s/%s", ar.Notify.Url, ar.Id)

	err := prov.Client.SendRequest(client.DELETE, "", url, reqData)
	if err != nil {
		log.Errorf("[res %s] Caller error on Asset Detach URL [%s],  Error[%v]", ar.ResourceId, err, url)
		return err
	}
	log.Debugf("[res %s] Deleted the attached asset in  Caller [%s]", ar.ResourceId, url)
	return nil
}

func (prov *Provisioner) getAssetProvider(id string) (*persistence.AssetProvider, error) {
	assetProvider := new(persistence.AssetProvider)
	vertexURL := util.GetString("external", "vertex-url")
	url := fmt.Sprintf("%s/assetprovider/%s", vertexURL, id)
	reqData := &goosehttp.RequestData{RespValue: assetProvider}
	err := prov.Client.SendRequest(client.GET, "", url, reqData)
	log.Debugf("Response :" + assetProvider.Username)
	return assetProvider, err
}

func (prov *Provisioner) getAssetModel(id string) (*persistence.AssetModel, error) {
	assetModel := new(persistence.AssetModel)
	vertexURL := util.GetString("external", "vertex-url")
	url := fmt.Sprintf("%s/product/%s", vertexURL, id)
	reqData := &goosehttp.RequestData{RespValue: assetModel}
	err := prov.Client.SendRequest(client.GET, "", url, reqData)
	log.Debugf("Response:" + assetModel.Name)
	return assetModel, err
}

func (prov *Provisioner) RescheduleOldRequests() {
	//Don't start immediately, wait for 5 min and start
	time.Sleep(time.Duration(2) * time.Minute)

	for {
		conn, err := persistence.DefaultSession()
		if err != nil {
			log.Errorf("Error in getting connection :%v", err)
			return
		}
		status := []string{persistence.RequestRetry, persistence.RequestMarkDeletion}
		assetReqs, err := conn.FindAll(bson.M{"status": bson.M{"$in": status}})
		log.Debugf("These many %d asset requests have to retry", len(assetReqs))
		if err != nil {
			return
		}
		for _, assetReq := range assetReqs {
			switch assetReq.Status {
			case persistence.RequestMarkDeletion:
				log.Debugf("[areq %s][res %s] Marked for Deletion. Terminate the asset for HostName[%s]", assetReq.Id, assetReq.ResourceId, assetReq.HostName)
				prov.notifyDeActivation(assetReq)
			case persistence.RequestRetry:
				log.Debugf("[areq %s][res %s]  Recreating the asset HostName[%s]", assetReq.Id, assetReq.ResourceId, assetReq.HostName)
				// Ravi: RequestRetry happens when openstack server creation failed in last attempt. Hence no need to delete salt key
				prov.terminateFailedResource(assetReq, false)
				prov.CRequest <- assetReq
			}

		}

		conn.Close()
		time.Sleep(time.Duration(5) * time.Minute)
	}
	return
}

func (prov *Provisioner) terminateFailedResource(ar *persistence.AssetRequest, deleteSaltKey bool) error {
	prov.notifyDettachAsset(ar)
	if ar.Provider == (persistence.AssetProvider{}) {
		return nil
	}
	return prov.terminateInstance(ar, deleteSaltKey)
}

func (prov *Provisioner) terminateInstance(ar *persistence.AssetRequest, deleteKey bool) (err error) {
	serviceProvision, err := cache.GetProvider(&ar.Provider)
	if err != nil {
		log.Criticalf("[%s][%s]No service provision instance, can't proceed with server deletion", ar.Id, ar.ResourceId)
		return fmt.Errorf("No valid asset provider credentials")
	}

	log.Debugf("[areq %s][res %s] Verify connectivity to the resource", ar.Id, ar.ResourceId)
	//Assumption is, once delete call went to openstack, it will surely get terminated.
	if entity, _ := serviceProvision.GetServer(ar.HostName, ar.ServerId); entity != nil {
		i := 0
		for ; i < 3; i++ {
			log.Debugf("[areq %s][res %s]  About to deprovision the instance with serverID %s", ar.Id, ar.ResourceId, ar.ServerId)
			if err = serviceProvision.DeprovisionInstance(ar); err == nil {
				break
			}
			log.Errorf("[areq %s][res %s] Failed to delete server :%s, Error;%v", ar.Id, ar.ResourceId, ar.ServerId, err)
			time.Sleep(time.Duration(1) * time.Second)
		}
	}

	// waiting 10 sec after terminating the resource
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func (prov *Provisioner) notifyDeActivation(ar *persistence.AssetRequest) (err error) {
	if ar.Provider == (persistence.AssetProvider{}) {
		return nil
	}
	if err = prov.terminateInstance(ar, false); err == nil {
		conn, err := persistence.DefaultSession()
		if err == nil {
			defer conn.Close()
		}
		return conn.Remove(ar.Id)
	}
	return err
}

//On notify activation, modules will be installed and pushes configuration.
func (prov *Provisioner) notifyActivation(resourceId string) error {
	conn, cerr := persistence.DefaultSession()
	if cerr != nil {
		log.Errorf("[res %s] Error in getting connection :%v", resourceId, cerr)
		return cerr
	}
	//find the asset request for this  notification
	ar, err := conn.Find(bson.M{"resourceid": resourceId})
	defer conn.Close()

	if err != nil {
		return err
	}
	if ar.Status == persistence.RequestFulfilled {
		log.Errorf("[res %s] Resource already fullfilled", resourceId)
		return fmt.Errorf("Resource is %s already full filled", resourceId)
	}
	conn.Update(ar)

	if err = prov.activateVertexResource(resourceId); err != nil {
		//TODO This needs to be fixed, what is the correct status
		ar.Status = persistence.RequestRetry
		conn.Update(ar)
		return err
	}

	log.Debugf("[res %s] Successfully activated resource", resourceId)
	ar.Status = persistence.RequestFulfilled
	if ar.Remediation {
		ar.Remediation = false
	}
	ar.Remediation = false
	conn.Update(ar)
	return nil
}

func (prov *Provisioner) activateVertexResource(resourceId string) (err error) {
	resp := make(util.Response)
	reqData := &goosehttp.RequestData{RespValue: &resp,
		ExpectedStatus: []int{http.StatusOK}}
	vertexPlatformURL := util.GetString("external", "vertex-url")
	url := fmt.Sprintf("%s/resource/%s/activated", vertexPlatformURL, resourceId)
	log.Debugf("[res %s] Setting resource into Active state", resourceId)
	err = prov.Client.SendRequest(client.PUT, "", url, reqData)

	if err != nil {
		log.Errorf("[res %s] Changing resource status failed,  Error[%v]", resourceId, err)
		return
	}
	return
}

func (prov *Provisioner) UploadImage(assetProvider *persistence.AssetProvider, req *http.Request) string {
	svcProv, err := cache.GetProvider(assetProvider)
	if err != nil {
		log.Criticalf("No service provision instance, can't proceed with image upload")
		return util.Response{"error": "No service provision instance, can't proceed with image upload"}.String()
	}

	imageDetails, err := svcProv.UploadImageToGlance(req)
	if err != nil {
		return util.Response{"error": "Unable to upload the image"}.String()
	}
	return util.ToString(imageDetails)
}

func (prov *Provisioner) ValidateAssetProvider(ap *persistence.AssetProvider) (authDetails *identity.AuthDetails, err error) {
	authDetails, err = provision.ValidateAssetProvider(ap)
	return
}

func (prov *Provisioner) CheckFIPAvailability(ar *persistence.AssetRequest) (count int, err error) {
	if serviceProvision, err := cache.GetProvider(&ar.Provider); err == nil {
		return serviceProvision.CheckAvailability()
	}
	return -1, fmt.Errorf("Floating IPS are not available")
}

func (prov *Provisioner) RenameServer(assetId, newName string) error {
	conn, err := persistence.DefaultSession()
	if err != nil {
		log.Errorf("areq %s] Error in getting connection :%v", assetId, err)
		return err
	}
	defer conn.Close()
	ar, _ := conn.Find(bson.M{"_id": assetId})
	svcProv, err := cache.GetProvider(&ar.Provider)
	if err != nil {
		log.Criticalf("[%s][%s]No service provision instance, can't proceed with renaming server ", ar.Id, ar.ResourceId)
		return fmt.Errorf("No valid asset provider credentials, can't rename")
	}

	ar.HostName = newName
	conn.Update(ar)
	return svcProv.RenameServer(ar.ServerId, newName)
}
