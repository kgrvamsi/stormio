package provision

import (
	log "github.com/cihub/seelog"
	"launchpad.net/goose/glance"
	"net/http"
)

func (svc *ServiceProvision) UploadImageToGlance(req *http.Request) (*glance.ImageDetail, error) {
	log.Info("Delegating to glance")
	return svc.glance.UploadImage(req)
}
