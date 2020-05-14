package driver

import (
	"fmt"
	"strings"

	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

//S3fsDriver is a volume driver over s3fs
type S3fsDriver struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Region    string
	RootMount string
	Defaults  map[string]interface{}
}

//NewDriver creates a new S3FS driver
func NewDriver(c *cli.Context) (*S3fsDriver, error) {
	endpoint := c.String("endpoint")
	accesskey := c.String("accesskey")
	secretkey := c.String("secretkey")
	region := c.String("region")
	mount := c.String("mount")
	defaults := make(map[string]interface{})
	options := strings.Split(c.String("defaults"), ",")
	for _, o := range options {
		if !strings.Contains(o, "=") {
			defaults[o] = true
			continue
		}
		infos := strings.SplitN(o, "=", 1)
		if len(infos) != 2 {
			log.WithField("command", "driver").Errorf("could not parse default options: %s", o)
			return nil, fmt.Errorf("could not parse default options: %s", o)
		}
		defaults[infos[0]] = infos[1]
	}
	driver := &S3fsDriver{
		Endpoint:  endpoint,
		AccessKey: accesskey,
		SecretKey: secretkey,
		Region:    region,
		RootMount: mount,
		Defaults:  defaults,
	}
	log.WithField("command", "driver").Infof("endpoint: %s", endpoint)
	log.WithField("command", "driver").Infof("accesskey: %s", accesskey)
	log.WithField("command", "driver").Infof("region: %s", region)
	log.WithField("command", "driver").Infof("mount: %s", mount)
	log.WithField("command", "driver").Infof("default options: %+v", defaults)
	return driver, nil
}

//Create creates a volume
func (d *S3fsDriver) Create(req *volume.CreateRequest) error {
	return fmt.Errorf("not implemented")
}

//List lists volumes
func (d *S3fsDriver) List() (*volume.ListResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

//Get gets a volume
func (d *S3fsDriver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

//Remove removes a volume
func (d *S3fsDriver) Remove(req *volume.RemoveRequest) error {
	return fmt.Errorf("not implemented")
}

//Path provides the path
func (d *S3fsDriver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

//Mount mounts a volume
func (d *S3fsDriver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

//Unmount unmounts a volume
func (d *S3fsDriver) Unmount(req *volume.UnmountRequest) error {
	return fmt.Errorf("not implemented")
}

//Capabilities returns capabilities
func (d *S3fsDriver) Capabilities() *volume.CapabilitiesResponse {
	return nil
}
