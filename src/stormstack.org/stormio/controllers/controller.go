package controllers

import (
	"stormstack.org/stormio/persistence"
	"stormstack.org/stormio/scheduler"
	"stormstack.org/stormio/util"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	log "github.com/cihub/seelog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"labix.org/v2/mgo/bson"
	"net/http"
	"os"
)

// var assetDS = new(persistence.AssetDS)
var provisioner *scheduler.Provisioner

func StartServer(host string, port string) {
	contextPath := util.GetString("web-app", "context-path")
	router := mux.NewRouter()

	initAssetRoutes(contextPath, router)
	initResourceMappings(contextPath, router)
	initAssetProviderMappings(contextPath, router)
	initSvc()
	appName := util.GetString("application", "name")
	log.Infof("%s running @ %s:%s", appName, host, port)
	out, _ := os.Create(util.GetString("path", "access-log"))
	handler := handlers.LoggingHandler(out, router)
	http.ListenAndServe(host+":"+port, handler)
}

func initSvc() {
	provisioner = scheduler.NewProvisioner()
}

/*
func initDB() {
	dbName := util.GetString("database", "db-name")
	username := util.GetString("database", "username")
	password := util.GetString("database", "password")
	host := util.GetString("database", "host")
	port := util.GetString("database", "port")
	assetDS.InitDatabase("mysql", dbName, username, password, host, port)
}
*/

func initResourceMappings(contextPath string, router *mux.Router) {
	subRouter := router.PathPrefix(contextPath + "/resource").Subrouter()
	subRouter.HandleFunc("/{id}/status", resourceStatus).Methods("GET")
}

func initAssetProviderMappings(contextPath string, router *mux.Router) {
	subRouter := router.PathPrefix(contextPath + "/assetprovider").Subrouter()
	subRouter.HandleFunc("/image", listImages).Methods("GET")
	subRouter.HandleFunc("/flavor", listFlavors).Methods("GET")
	subRouter.HandleFunc("/image/upload", uploadImage).Methods("POST")
	subRouter.HandleFunc("/validate", validateAssetProvider).Methods("POST")
	subRouter.HandleFunc("/service/{name}/test", validateProvidersService).Methods("POST")
}

//Assetprovider information is going to come as AES/ECB/PKCS5Padding
func listImages(response http.ResponseWriter, request *http.Request) {
	if prov, err := ValidateAssetProvider(response, request); err == nil {
		images, err := prov.ListImageNames()
		if err != nil {
			sendErrorResponse(response, http.StatusBadGateway, err)
			return
		}
		sendResponse(images.String(), http.StatusOK, response)
		return
	} else {
		sendErrorResponse(response, http.StatusBadGateway, err)
	}
}

func listFlavors(response http.ResponseWriter, request *http.Request) {
	if prov, err := ValidateAssetProvider(response, request); err == nil {
		images, err := prov.ListFlavorNames()
		if err != nil {
			sendErrorResponse(response, http.StatusBadGateway, err)
			return
		}
		sendResponse(images.String(), http.StatusOK, response)
		return
	} else {
		sendErrorResponse(response, http.StatusBadGateway, err)
	}
}

func uploadImage(response http.ResponseWriter, request *http.Request) {
	assetProvider, err := extractAssetProvider(request.Header.Get("Authorization"))
	if err != nil {
		sendResponse("Error in extracting AssetProvider", http.StatusInternalServerError, response)
		return
	}
	log.Info("Uploading Image, delegating to Provisioner")
	response.Header().Set("Content-Type", "application/json")
	sendResponse(util.ToString(provisioner.UploadImage(assetProvider, request)), http.StatusOK, response)
	return
}

func validateProvidersService(response http.ResponseWriter, request *http.Request) {
	return
}

func validateAssetProvider(response http.ResponseWriter, request *http.Request) {
	return
}

func resourceStatus(response http.ResponseWriter, request *http.Request) {
	resourceId := mux.Vars(request)["id"]
	type Status struct {
		Installed  bool
		Configured bool
	}

	var statusResponse struct {
		Percentage   int                `json:"percentage"`
		Status       string             `json:"status"`
		ModuleStatus map[string]*Status `json:"moduleStatus"`
	}

	conn, err := persistence.DefaultSession()
	if err != nil {
		log.Errorf("Error in getting connection :%v", err)
		sendResponse("Error in getting connection", http.StatusInternalServerError, response)
		return
	}
	defer conn.Close()

	//find the asset request for this resource
	ar, err := conn.Find(bson.M{"resourceid": resourceId})

	if err != nil {
		sendResponse("Resource not found", http.StatusNotFound, response)
		return
	}
	statusResponse.ModuleStatus = make(map[string]*Status)
	mpercent := 0
	mCount := len(ar.Modules)
	increment := 0
	if len(ar.Modules) > 0 {
		increment = 30 / mCount
	}
	for _, module := range ar.Modules {
		status := &Status{Installed: module.Installed, Configured: module.Configured}
		statusResponse.ModuleStatus[module.Name] = status
		if status.Installed {
			mpercent += increment
		}
		if status.Configured {
			mpercent += increment
		}
	}
	percent := 0
	switch ar.Status {
	case persistence.RequestNew:
		percent = 0
	case persistence.RequestBuild:
		percent = 10
	case persistence.RequestHalfFilled:
		percent = 40 + mpercent
	case persistence.RequestRetryModuleInstall:
		percent = 40 + mpercent
	case persistence.RequestRetryModuleConfig:
		percent = 40 + mpercent
	case persistence.RequestRetry:
		percent = 40 + mpercent
	case persistence.RequestProvision:
		percent = 40 + mpercent
	case persistence.RequestFulfilled:
		percent = 100
	}

	statusResponse.Status = ar.Status
	statusResponse.Percentage = percent
	sendResponse(util.ToString(statusResponse), http.StatusOK, response)
	return
}

func sendErrorResponse(response http.ResponseWriter, errorCode int, err error) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(errorCode)
	respMap := util.Response{"error": err.Error()}
	response.Write([]byte(respMap.String()))
}

func sendResponse(out string, responseCode int, response http.ResponseWriter) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(responseCode)
	response.Write([]byte(out))
}

func sendByteResponse(out []byte, responseCode int, response http.ResponseWriter) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(responseCode)
	response.Write(out)
}

func extractAssetProvider(encoded string) (*persistence.AssetProvider, error) {
	decoded, _ := base64.StdEncoding.DecodeString(encoded)
	log.Debugf("Asset Provider:%s", decoded)
	// if decrypted, err := decryptAESCFB(encrypted); err == nil {
	assetProvider := new(persistence.AssetProvider)
	util.ToObject([]byte(decoded), assetProvider)
	return assetProvider, nil
}

//AES/ECB/PKCS5Padding
func decryptAESCFB(src []byte) (string, error) {
	key := util.GetString("encyrption", "secret")
	var iv = []byte(key)[:aes.BlockSize] // Using IV same as key is probably bad
	aesBlockDecrypter, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	decrypted := make([]byte, 1024)
	aesDecrypter := cipher.NewCFBDecrypter(aesBlockDecrypter, iv)
	aesDecrypter.XORKeyStream(decrypted, src)
	return string(decrypted), nil
}
