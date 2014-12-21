package provision

import (
	"fmt"
	log "github.com/cihub/seelog"
	"launchpad.net/goose/client"
	"launchpad.net/goose/glance"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/neutron"
	"launchpad.net/goose/nova"
	"net"
	persistence "stormstack.org/stormio/persistence"
	"stormstack.org/stormio/stormstack"
	"stormstack.org/stormio/util"
	"sync"
	"time"
)

type FloatingIPService interface {
	CheckAvailability() (count int, err error)
	Attach(serverId string) (ip string, err error)
	Dettach(floatingIp string) (err error)
	Retain(serverId, fip string) (ip string, err error)
	Track(fip string)
}

type ServiceProvision struct {
	nova        *nova.Client
	glance      *glance.Client
	neutron     *neutron.Client
	floatingSvc FloatingIPService
}

const (
	ErrorServerCreate = iota
	ErrorAssociateIP
	ErrorSettingHostName
	ErrorFindFlavor
	ErrorFindImage
	ErrorServerDetail
	ErrorStormRegister
)

type RemediationList struct {
	sync.RWMutex
	remediationList map[string]string
}

type FIPWithNova struct {
	nova *nova.Client
	*RemediationList
}

type FIPWithNeutron struct {
	neutron *neutron.Client
	*RemediationList
}

type ProvisionError struct {
	Code int   //error code
	Err  error //description of error
}

func (pe *ProvisionError) Error() string {
	return fmt.Sprintf("Provision: Server not created %v", pe.Err)
}

func NewServiceProvision(provider *persistence.AssetProvider) *ServiceProvision {
	creds := &identity.Credentials{URL: provider.EndPointURL,
		User:       provider.Username,
		Secrets:    provider.Password,
		Region:     provider.RegionName,
		TenantName: provider.Tenant}
	client := client.NewClient(creds, identity.AuthUserPass, nil, overrideServiceURLs(provider))
	nova := nova.New(client)
	glance := glance.New(client)
	neutron := neutron.New(client)
	//check network capabilities
	svp := &ServiceProvision{nova: nova, glance: glance, neutron: neutron}
	rmdtrk := &RemediationList{remediationList: make(map[string]string)}
	if networks, _ := neutron.ListNetworks(); len(networks) > 0 {
		svp.floatingSvc = &FIPWithNeutron{neutron, rmdtrk}
	} else {
		svp.floatingSvc = &FIPWithNova{nova, rmdtrk}
	}
	return svp
}

func overrideServiceURLs(provider *persistence.AssetProvider) identity.ServiceURLs {
	serviceURLs := make(identity.ServiceURLs)
	iflag, idflag, nflag, sflag, cflag := false, false, false, false, false
	if provider.Image != "" {
		serviceURLs["image"] = provider.Image
		iflag = true
	}
	if provider.Identity != "" {
		serviceURLs["identity"] = provider.Identity
		idflag = true
	}
	if provider.Neutron != "" {
		serviceURLs["network"] = provider.Neutron
		nflag = true
	}
	if provider.Storage != "" {
		serviceURLs["storage"] = provider.Storage
		sflag = true
	}
	if provider.Compute != "" {
		serviceURLs["compute"] = provider.Compute
		cflag = true
	}
	if (iflag && idflag && cflag) || (nflag || sflag) {
		return serviceURLs
	}
	return nil
}

func (svc *ServiceProvision) Ping() error {
	list, err := svc.nova.ListFlavors()
	if len(list) == 0 || err != nil {
		log.Errorf("Not a valid asset provider, Actual Error (%v)", err)
		return fmt.Errorf("Not a valid asset provider")
	}
	return nil
}

