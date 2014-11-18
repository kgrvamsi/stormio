package persistence

import (
	"encoding/json"
	"github.com/nu7hatch/gouuid"
	"net/http"
)

type AssetProvider struct {
	Id             string `json:"id" bson:"_id"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	EndPointURL    string `json:"endPoint"`
	Tenant         string `json:"tenant"`
	RegionName     string `json:"regionName"`
	DefaultNetName string `json:"defaultNetName,omitempty"`
	NetworkName    string `json:"networkName,omitempty"`
	RouterId       string `json:"routerId,omitempty"`
	NetworkId      string `json:"networkId,omitempty"`
	Image          string `json:"image,omitempty"`
	Compute        string `json:"compute,omitempty"`
	Storage        string `json:"storage,omitempty"`
	Neutron        string `json:"neutron,omitempty"`
	Identity       string `json:"identity,omitempty"`
}

type AssetModel struct {
	Id     string `json:"id" bson:"_id"`
	Name   string `json:"name"`
	Flavor string `json:"flavor"`
	Image  string `json:"image"`
}

type StormBolt struct {
	Uplinks        []string `json:"uplinks"`
	UplinkStrategy string   `json:"uplinkStrategy"`
	AllowRelay     bool     `json:"allowRelay"`
	RelayPort      int      `json:"relayPort"`
	AllowedPorts   []int    `json:"allowedPorts"`
	ListenPort     int      `json:"listenPort"`
	BeaconInterval int      `json:"beaconInterval"`
	BeaconRetry    int      `json:"beaconRetry"`
	BeaconValidty  int      `json:"beaconValidity"`
}

type StormAgent struct {
	Id        string    `json:"id"`
	SerialKey string    `json:"serialkey"`
	Stoken    string    `json:"stoken"`
	StormBolt StormBolt `json:"bolt"`
}

//Control Provider
type ControlProvider struct {
	Id              string    `json:"id" bson:"_id"`
	StormtrackerURL string    `json:"stormtracker"`
	StormlightURL   string    `json:"stormlight"`
	StormkeeperURL  string    `json:"stormkeeper"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Bolt            StormBolt `json:"bolt"`
	DefaultDomainId string    `json:"domain"`
}

const (
	RequestNew                = "NEW"
	RequestFail               = "FAIL"
	RequestBuild              = "BUILD"
	RequestHalfFilled         = "SERVER_CREATED"
	RequestMarkDeletion       = "MARKED_FOR_DELETION"
	RequestFulfilled          = "FUL_FILLED"
	RequestProvision          = "SERVICE_PROVISION"
	RequestRetry              = "RETRY"
	RequestRetryModuleInstall = "RETRY_MINSTALL"
	RequestRetryModuleConfig  = "RETRY_MCONFIG"
	RequestNotifyFail         = "NOTIFICATION_FAILED"
	RequestRemediation        = "REMEDIATION"
)

type NotifyCaller struct {
	Url   string `json:"url"`
	Token string
}

// Entity AssetRequest
type AssetRequest struct {
	Id              string        `json:"id,omitempty" bson:"_id"` // qbs:"pk"`
	HostName        string        `json:"hostName"`
	ResourceId      string        `json:"resource"`
	ServerId        string        `json:"serverId"`
	IpAddress       string        `json:"ipAddress"`
	ReceivedOn      string        `json:"receivedOn"`
	ModelId         string        `json:"modelId"`
	Provider        AssetProvider `json:"assetProvider"`
	Status          string        `json:"status"`
	PreviousStatus  string
	Remediation     bool
	Model           AssetModel     `json:"assetModel"`
	Modules         []ModuleStatus `json:"-" bson:"-"`
	ModuleInitFlag  bool           `json:"-" bson:"-"`
	Logs            []Log          `json:"logs"`
	ActivationInfo  ActivationInfo
	ControlTokenId  string          `json:"stormTokenId"`
	SerialKey       string          `json:"serialkey"`
	AgentId         string          `json:"agentId"`
	ControlProvider ControlProvider `json:"controlProvider"`
	Notify          NotifyCaller
}

type ActivationInfo struct {
	Controller string
	Status     string
}

type Log struct {
	Msg  string
	Type string
}

type ModuleStatus struct {
	Id             string `json:"-" bson:"-"` // qbs:"pk"`
	Name           string
	Installed      bool
	Configured     bool
	AssetRequestId string `qbs:"fk:AssetRequest" bson:"-" json:"-"`
}

type ConfigPassThru struct {
	ResourceId string `bson:"_id"`
	Config     map[string]Configuration
}

type Configuration struct {
	Object interface{}
	Pushed bool
}

func NewUUID() string {
	u4, _ := uuid.NewV4()
	return u4.String()
}

func (ar *AssetRequest) DecodeFromRequest(req *http.Request) error {
	var jar AssetRequest = AssetRequest(*ar)
	dec := json.NewDecoder(req.Body)
	err := dec.Decode(&jar)
	*ar = AssetRequest(jar)
	return err
}

func (ar *AssetRequest) DecodeFromBuffer(buffer []byte) error {
	var jar AssetRequest = AssetRequest(*ar)
	err := json.Unmarshal(buffer, &jar)
	*ar = AssetRequest(jar)
	return err
}
