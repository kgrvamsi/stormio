package identity

import (
	"fmt"
	goosehttp "launchpad.net/goose/http"
	"time"
)

const OS_TIME_FORMAT = "2006-01-02T15:04:05Z07:00"

type passwordCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authRequest struct {
	PasswordCredentials passwordCredentials `json:"passwordCredentials"`
	TenantName          string              `json:"tenantName"`
}

type authWrapper struct {
	Auth authRequest `json:"auth"`
}

type endpoint struct {
	AdminURL    string `json:"adminURL"`
	InternalURL string `json:"internalURL"`
	PublicURL   string `json:"publicURL"`
	Region      string `json:"region"`
}

type serviceResponse struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Endpoints []endpoint
}

type tokenResponse struct {
	Expires string `json:"expires"` // should this be a date object?
	Id      string `json:"id"`      // Actual token string
	Tenant  struct {
		Id          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
	} `json:"tenant"`
}

type roleResponse struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	TenantId string `json:"tenantId"`
}

type userResponse struct {
	Id    string         `json:"id"`
	Name  string         `json:"name"`
	Roles []roleResponse `json:"roles"`
}

type accessWrapper struct {
	Access accessResponse `json:"access"`
}

type accessResponse struct {
	ServiceCatalog []serviceResponse `json:"serviceCatalog"`
	Token          tokenResponse     `json:"token"`
	User           userResponse      `json:"user"`
}

type UserPass struct {
	client *goosehttp.Client
}

func (u *UserPass) Auth(creds *Credentials) (*AuthDetails, error) {
	if u.client == nil {
		u.client = goosehttp.New()
	}
	auth := authWrapper{Auth: authRequest{
		PasswordCredentials: passwordCredentials{
			Username: creds.User,
			Password: creds.Secrets,
		},
		TenantName: creds.TenantName}}

	var accessWrapper accessWrapper
	requestData := goosehttp.RequestData{ReqValue: auth, RespValue: &accessWrapper}
	err := u.client.JsonRequest("POST", creds.URL, "", &requestData)
	if err != nil {
		return nil, err
	}

	details := &AuthDetails{}
	access := accessWrapper.Access
	respToken := access.Token
	if respToken.Id == "" {
		return nil, fmt.Errorf("Did not get valid Token from auth request")
	}
	details.Token = respToken.Id
	details.TenantId = respToken.Tenant.Id
	details.ExpiresAt, _ = time.Parse(OS_TIME_FORMAT, respToken.Expires)
	details.UserId = access.User.Id
	details.RegionServiceURLs = make(map[string]ServiceURLs, len(access.ServiceCatalog))
	for _, service := range access.ServiceCatalog {
		for i, e := range service.Endpoints {
			endpointURLs, ok := details.RegionServiceURLs[e.Region]
			if !ok {
				endpointURLs = make(ServiceURLs)
				details.RegionServiceURLs[e.Region] = endpointURLs
			}
			endpointURLs[service.Type] = service.Endpoints[i].PublicURL
		}
	}
	return details, nil
}
