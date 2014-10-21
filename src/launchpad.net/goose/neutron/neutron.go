// Go package to interact with OpenStack network (Neutron) API.
// See https://wiki.openstack.org/wiki/Neutron
package neutron

import (
	"fmt"
	"launchpad.net/goose/client"
	"launchpad.net/goose/errors"
	goosehttp "launchpad.net/goose/http"
	"net/http"
	"net/url"
)

//API URL parts

const (
	apiNetworks    = "v2.0/networks"
	apiSubnets     = "v2.0/subnets"
	apiRouters     = "v2.0/routers"
	apiFloatingIPs = "v2.0/floatingips"
	apiPorts       = "v2.0/ports"
)

// Client provides a means to access the OpenStack Neutron Service.
type Client struct {
	client client.Client
}

// New creates a new Client.
func New(client client.Client) *Client {
	return &Client{client}
}

type FloatingIP struct {
	FloatingNetworkId string `json:"floating_network_id,omitempty"`
	TenantId          string `json:"tenant_id,omitempty"`
	FixedIPAddress    string `json:"fixed_ip_address,omitempty"`
	FloatingIPAddress string `json:"floating_ip_address,omitempty"`
	Id                string `json:"id,omitempty"`
	PortId            string `json:"port_id,omitempty"`
	RouterId          string `json:"router_id,omitempty"`
}

type Network struct {
	Status          string   `json:"status"`
	Subnets         []string `json:"subnets"`
	Name            string   `json:"name"`
	PhysicalNetwork string   `json:provider:physical_network`
	AdminStateUp    bool     `json:admin_state_up`
	TenantId        string   `json:tenant_id`
	NetworkType     string   `json:provider:network_type`
	External        bool     `json:router:external`
	Shared          bool     `json:"shared"`
	Id              string   `json:"id"`
	SegmentationId  int32    `json:provider:segmentation_id`
}

type IPRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Subnet struct {
	Name            string    `json:"name"`
	EnableDhcp      bool      `json:"enable_dhcp"`
	NetworkId       string    `json:"network_id"`
	TenantId        string    `json:"tenant_id"`
	DnsNameservers  []string  `json:"dns_nameservers"`
	AllocationPools []IPRange `json:"allocation_pools"`
	HostRoutes      []string  `json:"host_routes"`
	IpVersion       int       `json:"ip_version"`
	GatewayIp       string    `json:"gateway_ip"`
	Cidr            string    `json:"cidr"`
	Id              string    `json:"id"`
}

type FixedIp struct {
	SubnetId  string `json:"subnet_id"`
	IpAddress string `json:"ip_address"`
}

type Port struct {
	NetworkId    string    `json:"network_id"`
	FixedIps     []FixedIp `json:"fixed_ips"`
	AdminStateUp bool      `json:"admin_state_up"`
	Id           string    `json:"id"`
	Status       string    `json:"status"`
	Name         string    `json:"name"`
	TenantId     string    `json:"tenant_id"`
	DeviceOwner  string    `json:"device_owner"`
	MacAddress   string    `json:"mac_address"`
	DeviceId     string    `json:"device_id"`
}

type ExternalGatewayInfo struct {
	NetworkId string `json:"network_id"`
}

type Router struct {
	Status              string               `json:"status"`
	Name                string               `json:"name"`
	AdminStateUp        bool                 `json:"admin_state_up"`
	TenantId            string               `json:"tenant_id"`
	Id                  string               `json:"id"`
	ExternalGatewayInfo *ExternalGatewayInfo `json:"external_gateway_info"`
}

func (c *Client) ListNetworks() ([]Network, error) {
	var resp struct {
		Networks []Network `json:"networks"`
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "network", apiNetworks, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of networks ")
	}
	return resp.Networks, nil
}