func (svc *ServiceProvision) ProvisionInstance(asset *persistence.AssetRequest) (entityId string, fip string, err error) {
	log.Debugf("[areq %s][res %s] Inside ProvisionInstance", asset.Id, asset.ResourceId)
	model := asset.Model

	metadata := make(map[string]string)
	metadata["signerId"] = util.GetString("meta-data", "signer-id")
	metadata["nexusUrl"] = util.GetString("meta-data", "nexus-url")
	// Set the metadata with token
	stormdata := stormstack.BuildStormData(asset)
	metadata["stormtracker"] = stormdata

	serverOpts := &nova.RunServerOpts{Name: asset.HostName, FlavorId: model.Flavor, ImageId: model.Image,
		MinCount: 1, MaxCount: 1, Metadata: metadata}
	log.Debugf("[areq %s][res %s] Creating the server with options %v", asset.Id, asset.ResourceId, serverOpts)
	entity, err := svc.createInstance(serverOpts)
	if err != nil {
		log.Errorf("[areq %s][res %s] Unable to create the server %v", asset.Id, asset.ResourceId, err)
		err = &ProvisionError{ErrorServerCreate, err}
		return
	}
	entityId = entity.Id

	delayedUnit := 0
	if delayedUnit, err = svc.waitServerToStart(entity.Id); err != nil {
		err = &ProvisionError{ErrorServerCreate, err}
		return
	}

	time.Sleep(time.Duration(guessDelay(delayedUnit)) * time.Second)
	if asset.Remediation {
		fip, err = svc.floatingSvc.Retain(entity.Id, asset.IpAddress)
	} else {
		fip, err = svc.floatingSvc.Attach(entity.Id)
	}
	if err != nil {
		err = &ProvisionError{ErrorAssociateIP, err}
		//return TODO: Debate, do we need to throw an error if fip allocation fails??
	}
	address, err := net.LookupAddr(fip)
	if err != nil && len(address) > 0 {
		for _, address := range address {
			log.Debugf("%s", address)
		}
		_, err = svc.nova.RenameServer(entity.Id, address[0])
		if err != nil {
			err = &ProvisionError{ErrorSettingHostName, err}
		}
	}

	serverDetail, err := svc.nova.GetServer(entity.Id)
	if err != nil {
		err = &ProvisionError{ErrorServerDetail, err}
		return

	}

	var addresses []nova.IPAddress = nil
	//for plain openstacks
	if len(serverDetail.Addresses["public"]) > 0 {
		addresses = serverDetail.Addresses["public"]
	} else if len(serverDetail.Addresses["private"]) > 0 {
		addresses = serverDetail.Addresses["private"]
	} else if len(serverDetail.Addresses) > 0 {
		//for HP Cloud
		for _, _addresses := range serverDetail.Addresses {
			addresses = _addresses
		}
	}

	if fip == "" {
		if len(addresses) > 1 {
			fip = addresses[1].Address
		} else if len(addresses) > 0 {
			fip = addresses[0].Address
		}
	}
	if fip == "" {
		err = &ProvisionError{ErrorAssociateIP, fmt.Errorf("Unable to allocate floating ip")}
		return
	}

	// Register a new agent with StormTracker
	log.Debugf("[areq %s][res %s] About to register with stormtracker", asset.Id, asset.ResourceId)
	if stormdata != "" {
		err = stormstack.RegisterStormAgent(asset, entityId)
		if err != nil {
			log.Debugf("[areq %s][res %s] Unable to register Storm Agent %v", asset.Id, asset.ResourceId, err)
			err = &ProvisionError{ErrorStormRegister, err}
		}
	}
	return
}

func guessDelay(delayedUnit int) int {
	delayTime := util.GetInt("module-option", "delay-between-os-calls")
	if delayedUnit == 1 {
		return delayTime
	}
	return delayedUnit + delayTime
}

func (svc *ServiceProvision) GetServer(name, serverId string) (*nova.Entity, error) {
	log.Debugf("Getting the server details %s", name)
	filter := nova.NewFilter()
	filter.Set("name", name)
	entities, _ := svc.nova.ListServers(filter)

	for _, entity := range entities {
		if entity.Id == serverId {
			return &entity, nil
		}
	}

	return nil, fmt.Errorf("%s not found", serverId)
}

