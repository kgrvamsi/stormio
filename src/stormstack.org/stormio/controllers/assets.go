package controllers

import (
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/gorilla/mux"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"net/http"
	"stormstack.org/stormio/cache"
	"stormstack.org/stormio/persistence"
	"stormstack.org/stormio/provision"
	"stormstack.org/stormio/util"
	"time"
)

func initAssetRoutes(contextPath string, router *mux.Router) {
	router.HandleFunc(contextPath+"/createAsset", createAsset).Methods("POST")
	router.HandleFunc(contextPath+"/deleteAsset", destroyAsset).Methods("POST")
	subRouter := router.PathPrefix(contextPath + "/tasks").Subrouter()
	subRouter.HandleFunc("/{id}", retrieveAsset).Methods("GET")
	//subRouter.HandleFunc("/{id}/rename/{newName}", renameAsset).Methods("PUT")
	//subRouter.HandleFunc("/{id}", destroyAsset).Methods("DELETE")
}

// CRUD for AssetRequest starts from here
func retrieveAsset(response http.ResponseWriter, request *http.Request) {
	anAssetId := mux.Vars(request)["id"]
	conn, err := persistence.DefaultSession()
	if err != nil {
		sendResponse("DB connection failure", http.StatusServiceUnavailable, response)
		return
	}
	defer conn.Close()
	if ar, err := conn.Find(bson.M{"_id": anAssetId}); err != nil {
		sendErrorResponse(response, http.StatusNotFound, err)
	} else {
		b, _ := json.Marshal(ar)
		sendByteResponse(b, http.StatusOK, response)
	}
	return
}

func createAsset(response http.ResponseWriter, request *http.Request) {
	asset := &persistence.AssetRequest{}
	asset.DecodeFromRequest(request)
	asset.Id = persistence.NewUUID()       //set the new uuid
	asset.ReceivedOn = time.Now().String() //set the created time
	asset.Status = persistence.RequestNew
	asset.ModelId = asset.Model.Id
	log.Debugf("Asset Request recieved is %#v", asset)
	conn, err := persistence.DefaultSession()
	if err != nil {
		sendErrorResponse(response, http.StatusInternalServerError, err)
		return
	}
	//validate the assetprovider
	prov, err := cache.GetProvider(&asset.Provider)
	if err != nil {
		sendErrorResponse(response, http.StatusBadRequest, err)
		return
	}

	if asset.AttachFIP {
		if count, err := prov.CheckAvailability(); count <= 0 || err != nil {
			log.Debugf("No FIP available, sending 412 to caller")
			sendErrorResponse(response, http.StatusPreconditionFailed, fmt.Errorf("No FIP available"))
			return
		} else {
			log.Debugf("Still %d fips are available", count)
		}
	}

	if asset.Notify.Url == "" {
		sendErrorResponse(response, http.StatusPreconditionFailed, fmt.Errorf("No Notify callback URL present"))
		return
	}

	if asset.AgentId == "" {
		sendErrorResponse(response, http.StatusPreconditionFailed, fmt.Errorf("No agentId present in the request"))
		return
	}

	if asset.HostName == "" {
		asset.HostName = asset.Model.Name
	}

	if (asset.Model.Networks == nil || (asset.Model.Networks != nil && len(asset.Model.Networks) == 0)) && asset.Provider.Neutron != "" {
		sendErrorResponse(response, http.StatusPreconditionFailed, fmt.Errorf("No Networks present in the request"))
		return
	}

	for _, network := range asset.Model.Networks {
		log.Debug("network is %v", network)
		if network["uuid"] == "" {
			sendErrorResponse(response, http.StatusPreconditionFailed, fmt.Errorf("No Networks present in the request"))
			return
		}
	}

	err = conn.Create(asset)
	defer conn.Close()
	if err != nil {
		sendErrorResponse(response, http.StatusInternalServerError, err)
		return
	}

	resp := util.ToString(asset)
	//Provision for new instance starts from here
	log.Debug("Asset request created in the db, uuid:" + asset.Id + " Passing request to scheduler")
	provisioner.CRequest <- asset
	log.Debug("Passed to the scheduler, returning 202..")
	sendResponse(resp, http.StatusAccepted, response)
	return
}

type AssetDestroy struct {
	Id string `json:"id"`
}

func destroyAsset(response http.ResponseWriter, request *http.Request) {
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		sendErrorResponse(response, http.StatusBadRequest, fmt.Errorf("Could not read the request body"))
		return
	}

	var aAsset AssetDestroy
	err = json.Unmarshal(body, &aAsset)
	if err != nil {
		sendErrorResponse(response, http.StatusBadRequest, fmt.Errorf("Could not unmarshal the request body"))
		return
	}

	log.Debugf("Finding an asset with ID %v in DB", aAsset.Id)
	conn, err := persistence.DefaultSession()
	if err != nil {
		sendResponse("DB connection failure", http.StatusServiceUnavailable, response)
		return
	}
	defer conn.Close()
	asset, err := conn.Find(bson.M{"_id": aAsset.Id})

	if err != nil {
		sendResponse("Asset not found / already deleted", http.StatusNotFound, response)
		log.Debugf("Error in finding asset %s %v", asset.Id, err)
		return
	}
	log.Debugf("Asset Request recieved is %#v", asset)
	provisioner.DelNotification <- asset
	defer conn.Close()
	sendResponse("Asset "+aAsset.Id+" delete request accepted", http.StatusAccepted, response)
}

func ValidateAssetProvider(response http.ResponseWriter, request *http.Request) (*provision.ServiceProvision, error) {
	if assetProvider, err := extractAssetProvider(request.Header.Get("Authorization")); err != nil {
		return nil, err
	} else {
		return cache.GetProvider(assetProvider)
	}
}

func renameAsset(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	assetId := vars["id"]
	newName := vars["newName"]
	conn, err := persistence.DefaultSession()
	if err != nil {
		sendResponse("DB connection failure", http.StatusServiceUnavailable, response)
		return
	}
	defer conn.Close()
	if asset, err := conn.Find(bson.M{"_id": assetId}); err == nil {
		if prov, err := cache.GetProvider(&asset.Provider); err == nil {
			if err := prov.RenameServer(asset.ServerId, newName); err != nil {
				sendResponse("{'error':'Unable to rename'}", http.StatusExpectationFailed, response)
				return
			}
			sendResponse("{'message':'Rename success'}", http.StatusOK, response)
		} else {
			sendErrorResponse(response, http.StatusBadRequest, err)
		}
	} else {
		sendErrorResponse(response, http.StatusNotFound, err)
	}
	return
}