func (c *Client) FindNetwork(networkId string) (*Network, error) {
	var resp struct {
		Network Network `json:"network"`
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	url := fmt.Sprintf("%s/%s", apiNetworks, networkId)
	err := c.client.SendRequest(client.GET, "network", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get network details")
	}
	return &resp.Network, nil
}

func (c *Client) CreateNetwork(network *Network) (*Network, error) {
	type typenetwork struct {
		Network *Network `json:"network"`
	}
	var req, resp typenetwork
	req.Network = network
	requestData := &goosehttp.RequestData{ReqValue: req, RespValue: &resp,
		ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.POST, "network", apiNetworks+".json", requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to create network")
	}
	return resp.Network, nil
}

func (c *Client) DeleteNetwork(networkId string) error {
	url := fmt.Sprintf("%s/%s.json", apiNetworks, networkId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.DELETE, "network", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to delete network %s ", networkId)
	}
	return err
}

func (c *Client) CreateSubnet(subnet *Subnet) (*Subnet, error) {
	type typesubnet struct {
		Subnet *Subnet `json:"subnet"`
	}
	var req, resp typesubnet
	req.Subnet = subnet
	requestData := &goosehttp.RequestData{ReqValue: req, RespValue: &resp,
		ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.POST, "network", apiSubnets+".json", requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to create subnet")
	}
	return resp.Subnet, nil
}

func (c *Client) ListSubnets() ([]Subnet, error) {
	var resp struct {
		Subnets []Subnet
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "network", apiSubnets+".json", &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of networks ")
	}
	return resp.Subnets, nil
}

func (c *Client) FindSubnet(subnetId string) (*Subnet, error) {
	var resp struct {
		Subnet Subnet `json:"subnet"`
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	url := fmt.Sprintf("%s/%s.json", apiSubnets, subnetId)
	err := c.client.SendRequest(client.GET, "network", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get subnet details")
	}
	return &resp.Subnet, nil
}

func (c *Client) DeleteSubnet(subnetId string) error {
	url := fmt.Sprintf("%s/%s.json", apiSubnets, subnetId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.DELETE, "network", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to delete subnet %s ", subnetId)
	}
	return err
}

func (c *Client) CreateRouter(router *Router) (*Router, error) {
	type typerouter struct {
		Router *Router `json:"router"`
	}
	var req, resp typerouter
	req.Router = router
	requestData := &goosehttp.RequestData{ReqValue: req, RespValue: &resp,
		ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.POST, "network", apiRouters, requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to create router")
	}
	return resp.Router, nil
}

func (c *Client) ListRouters() ([]Router, error) {
	var resp struct {
		Routers []Router
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "network", apiRouters, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of networks ")
	}
	return resp.Routers, nil
}

func (c *Client) FindRouter(routerId string) (*Router, error) {
	var resp struct {
		Router Router `json:"router"`
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	url := fmt.Sprintf("%s/%s.json", apiRouters, routerId)
	err := c.client.SendRequest(client.GET, "network", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get router details")
	}
	return &resp.Router, nil
}

func (c *Client) DeleteRouter(routerId string) error {
	url := fmt.Sprintf("%s/%s.json", apiRouters, routerId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.DELETE, "network", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to delete router %s ", routerId)
	}
	return err
}

func (c *Client) AddRouterToSubnet(routerId string, subnetId string) (string, error) {
	type typerouter struct {
		SubnetId string `json:"subnet_id"`
		PortId   string `json:"port_id"`
	}
	var req, resp typerouter
	req.SubnetId = subnetId

	url := fmt.Sprintf("%s/%s/add_router_interface", apiRouters, routerId)
	requestData := &goosehttp.RequestData{ReqValue: req, RespValue: &resp,
		ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.PUT, "network", url, requestData)
	if err != nil {
		err = errors.Newf(err, "failed to attach router %s to subnet %s", routerId, subnetId)
		return "", err
	}
	return resp.PortId, nil
}

func (c *Client) DetachRouterFromSubnet(routerId string, subnetId string) ([]string, error) {
	type typerouter struct {
		SubnetId string `json:"subnet_id"`
	}
	var req typerouter
	req.SubnetId = subnetId

	var resp []string

	url := fmt.Sprintf("%s/%s/remove_router_interface", apiRouters, routerId)
	requestData := &goosehttp.RequestData{ReqValue: req, RespValue: &resp,
		ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.PUT, "network", url, requestData)
	if err != nil {
		err = errors.Newf(err, "failed to attach router %s to subnet %s", routerId, subnetId)
		return nil, err
	}
	return resp, nil
}

func (c *Client) DeleteFloatingIP(floatingipId string) error {
	requestData := &goosehttp.RequestData{ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.DELETE, "network", apiFloatingIPs+"/"+floatingipId, requestData)
	if err != nil {
		return errors.Newf(err, "failed to delete floatingip")
	}
	return nil
}

func (c *Client) AllocateFloatingIP(floatingip *FloatingIP) (*FloatingIP, error) {
	type typefloatingip struct {
		FloatingIP *FloatingIP `json:"floatingip"`
	}
	var req, resp typefloatingip
	req.FloatingIP = floatingip
	requestData := &goosehttp.RequestData{ReqValue: req, RespValue: &resp,
		ExpectedStatus: []int{http.StatusOK, http.StatusCreated}}
	err := c.client.SendRequest(client.POST, "network", apiFloatingIPs, requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to allocate floatingip")
	}
	return resp.FloatingIP, nil
}

func (c *Client) AssociateFloatingIP(floatingipId string, floatingip *FloatingIP) (*FloatingIP, error) {
	type typefloatingip struct {
		FloatingIP *FloatingIP `json:"floatingip"`
	}
	var req, resp typefloatingip
	req.FloatingIP = floatingip
	requestData := &goosehttp.RequestData{ReqValue: req, RespValue: &resp,
		ExpectedStatus: []int{http.StatusOK}}
	url := fmt.Sprintf("%s/%s", apiFloatingIPs, floatingipId)
	err := c.client.SendRequest(client.PUT, "network", url, requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to associate floatingip")
	}
	return resp.FloatingIP, nil
}

func (c *Client) DisAssociateFloatingIP(floatingipId string, floatingip *FloatingIP) (*FloatingIP, error) {
	type typefloatingip struct {
		FloatingIP *FloatingIP `json:"floatingip"`
	}
	var req, resp typefloatingip
	req.FloatingIP = floatingip
	requestData := &goosehttp.RequestData{ReqValue: req, RespValue: &resp,
		ExpectedStatus: []int{http.StatusOK}}
	url := fmt.Sprintf("%s/%s.json", apiFloatingIPs, floatingipId)
	err := c.client.SendRequest(client.PUT, "network", url, requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to disassociate floatingip")
	}
	return resp.FloatingIP, nil
}

func (c *Client) ListFloatingIPs(params *url.Values) ([]FloatingIP, error) {
	var resp struct {
		FloatingIPs []FloatingIP `json:"floatingips"`
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	if params != nil {
		requestData.Params = params
	}
	err := c.client.SendRequest(client.GET, "network", apiFloatingIPs, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of floatingips ")
	}
	return resp.FloatingIPs, nil
}

// Port API
func (c *Client) CreatePort(network *Port) (*Port, error) {
	type typenetwork struct {
		Port *Port `json:"port"`
	}
	var req, resp typenetwork
	req.Port = network
	requestData := &goosehttp.RequestData{ReqValue: req, RespValue: &resp,
		ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.POST, "network", apiPorts, requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to create port")
	}
	return resp.Port, nil
}

func (c *Client) ListPorts(filter *url.Values) ([]Port, error) {
	var resp struct {
		Ports []Port `json:"ports"`
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	if nil != filter {
		requestData.Params = filter
	}
	err := c.client.SendRequest(client.GET, "network", apiPorts, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of ports ")
	}
	return resp.Ports, nil
}