func (svc *ServiceProvision) ListFlavorNames() (*util.Response, error) {
	flavorMap := make(util.Response)
	flavors, err := svc.nova.ListFlavors()
	if err != nil {
		return nil, err
	}
	for _, flavor := range flavors {
		flavorMap[flavor.Id] = flavor.Name
	}
	return &flavorMap, nil
}

func (svc *ServiceProvision) ListImageNames() (*util.Response, error) {
	imageMap := make(util.Response)
	images, err := svc.glance.ListImages()
	if err != nil {
		return nil, err
	}
	for _, image := range images {
		imageMap[image.Id] = image.Name
	}
	return &imageMap, nil
}

func (svc *ServiceProvision) findImageId(imageName string) (string, error) {
	images, err := svc.glance.ListImages()
	if err != nil {
		return "", err
	}
	var imageId string
	for _, image := range images {
		if image.Name == imageName {
			imageId = image.Id
			// image.
			break
		}
	}
	if imageId == "" {
		return "", fmt.Errorf("No such image %s", imageName)
	}
	return imageId, nil
}

func (svc *ServiceProvision) findFlavorId(flavorName string) (string, error) {
	flavors, err := svc.nova.ListFlavors()
	if err != nil {
		return "", err
	}
	var flavorId string
	for _, flavor := range flavors {
		if flavor.Name == flavorName {
			flavorId = flavor.Id
			break
		}
	}
	if flavorId == "" {
		return "", fmt.Errorf("No such flavor %s", flavorName)
	}
	return flavorId, nil
}

//Terminate this instance with this Asset request
func (svc *ServiceProvision) DeprovisionInstance(ar *persistence.AssetRequest) error {
	/*Also delete the floating IP, the IP will be release from the pool.
	If this is the remediation request, don't release back to the pool.
	*/
	if ar.Remediation {
		svc.floatingSvc.Dettach(ar.IpAddress)
		svc.floatingSvc.Track(ar.IpAddress)
	}
	err := svc.nova.DeleteServer(ar.ServerId)
	if err != nil {
		//log something
		log.Debugf("[areq %s][res %s] Failed to delete the server :%s , error is :%v", ar.Id, ar.ResourceId, ar.ServerId, err.Error())
	}
	return err
}

func (svc *ServiceProvision) RenameServer(serverId, newName string) (err error) {
	_, err = svc.nova.RenameServer(serverId, newName)
	return
}

func (fpno *FIPWithNova) Dettach(floatingIp string) error {
	filter := util.NewFilter()
	filter.Set("ip", floatingIp)
	if fips, err := fpno.nova.ListFloatingIPs(&filter.Params); err == nil && len(fips) > 1 {
		for _, fip := range fips {
			if fip.IP == floatingIp {
				log.Debugf("Deleting the floating ip :%s", fip.IP)
				return fpno.nova.DeleteFloatingIP(fip.Id)
			}
		}
	}
	return fmt.Errorf("Floating ip object not found with :%s", floatingIp)
}

func (fpno *FIPWithNova) Attach(serverId string) (string, error) {
	for i := 0; i < 2; i++ {
		//if allocation fails, get the list, release them and retry.
		if ip, err := fpno.nova.AllocateFloatingIP(); err != nil || ip == nil {
			log.Debugf("Failed to allocate fip ondemand,releasing the free ones and retrying...")
			log.Debug("Getting floatingIP list")
			if ips, err := fpno.nova.ListFloatingIPs(nil); err == nil {
				for _, ip := range ips {
					if fpno.Find(ip.IP) {
						//don't delete, there remediation requests going on
						continue
					}
					if ip.InstanceId == nil {
						fpno.nova.DeleteFloatingIP(ip.Id)
					}
				}
			}
			continue
		} else {
			log.Debugf("FloatingIP allocated :%s", ip.IP)
			err := fpno.nova.SetIPV4Address(serverId, ip.IP)
			if err != nil {
				log.Errorf("Fail to set accessIPv4 address, error:%v", err)
				fpno.nova.DeleteFloatingIP(ip.Id)
				return "", fmt.Errorf("Failed to attach access IP with Floating IP")
			}
			err = fpno.nova.AddServerFloatingIP(serverId, ip.IP)
			if err != nil {
				fpno.nova.DeleteFloatingIP(ip.Id)
				log.Errorf("Fail to attach floating ip, error:%v", err)
				return "", fmt.Errorf("Failed to attach Floating IP")
			}
			return ip.IP, nil
		}
	}
	return "", fmt.Errorf("No floating IPs found")
}

