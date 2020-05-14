package driver

import (
	"fmt"
	"net/url"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/minio/minio-go/v6"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

//S3fsDriver is a volume driver over s3fs
type S3fsDriver struct {
	Endpoint  string
	UseSSL    bool
	AccessKey string
	SecretKey string
	Region    string
	RootMount string
	Defaults  map[string]interface{}
}

//NewDriver creates a new S3FS driver
func NewDriver(c *cli.Context) (*S3fsDriver, error) {
	u, err := url.Parse(c.String("endpoint"))
	if err != nil {
		log.WithField("command", "driver").Errorf("could not parse endpoint: %s", err)
		return nil, fmt.Errorf("could not parse enpoint: %s", err)
	}
	endpoint := u.Host
	if u.Scheme != "https" && u.Scheme != "http" {
		log.WithField("command", "driver").Errorf("s3 scheme not http(s)")
		return nil, fmt.Errorf("s3 scheme not http(s)")
	}
	usessl := true
	if u.Scheme == "http" {
		usessl = false
	}
	accesskey := c.String("accesskey")
	secretkey := c.String("secretkey")
	region := c.String("region")
	mount := c.String("mount")
	defaults, err := parseOptions(c.String("defaults"))
	if err != nil {
		log.WithField("command", "driver").Errorf("could not parse options: %s", err)
		return nil, fmt.Errorf("could not parse options: %s", err)
	}
	driver := &S3fsDriver{
		Endpoint:  endpoint,
		UseSSL:    usessl,
		AccessKey: accesskey,
		SecretKey: secretkey,
		Region:    region,
		RootMount: mount,
		Defaults:  defaults,
	}
	log.WithField("command", "driver").Infof("endpoint: %s", endpoint)
	log.WithField("command", "driver").Infof("usessl: %v", usessl)
	log.WithField("command", "driver").Infof("accesskey: %s", accesskey)
	log.WithField("command", "driver").Infof("region: %s", region)
	log.WithField("command", "driver").Infof("mount: %s", mount)
	log.WithField("command", "driver").Infof("default options: %#v", defaults)
	return driver, nil
}

//Create creates a volume
func (d *S3fsDriver) Create(req *volume.CreateRequest) error {
	log.WithField("command", "driver").WithField("method", "create").Debugf("request: %#v", req)
	return fmt.Errorf("not implemented")
}

//List lists volumes
func (d *S3fsDriver) List() (*volume.ListResponse, error) {
	log.WithField("command", "driver").WithField("method", "list").Debugf("list")
	// get a s3 client
	clt, err := minio.New(d.Endpoint, d.AccessKey, d.SecretKey, d.UseSSL)
	if err != nil {
		log.WithField("command", "driver").WithField("method", "list").Errorf("cannot get s3 client: %s", err)
		return nil, fmt.Errorf("cannot get s3 client: %s", err)
	}
	// list buckets
	buckets, err := clt.ListBuckets()
	if err != nil {
		log.WithField("command", "driver").WithField("method", "list").Errorf("cannot list s3 bucketst: %s", err)
		return nil, fmt.Errorf("cannot list s3 buckets: %s", err)
	}
	for _, bucket := range buckets {
		log.WithField("command", "driver").WithField("method", "list").Debugf("available bucket: %s", bucket)
	}
	return nil, fmt.Errorf("not implemented")
}

//Get gets a volume
func (d *S3fsDriver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	log.WithField("command", "driver").WithField("method", "get").Debugf("request: %#v", req)
	return nil, fmt.Errorf("not implemented")
}

//Remove removes a volume
func (d *S3fsDriver) Remove(req *volume.RemoveRequest) error {
	log.WithField("command", "driver").WithField("method", "remove").Debugf("request: %#v", req)
	return fmt.Errorf("not implemented")
}

//Path provides the path
func (d *S3fsDriver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	log.WithField("command", "driver").WithField("method", "path").Debugf("request: %#v", req)
	return nil, fmt.Errorf("not implemented")
}

//Mount mounts a volume
func (d *S3fsDriver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	log.WithField("command", "driver").WithField("method", "mount").Debugf("request: %#v", req)
	return nil, fmt.Errorf("not implemented")
}

//Unmount unmounts a volume
func (d *S3fsDriver) Unmount(req *volume.UnmountRequest) error {
	log.WithField("command", "driver").WithField("method", "unmount").Debugf("request: %#v", req)
	return fmt.Errorf("not implemented")
}

//Capabilities returns capabilities
func (d *S3fsDriver) Capabilities() *volume.CapabilitiesResponse {
	log.WithField("command", "driver").WithField("method", "capabilities").Debugf("scope: global")
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "global"}}
}
