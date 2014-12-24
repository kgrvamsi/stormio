package provision

import (
	dba "stormstack.org/stormio/persistence"
	log "github.com/cihub/seelog"
	"launchpad.net/goose/identity"
)

func ValidateAssetProvider(ap *dba.AssetProvider) (authDetails *identity.AuthDetails,err error) {
	log.Debugf("Validating asset provider details :%v", ap)
    userpass:=&identity.UserPass{}
	creds := &identity.Credentials{URL: ap.EndPointURL,
		User:       ap.Username,
		Secrets:    ap.Password,
		Region:     ap.RegionName,
		TenantName: ap.Tenant}
        authDetails,err=userpass.Auth(creds)
	    if err!=nil{
	        return
        }
	return
}