func (fpno *FIPWithNova) CheckAvailability() (count int, err error) {
	afip := util.GetInt("openstack", "maximum-fip")
	consumed := 0
	if ips, err := fpno.nova.ListFloatingIPs(nil); err == nil {
		for _, ip := range ips {
			if ip.InstanceId != nil {
				consumed++
			}
		}
	} else {
		return -1, err
	}

	return (afip - consumed), nil
}

func (fpno *FIPWithNova) Retain(serverId, ipAddress string) (ip string, err error) {
	//get the list, release them and retry.
	available := false
	log.Debug("This is a remediation request, trying to retain old ip \nGetting floatingIP list")
	if ips, err := fpno.nova.ListFloatingIPs(nil); err == nil {
		for _, ip := range ips {
			if ip.IP == ipAddress {
				available = true
				break
			}
		}
	}

	if available {
		log.Debugf("FloatingIP available :%s", ipAddress)
		err := fpno.nova.SetIPV4Address(serverId, ipAddress)
		if err = fpno.nova.AddServerFloatingIP(serverId, ipAddress); err != nil {
			log.Debugf("Attempt to retain on old ip[%s] with new vcg [%s] is failed, trying with new...", ipAddress, serverId)
			fpno.Delete(ipAddress)
			ip, err = fpno.Attach(serverId)
		}
	} else {
		log.Debugf("Old ip[%s] not found with new vcg [%s], trying with new...", ipAddress, serverId)
		fpno.Delete(ipAddress)
		ip, err = fpno.Attach(serverId)
	}
	return
}

func (fipne *FIPWithNeutron) Dettach(floatingIp string) error {
	filter := util.NewFilter()
	filter.Set("floating_ip_address", floatingIp)
	if fip, err := fipne.neutron.ListFloatingIPs(&filter.Params); err == nil && len(fip) == 1 {
		log.Debugf("Deleting the floating ip :%s", fip[0].Id)
		return fipne.neutron.DeleteFloatingIP(fip[0].Id)
	}
	return fmt.Errorf("Floating ip object not found with :%s", floatingIp)
}

func (fipne *FIPWithNeutron) CheckAvailability() (count int, err error) {
	return 0, nil
}

func (fipne *FIPWithNeutron) Attach(serverId string) (string, error) {
	var extNet string
	if routers, err := fipne.neutron.ListRouters(); err == nil {
		for _, router := range routers {
			if router.ExternalGatewayInfo != nil {
				extNet = router.ExternalGatewayInfo.NetworkId
				log.Debugf("External network identified %s", extNet)
				break
			}
		}
	}

	if extNet == "" {
		log.Errorf("Unable to find the external network")
		return "", fmt.Errorf("Unable to find the external network")
	}
	filter := util.NewFilter()
	filter.Set("device_id", serverId)

	for i := 0; i < 2; i++ {
		//find the port id
		var _port *neutron.Port
		if ports, err := fipne.neutron.ListPorts(&filter.Params); err == nil {
			for _, port := range ports {
				if port.DeviceId == serverId {
					_port = &port
					fip := &neutron.FloatingIP{FloatingNetworkId: extNet, PortId: port.Id}
					if fip, err := fipne.neutron.AllocateFloatingIP(fip); err == nil {
						return fip.FloatingIPAddress, nil
					}
					log.Errorf("Error allocating fip %v", err)
					log.Debugf("Unable to allocate the fip")
					break
				}
			}
		}

		if _port == nil {
			log.Errorf("No port found for the server %s", serverId)
			return "", fmt.Errorf("No port found for the server %s", serverId)
		}

		filter := util.NewFilter()
		filter.Set("port_id", "")
		log.Debugf("Floating IP allocation failed, trying to release the floatings from the pool")
		if fips, err := fipne.neutron.ListFloatingIPs(&filter.Params); err == nil {
			for _, fip := range fips {
				if fip.PortId == "" {
					log.Debugf("Releasing FIP:%v Port:%v", fip, _port)
					// _fip := &neutron.FloatingIP{PortId: _port.Id}
					// if rfip, err := fipne.neutron.AssociateFloatingIP(fip.Id, _fip); err == nil {
					//	return rfip.FloatingIPAddress, nil
					// }
					fipne.neutron.DeleteFloatingIP(fip.FloatingIPAddress)
				}
			}
		}
	}
	return "", fmt.Errorf("Failed to allocate fip")
}

