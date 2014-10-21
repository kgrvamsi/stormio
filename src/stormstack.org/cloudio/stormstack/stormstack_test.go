// stormstack_test test functions in stormstack.go
// Required to have cloudio config. 
// In the configuration set vertex host to localhost and port to 8080

package stormstack_test

import (
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"stormstack.org/cloudio/persistence"
	"github.com/gorilla/handlers"
	log "github.com/cihub/seelog"
	"stormstack.org/cloudio/util"
	"time"
	"testing"
	. "gopkg.in/check.v1"
	"os"
	. "stormstack.org/cloudio/stormstack"
	"syscall"
	"io/ioutil"
	"encoding/json"
)
// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }


type StormSuite struct {
	id string
}

const (
	apiAgents = "/agents"
	apiControlproviders = "/VertexPlatform/control/provider"
	localhost	= "localhost"
	port		= "8080"
	controlId	= "1"
	token		= "token"
	agentId		= "1234"
)

var _ = Suite(&StormSuite{})

func (st *StormSuite) SetUpSuite(c *C) {
    util.LoadProperties()
    defer log.Flush()
    seelogconfig := util.GetString("path", "log-conf")
    log.LoggerFromConfigAsFile(seelogconfig)
    // start an local webserver that hosts endpoints agents and controlprovider
    go startServer("localhost", "8080")
}

func (st *StormSuite) TestOnce(c *C) {
    assetReq := persistence.AssetRequest{HostName: "Testing Host", ResourceId: persistence.NewUUID(),
        ReceivedOn: time.Now().String(), ControlTokenId: token,ControlProviderId: controlId}

    stormdata, err := BuildStormData(&assetReq)
    if err != nil {
	c.Error(err, "failed to build stormdata")
    }
    expected  := fmt.Sprintf("http://%s@%s:%s/skey",token, localhost,port)
    c.Assert(stormdata, Equals, expected)
    skey := persistence.NewUUID()
    err = RegisterStormAgent(&assetReq, skey) 
    if err != nil {
        c.Error(err, "failed to register stormagent")
    }
    err = DeRegisterStormAgent(&assetReq)
    if err != nil {
        c.Error(err, "failed to deregister stormagent")
    }
}



func startServer(host string, port string) {
	router := mux.NewRouter()
	router.HandleFunc(apiAgents,handleAgents).Methods("POST")
	router.HandleFunc(apiAgents+"/{id}",handleAgentsDel).Methods("DELETE")
	router.HandleFunc(apiControlproviders+"/{id}", handleProviders).Methods("GET")
	Stdout	:= os.NewFile(uintptr(syscall.Stdout), "/dev/stdout")
	handler := handlers.LoggingHandler(Stdout, router)
	fmt.Printf("starting web server")
	http.ListenAndServe(host+":"+port, handler)
}

func handleAgents(response http.ResponseWriter, request *http.Request) {
	body,_ := ioutil.ReadAll(request.Body)
	var b persistence.StormAgent
	json.Unmarshal(body, &b)
	fmt.Printf("Body recieved is %v, json is %v", string(body),  b)
	response.Header().Set("Content-Type", "application/json")
	agent := persistence.StormAgent{Id:agentId, Stoken:token}
	sendResponse(util.ToString(agent),http.StatusOK, response)
	return
}
func handleAgentsDel(response http.ResponseWriter, request *http.Request) {
	agentid := mux.Vars(request)["id"]
	fmt.Printf("handleAgentsDel: Fetched agent id is %v", agentid);
	response.Header().Set("Content-Type", "application/json")
	if agentid == agentId {
		response.WriteHeader(http.StatusNoContent)
	} else {
		response.WriteHeader(http.StatusNotFound)
	}
	return
}
func handleProviders(response http.ResponseWriter, request *http.Request) {
        response.Header().Set("Content-Type", "application/json")
	url := fmt.Sprintf("http://%s:%s", localhost, port)
        //provider := persistence.ControlProvider{Id:controlId, StormtrackerURL: url, StormkeeperURL: url}
        provider := persistence.ControlProvider{Id:controlId, StormtrackerURL: url}
        sendResponse(util.ToString(provider),http.StatusOK, response)
        return
}

func sendResponse(out string, responseCode int, response http.ResponseWriter) {
    response.Header().Set("Content-Type", "application/json")
    response.WriteHeader(responseCode)
    response.Write([]byte(out))
}


