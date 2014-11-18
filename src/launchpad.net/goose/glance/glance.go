// goose/glance - Go package to interact with OpenStack Image Service (Glance) API.
// See http://docs.openstack.org/api/openstack-image-service/2.0/content/.

package glance

import (
	"fmt"
	log "github.com/cihub/seelog"
	"io"
	"io/ioutil"
	"launchpad.net/goose/client"
	"launchpad.net/goose/errors"
	goosehttp "launchpad.net/goose/http"
	"net/http"
	"os"
	"strings"
)

// API URL parts.
const (
	apiImages       = "images"
	apiImagesDetail = "images/detail"
)

// Client provides a means to access the OpenStack Image Service.
type Client struct {
	client client.Client
}

// New creates a new Client.
func New(client client.Client) *Client {
	return &Client{client}
}

// Link describes a link to an image in OpenStack.
type Link struct {
	Href string
	Rel  string
	Type string
}

// Image describes an OpenStack image.
type Image struct {
	Id    string
	Name  string
	Links []Link
}

// ListImages lists IDs, names, and links for available images.
func (c *Client) ListImages() ([]Image, error) {
	var resp struct {
		Images []Image
	}
	requestData := goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.GET, "compute", apiImages, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of images")
	}
	return resp.Images, nil
}

// ImageMetadata describes metadata of an image
type ImageMetadata struct {
	Architecture string
	State        string      `json:"image_state"`
	Location     string      `json:"image_location"`
	KernelId     interface{} `json:"kernel_id"`
	ProjectId    interface{} `json:"project_id"`
	RAMDiskId    interface{} `json:"ramdisk_id"`
	OwnerId      interface{} `json:"owner_id"`
}

// ImageDetail describes extended information about an image.
type ImageDetail struct {
	Id              string
	Name            string
	Created         string
	Updated         string
	Progress        int
	Status          string
	MinimumRAM      int `json:"minRam"`
	MinimumDisk     int `json:"minDisk"`
	Links           []Link
	Metadata        ImageMetadata
	Uri             string
	DiskFormat      string
	ContainerFormat string
	Size            int64
	Checksum        string
	CreatedAt       string
	UpdatedAt       string
	DeletedAt       string
	Deleted         bool
	IsPublic        bool
	IsProtected     bool
	Owner           string
}

// ListImageDetails lists all details for available images.
func (c *Client) ListImagesDetail() ([]ImageDetail, error) {
	var resp struct {
		Images []ImageDetail
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", apiImagesDetail, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of image details")
	}
	return resp.Images, nil
}

// GetImageDetail lists details of the specified image.
func (c *Client) GetImageDetail(imageId string) (*ImageDetail, error) {
	var resp struct {
		Image ImageDetail
	}
	url := fmt.Sprintf("%s/%s", apiImages, imageId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get details of imageId: %s", imageId)
	}
	return &resp.Image, nil
}

func tempBuffer(req *http.Request) (exported *os.File, err error) {
	out, err := ioutil.TempFile("", "cloudnode.image")
	defer out.Close()
	n, err := io.Copy(out, req.Body)
	log.Debugf("Downloaded image size :%d", n)
	err = nil
	exported, err = os.Open(out.Name())
	return
}

func (c *Client) UploadImage(req *http.Request) (*ImageDetail, error) {
	type imgResponse struct{
        Image ImageDetail `json:"image"`
    }
	imgDetails := &imgResponse{}
	headers := make(http.Header)
	for header, values := range req.Header {
		for _, value := range values {
			if strings.HasPrefix(strings.ToLower(header), "x-image-meta") {
				headers.Add(header, value)
			}
		}
	}
	rawImage, _ := tempBuffer(req)
	defer func() { rawImage.Close(); os.Remove(rawImage.Name()) }()
	reqData := goosehttp.RequestData{UnMarshalJson: true, Binary: true, ReqHeaders: headers, ReqReader: rawImage, RespValue: imgDetails, ExpectedStatus: []int{http.StatusCreated}}
	err := c.client.SendRequest(client.POST, "image", apiImages, &reqData)
	if err != nil {
        log.Errorf("Failed to upload: %v", err)
		return nil, errors.Newf(err, "Failed to upload the image")
	}
	log.Debugf("Image upload finished :%v", imgDetails)
	return &imgDetails.Image, nil
}