func (fipne *FIPWithNeutron) Retain(serverId, ipAddress string) (ip string, err error) {
	filter := util.NewFilter()
	filter.Set("device_id", serverId)
	var _port *neutron.Port
	if ports, err := fipne.neutron.ListPorts(&filter.Params); err == nil {
		for _, port := range ports {
			if port.DeviceId == serverId {
				_port = &port
				break
			}
		}
	}
	filter = util.NewFilter()
	filter.Set("port_id", "")
	filter.Set("floating_ip_address", ipAddress)
	log.Debugf("Find the uuid for the ip [%s]", ipAddress)
	if fips, err := fipne.neutron.ListFloatingIPs(&filter.Params); err == nil && len(fips) > 0 {
		for _, fip := range fips {
			if fip.PortId == "" && fip.FloatingIPAddress == ipAddress {
				log.Debugf("Attaching old  FIP:%v to new Port:%v", fip, _port)
				_fip := &neutron.FloatingIP{PortId: _port.Id}
				if rfip, err := fipne.neutron.AssociateFloatingIP(fip.Id, _fip); err == nil {
					fipne.Delete(ipAddress)
					return rfip.FloatingIPAddress, nil
				}
			}
		}
	} else {
		log.Debugf("Ip[%s] is either not available or using some one.. allocating new...")
		fipne.Delete(ipAddress)
		return fipne.Attach(serverId)
	}

	return
}

func (svc *ServiceProvision) createInstance(serverOpts *nova.RunServerOpts) (instance *nova.Entity, err error) {
	instance, err = svc.nova.RunServer(*serverOpts)
	return
}

func (svc *ServiceProvision) CheckAvailability() (count int, err error) {
	return svc.floatingSvc.CheckAvailability()
}

func (svc *ServiceProvision) waitServerToStart(serverId string) (delayedUnit int, err error) {
	delayedUnit = 1
	// Wait until the  server is actually running
	log.Infof("waiting the server %s to start...", serverId)
	for {
		server, err := svc.nova.GetServer(serverId)
		if err != nil {
			log.Errorf("Unable to get server details %v", err)
			break
		}

		if server.Status == nova.StatusActive {
			err = nil
			break
		}

		if server.Status == nova.StatusError {
			break
		}
		// We dont' want to flood the connection while polling the server waiting for it to start.
		log.Debugf("server has status %s, waiting 10 seconds before polling again...", server.Status)
		time.Sleep(10 * time.Second)
		delayedUnit += 1
	}
	log.Info("started")
	return
}

func (rmdtrk *RemediationList) Track(fip string) {
	rmdtrk.Lock()
	defer rmdtrk.Unlock()
	rmdtrk.remediationList[fip] = fip
	return
}

func (rmdtrk *RemediationList) Delete(id string) {
	rmdtrk.Lock()
	defer rmdtrk.Unlock()
	delete(rmdtrk.remediationList, id)
	return
}

func (rmdtrk *RemediationList) Find(fip string) (ok bool) {
	_, ok = rmdtrk.remediationList[fip]
	return
}
