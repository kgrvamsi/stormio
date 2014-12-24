package cache

import (
	"stormstack.org/stormio/persistence"
	"stormstack.org/stormio/provision"
	log "github.com/cihub/seelog"
	goosehttp "launchpad.net/goose/http"
	"sync"
)

var (
	ap = struct {
		svcProvCache map[string]*provision.ServiceProvision
		sync.RWMutex
	}{svcProvCache: make(map[string]*provision.ServiceProvision)}

	gooseclient = goosehttp.New()
)

// func loadAssetProvider(id string) (*persistence.AssetProvider, error) {
//	assetProvider := new(persistence.AssetProvider)
//	vertexURL := util.GetString("external", "vertex-url")
//	url := fmt.Sprintf("%s/assetprovider/%s", vertexURL, id)
//	reqData := &goosehttp.RequestData{RespValue: assetProvider}
//	err := gooseclient.JsonRequest("GET", url, "", reqData)
//	return assetProvider, err
// }

/*
* Check in cache, otherwise keep it in cache and return
 */
func GetProvider(ar *persistence.AssetProvider) (*provision.ServiceProvision, error) {
	ap.Lock()
	defer ap.Unlock()
	key := ar.Username + ":" + ar.Password + ":" + ar.EndPointURL
	svcProv, found := ap.svcProvCache[key]
	if found {
		return svcProv, nil
	} else {
		svcProv = provision.NewServiceProvision(ar)
		if err := svcProv.Ping(); err != nil {
			return nil, err
		}
		ap.svcProvCache[key] = svcProv
		log.Debug("[areq %s] Not Found Asset Provider in Cache, Keep it in Cache UUID", ar.EndPointURL)
		return svcProv, nil
	}
}

/*
func GetProviderById(id string) *provision.ServiceProvision {
	ap.Lock()
	defer ap.Unlock()
	svcProv, found := ap.svcProvCache[id]
	if found {
		return svcProv
	} else {
		assetProvCreds, _ := loadAssetProvider(id)
		svcProv = provision.NewServiceProvision(assetProvCreds)
		ap.svcProvCache[id] = svcProv
		log.Debug("Not Found Asset Provider in Cache, Keep it in Cache UUID:" + id)
		return svcProv
	}
}
*/
