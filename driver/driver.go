package driver

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/minio/minio-go/v6"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

//S3fsDriver is a volume driver over s3fs
type S3fsDriver struct {
	Endpoint           string
	UseSSL             bool
	AccessKey          string
	SecretKey          string
	Region             string
	RootMount          string
	ReplaceUnderscores bool
	ConfigBucketName   string
	Defaults           map[string]string
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
	replaceunderscores := c.Bool("replaceunderscores")
	configbucketname := c.String("configbucketname")
	mount := c.String("mount")
	mount = strings.TrimLeft(mount, "/")
	defaults, err := parseOptions(c.String("defaults"))
	if err != nil {
		log.WithField("command", "driver").Errorf("could not parse options: %s", err)
		return nil, fmt.Errorf("could not parse options: %s", err)
	}
	driver := &S3fsDriver{
		Endpoint:           endpoint,
		UseSSL:             usessl,
		AccessKey:          accesskey,
		SecretKey:          secretkey,
		Region:             region,
		RootMount:          mount,
		ReplaceUnderscores: replaceunderscores,
		ConfigBucketName:   configbucketname,
		Defaults:           defaults,
	}
	log.WithField("command", "driver").Infof("endpoint: %s", endpoint)
	log.WithField("command", "driver").Infof("use ssl: %v", usessl)
	log.WithField("command", "driver").Infof("access key: %s", accesskey)
	log.WithField("command", "driver").Infof("region: %s", region)
	log.WithField("command", "driver").Infof("replace underscores: %s", replaceunderscores)
	log.WithField("command", "driver").Infof("mount: %s", mount)
	log.WithField("command", "driver").Infof("config bucket: %s", configbucketname)
	log.WithField("command", "driver").Infof("default options: %+v", defaults)
	// test connection to s3
	clt, err := driver.getClient()
	if err != nil {
		log.WithField("command", "driver").Errorf("could not connect to s3: %s", err)
		return nil, fmt.Errorf("could not connect to s3: %s", err)
	}
	ok, err := clt.BucketExists(configbucketname)
	if err != nil {
		log.WithField("command", "driver").Errorf("could not check existance of bucket %s: %s", configbucketname, err)
		return nil, fmt.Errorf("could not check existance of bucket %s: %s", configbucketname, err)
	}
	if ok {
		return driver, nil
	}
	err = clt.MakeBucket(configbucketname, region)
	if err != nil {
		log.WithField("command", "driver").Errorf("could not create bucket %s: %s", configbucketname, err)
		return nil, fmt.Errorf("could not create bucket %s: %s", configbucketname, err)
	}
	// check for config bucket
	return driver, nil
}

func (d *S3fsDriver) getClient() (*minio.Client, error) {
	// get a s3 client
	clt, err := minio.NewWithRegion(d.Endpoint, d.AccessKey, d.SecretKey, d.UseSSL, d.Region)
	if err != nil {
		log.WithField("command", "driver").Errorf("cannot get s3 client: %s", err)
		return nil, fmt.Errorf("cannot get s3 client: %s", err)
	}
	return clt, nil
}

//Create creates a volume
func (d *S3fsDriver) Create(req *volume.CreateRequest) error {
	log.WithField("command", "driver").WithField("method", "create").Debugf("request: %+v", req)
	/*
		// get a list of current volumes
		resp, err := d.List()
		if err != nil {
			log.WithField("command", "driver").WithField("method", "create").Errorf("cannot list volumes: %s", err)
			return fmt.Errorf("cannot list volumes: %s", err)
		}
			// search for corresponding volume
			var vol *volume.Volume
			for _, v := range resp.Volumes {
				if v.Name == req.Name {
					vol = v
					break
				}
			}
				if vol != nil {
					// got a volume with same name
					reqOpts := optionsToString(req.Options)
				}
				// get a s3 client
				clt, err := minio.New(d.Endpoint, d.AccessKey, d.SecretKey, d.UseSSL)
				if err != nil {
					log.WithField("command", "driver").WithField("method", "list").Errorf("cannot get s3 client: %s", err)
					return fmt.Errorf("cannot get s3 client: %s", err)
				}
	*/
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
	// prepare a list of volumes
	vols := make([]*volume.Volume, len(buckets))
	for i, bucket := range buckets {
		log.WithField("command", "driver").WithField("method", "list").Debugf("available bucket: %s", bucket.Name)
		vols[i] = &volume.Volume{
			Name:       bucket.Name,
			Mountpoint: fmt.Sprintf("%s/%s", d.RootMount, bucket.Name),
			CreatedAt:  bucket.CreationDate.UTC().Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	// send response
	resp := &volume.ListResponse{
		Volumes: vols,
	}
	log.WithField("command", "driver").WithField("method", "list").Debugf("resp: %+v", resp)
	return resp, nil
}

//Get gets a volume
func (d *S3fsDriver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	log.WithField("command", "driver").WithField("method", "get").Debugf("request: %+v", req)
	return nil, fmt.Errorf("not implemented")
}

//Remove removes a volume
func (d *S3fsDriver) Remove(req *volume.RemoveRequest) error {
	log.WithField("command", "driver").WithField("method", "remove").Debugf("request: %+v", req)
	return fmt.Errorf("not implemented")
}

//Path provides the path
func (d *S3fsDriver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	log.WithField("command", "driver").WithField("method", "path").Debugf("request: %+v", req)
	return nil, fmt.Errorf("not implemented")
}

//Mount mounts a volume
func (d *S3fsDriver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	log.WithField("command", "driver").WithField("method", "mount").Debugf("request: %+v", req)
	return nil, fmt.Errorf("not implemented")
}

//Unmount unmounts a volume
func (d *S3fsDriver) Unmount(req *volume.UnmountRequest) error {
	log.WithField("command", "driver").WithField("method", "unmount").Debugf("request: %+v", req)
	return fmt.Errorf("not implemented")
}

//Capabilities returns capabilities
func (d *S3fsDriver) Capabilities() *volume.CapabilitiesResponse {
	log.WithField("command", "driver").WithField("method", "capabilities").Debugf("scope: global")
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "global"}}
}
