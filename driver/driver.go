package driver

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/minio/minio-go/v6"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	emptyVolume = `# s3vol configuration
# volumename;bucket;options
`
	configObject = "volumes"
	s3fspwdfile  = "/etc/passwd-s3fs"
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
	s3client           *minio.Client
	s3fspath           string
	mounts             map[string]int
	mountsLock         sync.Mutex
}

//VolConfig represents the configuration of a volume
type VolConfig struct {
	Name    string
	Bucket  string
	Options map[string]string
}

//NewDriver creates a new S3FS driver
func NewDriver(c *cli.Context) (*S3fsDriver, error) {
	s3fspath := c.String("s3fspath")
	if len(s3fspath) == 0 {
		path := os.Getenv("PATH")
		paths := strings.Split(path, ":")
		for _, p := range paths {
			log.WithField("command", "driver").Debugf("checking for s3fs in %s", p)
			info, err := os.Stat(fmt.Sprintf("%s/s3fs", p))
			if err != nil {
				log.WithField("command", "driver").Debugf("could not stat %s/s3fs: %s", p, err)
				continue
			}
			if info.IsDir() {
				log.WithField("command", "driver").Debugf("path %s/s3fs is a directory", p)
				continue
			}
			if !strings.Contains(info.Mode().String(), "x") {
				log.WithField("command", "driver").Debugf("file %s/s3fs is not executable (%s)", p, info.Mode().String())
				continue
			}
			log.WithField("command", "driver").Debugf("found s3fs path: %s/s3fs", p)
			s3fspath = fmt.Sprintf("%s/s3fs", p)
			break
		}
	}
	if len(s3fspath) == 0 {
		log.WithField("command", "driver").Errorf("could not get s3fs path: provide s3fs path or install it")
		return nil, fmt.Errorf("could not get s3fs path: provide s3fs path or install it")
	}
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
	configbucketname := c.String("configbucket")
	mount := c.String("mount")
	mount = strings.TrimRight(mount, "/")
	defaults, err := parseOptions(c.String("defaults"))
	if err != nil {
		log.WithField("command", "driver").Errorf("could not parse options: %s", err)
		return nil, fmt.Errorf("could not parse options: %s", err)
	}
	// save s3fs password
	err = ioutil.WriteFile(s3fspwdfile, []byte(fmt.Sprintf("%s:%s", accesskey, secretkey)), 0660)
	if err != nil {
		log.WithField("command", "driver").Errorf("could not write s3fs password file: %s", err)
		return nil, fmt.Errorf("could not write s3fs password file: %s", err)
	}
	// add connection info to default options
	defaults["url"] = u.String()
	defaults["endpoint"] = region
	// default use path request style for minio
	defaults["use_path_request_style"] = "true"
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
		s3fspath:           s3fspath,
		mounts:             make(map[string]int),
	}
	log.WithField("command", "driver").Infof("endpoint: %s", endpoint)
	log.WithField("command", "driver").Infof("use ssl: %v", usessl)
	log.WithField("command", "driver").Infof("access key: %s", accesskey)
	log.WithField("command", "driver").Infof("region: %s", region)
	log.WithField("command", "driver").Infof("replace underscores: %s", replaceunderscores)
	log.WithField("command", "driver").Infof("mount: %s", mount)
	log.WithField("command", "driver").Infof("config bucket: %s", configbucketname)
	log.WithField("command", "driver").Infof("default options: %+v", defaults)
	// get a s3 client
	clt, err := minio.NewWithRegion(endpoint, accesskey, secretkey, usessl, region)
	if err != nil {
		log.WithField("command", "driver").Errorf("cannot get s3 client: %s", err)
		return nil, fmt.Errorf("cannot get s3 client: %s", err)
	}
	driver.s3client = clt
	err = driver.createBucket(configbucketname)
	if err != nil {
		log.WithField("command", "driver").Errorf("could check bucket '%s': %s", configbucketname, err)
		return nil, fmt.Errorf("could not check bucket '%s': %s", configbucketname, err)
	}
	// check config object existance
	_, err = clt.StatObject(configbucketname, configObject, minio.StatObjectOptions{})
	if err != nil {
		// create an empty config object
		reader := strings.NewReader(emptyVolume)
		_, err := clt.PutObject(configbucketname, configObject, reader, reader.Size(), minio.PutObjectOptions{})
		if err != nil {
			log.WithField("command", "driver").Errorf("could not create config in %s: %s", configbucketname, err)
			return nil, fmt.Errorf("could not create config in %s: %s", configbucketname, err)
		}
	}
	// return the driver
	return driver, nil
}

//Create creates a volume
func (d *S3fsDriver) Create(req *volume.CreateRequest) error {
	log.WithField("command", "driver").WithField("method", "create").Debugf("request: %+v", req)
	// check bucket name
	bucket := req.Name
	if strings.Contains(bucket, "_") && d.ReplaceUnderscores {
		bucket = strings.ReplaceAll(bucket, "_", "-")
	}
	// check that the bucket exists
	err := d.createBucket(bucket)
	if err != nil {
		log.WithField("command", "driver").WithField("method", "create").Errorf("could check bucket '%s': %s", bucket, err)
		return fmt.Errorf("could check bucket '%s': %s", bucket, err)
	}
	volConf := VolConfig{
		Name:    req.Name,
		Bucket:  bucket,
		Options: req.Options,
	}
	// add volume to config
	err = d.addVolumeConfig(&volConf)
	if err != nil {
		log.WithField("command", "driver").WithField("method", "create").Errorf("could add volume config: %s", err)
		return fmt.Errorf("could add volume config: %s", err)
	}
	return nil
}

//List lists volumes
func (d *S3fsDriver) List() (*volume.ListResponse, error) {
	log.WithField("command", "driver").WithField("method", "list").Debugf("list")
	// get volumes config
	vols, err := d.getVolumesConfig()
	if err != nil {
		log.WithField("command", "driver").Errorf("could not get volumes config: %s", err)
		return nil, fmt.Errorf("could not get volumes config: %s", err)
	}
	// get bucket infos
	bucketInfos, err := d.s3client.ListBuckets()
	if err != nil {
		log.WithField("command", "driver").Errorf("could not get bucket infos: %s", err)
		return nil, fmt.Errorf("could not get bucket infos: %s", err)
	}
	resp := make([]*volume.Volume, len(vols))
	for i, v := range vols {
		// search for the bucket creation date
		creation := ""
		for _, b := range bucketInfos {
			if v.Bucket == b.Name {
				creation = b.CreationDate.UTC().Format(time.RFC3339)
				break
			}
		}
		resp[i] = &volume.Volume{
			Name:       v.Name,
			Mountpoint: fmt.Sprintf("%s/%s", d.RootMount, v.Name),
			CreatedAt:  creation,
		}
	}
	return &volume.ListResponse{Volumes: resp}, nil
}

//Get gets a volume
func (d *S3fsDriver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	log.WithField("command", "driver").WithField("method", "get").Debugf("request: %+v", req)
	vol, err := d.getVolumeConfig(req.Name)
	if err != nil {
		log.WithField("command", "driver").WithField("method", "get").Warnf("could not get volume config for '%s': %s", req.Name, err)
		return nil, fmt.Errorf("could not get volume config for '%s': %s", req.Name, err)
	}
	// get bucket infos
	bucketInfos, err := d.s3client.ListBuckets()
	if err != nil {
		log.WithField("command", "driver").WithField("method", "get").Errorf("could not get bucket infos: %s", err)
		return nil, fmt.Errorf("could not get bucket infos: %s", err)
	}
	// get creation date
	creation := ""
	for _, b := range bucketInfos {
		if vol.Bucket == b.Name {
			creation = b.CreationDate.UTC().Format(time.RFC3339)
			break
		}
	}
	return &volume.GetResponse{
		Volume: &volume.Volume{
			Name:       vol.Name,
			Mountpoint: fmt.Sprintf("%s/%s", d.RootMount, vol.Name),
			CreatedAt:  creation,
		},
	}, nil
}

//Remove removes a volume
func (d *S3fsDriver) Remove(req *volume.RemoveRequest) error {
	log.WithField("command", "driver").WithField("method", "remove").Debugf("request: %+v", req)
	// get volume config
	volConfig, err := d.getVolumeConfig(req.Name)
	if err != nil {
		log.WithField("command", "driver").WithField("method", "remove").Errorf("could not get vol infos: %s", err)
		return fmt.Errorf("could not get vol infos: %s", err)
	}
	// check bucket
	buckets, err := d.s3client.ListBuckets()
	if err != nil {
		log.WithField("command", "driver").WithField("method", "remove").Errorf("could not list buckets: %s", err)
		return fmt.Errorf("could not list buckets: %s", err)
	}
	for _, bucket := range buckets {
		if bucket.Name == volConfig.Bucket {
			log.WithField("command", "driver").WithField("method", "remove").Infof("removing bucket: %s", volConfig.Bucket)
			// empty bucket
			// channel of objects to remove
			objectsCh := make(chan string)
			// Send object names that are needed to be removed to objectsCh
			go func() {
				defer close(objectsCh)
				// List all objects from a bucket
				for object := range d.s3client.ListObjects(volConfig.Bucket, "", true, nil) {
					if object.Err != nil {
						log.WithField("command", "driver").WithField("method", "remove").Errorf("removing object from bucket '%s': %s", volConfig.Bucket, object.Err)
						break
					}
					objectsCh <- object.Key
				}
			}()
			// remove the obtained objects from channel
			for rErr := range d.s3client.RemoveObjects(volConfig.Bucket, objectsCh) {
				log.WithField("command", "driver").WithField("method", "remove").Errorf("error emptying bucket '%s': %s", volConfig.Bucket, rErr)
				// don't exist: try to remove the bucket anyway
				break
			}
			// remove bucket
			err = d.s3client.RemoveBucket(volConfig.Bucket)
			if err != nil {
				log.WithField("command", "driver").WithField("method", "remove").Errorf("could not remove bucket: %s", err)
				return fmt.Errorf("could not remove bucket: %s", err)
			}
			break
		}
	}
	// remove config
	log.WithField("command", "driver").WithField("method", "remove").Infof("removing config: %s", volConfig.Name)
	err = d.removeVolumeConfig(volConfig.Name)
	if err != nil {
		log.WithField("command", "driver").WithField("method", "remove").Errorf("could not remove volume config: %s", err)
		return fmt.Errorf("could not remove volume config: %s", err)
	}
	return nil
}

//Path provides the path
func (d *S3fsDriver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	log.WithField("command", "driver").WithField("method", "path").Debugf("request: %+v", req)
	// get volume config
	volConfig, err := d.getVolumeConfig(req.Name)
	if err != nil {
		log.WithField("command", "driver").WithField("method", "remove").Errorf("could not get vol infos: %s", err)
		return nil, fmt.Errorf("could not get vol infos: %s", err)
	}
	return &volume.PathResponse{Mountpoint: fmt.Sprintf("%s/%s", d.RootMount, volConfig.Name)}, nil
}

//Mount mounts a volume
func (d *S3fsDriver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	log.WithField("command", "driver").WithField("method", "mount").Debugf("request: %+v", req)
	// get volume configurtion
	// get volume config
	volConfig, err := d.getVolumeConfig(req.Name)
	if err != nil {
		log.WithField("command", "driver").WithField("method", "mount").Errorf("could not get vol infos: %s", err)
		return nil, fmt.Errorf("could not get vol infos: %s", err)
	}
	// generate mount path
	path := fmt.Sprintf("%s/%s", d.RootMount, volConfig.Name)
	// check if already mounted
	d.mountsLock.Lock()
	defer d.mountsLock.Unlock()
	if _, ok := d.mounts[volConfig.Name]; ok {
		d.mounts[volConfig.Name] = 0
	}
	if d.mounts[volConfig.Name] > 0 {
		d.mounts[volConfig.Name]++
		log.WithField("command", "driver").WithField("method", "mount").Infof("volume %s is used by %d containers", volConfig.Name, d.mounts[volConfig.Name])
		return &volume.MountResponse{Mountpoint: path}, nil
	}
	// merging driver options and volume options
	// volume options have precedence
	options := d.Defaults
	for k, v := range volConfig.Options {
		options[k] = v
	}
	// create path if not exists
	info, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		log.WithField("command", "driver").WithField("method", "mount").Errorf("could not get mount path %s: %s", path, err)
		return nil, fmt.Errorf("could not get mount path %s: %s", path, err)
	}
	if os.IsNotExist(err) {
		// create path
		err := os.Mkdir(path, 0770)
		if err != nil {
			log.WithField("command", "driver").WithField("method", "mount").Errorf("could not create mount path %s: %s", path, err)
			return nil, fmt.Errorf("could not create mount path %s: %s", path, err)
		}
	} else {
		if !info.IsDir() {
			log.WithField("command", "driver").WithField("method", "mount").Errorf("mount path %s is not a directory: %s", path, err)
			return nil, fmt.Errorf("mount path %s is not a directory: %s", path, err)
		}
	}
	// generate command
	cmd := fmt.Sprintf("%s %s %s -o %s", d.s3fspath, volConfig.Bucket, path, optionsToString(options))
	log.WithField("command", "driver").WithField("method", "mount").Infof("cmd: %s", cmd)
	err = exec.Command("sh", "-c", cmd).Run()
	if err != nil {
		log.WithField("command", "driver").WithField("method", "mount").Errorf("error executing the mount command: %s", err)
		return nil, fmt.Errorf("error executing the mount command: %s", err)
	}
	d.mounts[volConfig.Name]++
	log.WithField("command", "driver").WithField("method", "mount").Infof("volume %s is used by %d containers", volConfig.Name, d.mounts[volConfig.Name])
	return &volume.MountResponse{Mountpoint: path}, nil
}

//Unmount unmounts a volume
func (d *S3fsDriver) Unmount(req *volume.UnmountRequest) error {
	log.WithField("command", "driver").WithField("method", "unmount").Debugf("request: %+v", req)
	// get volume configurtion
	// get volume config
	volConfig, err := d.getVolumeConfig(req.Name)
	if err != nil {
		log.WithField("command", "driver").WithField("method", "unmount").Errorf("could not get vol infos: %s", err)
		return fmt.Errorf("could not get vol infos: %s", err)
	}
	// aquire mount lock
	d.mountsLock.Lock()
	defer d.mountsLock.Unlock()
	// check that volume was mounted at least once
	if _, ok := d.mounts[volConfig.Name]; !ok {
		log.WithField("command", "driver").WithField("method", "unmount").Errorf("could not find mount infos for %s", volConfig.Name)
		return fmt.Errorf("could not find mount infos for %s", volConfig.Name)
	}
	if d.mounts[volConfig.Name] <= 0 {
		log.WithField("command", "driver").WithField("method", "unmount").Errorf("volume %s is apparently not mouted", volConfig.Name)
		return fmt.Errorf("volume %s is apparently not mouted", volConfig.Name)
	}
	// check if other container still have this mounted
	if d.mounts[volConfig.Name] > 1 {
		d.mounts[volConfig.Name]--
		log.WithField("command", "driver").WithField("method", "unmount").Infof("volume %s is used by %d containers", volConfig.Name, d.mounts[volConfig.Name])
		return nil
	}
	// generate mount path
	path := fmt.Sprintf("%s/%s", d.RootMount, volConfig.Name)
	// unmount volume
	// generate command
	cmd := fmt.Sprintf("umount %s", path)
	log.WithField("command", "driver").WithField("method", "unmount").Infof("cmd: %s", cmd)
	err = exec.Command("sh", "-c", cmd).Run()
	if err != nil {
		log.WithField("command", "driver").WithField("method", "umount").Errorf("error executing the umount command: %s", err)
		return fmt.Errorf("error executing the umount command: %s", err)
	}
	d.mounts[volConfig.Name]--
	log.WithField("command", "driver").WithField("method", "unmount").Infof("volume %s is used by %d containers", volConfig.Name, d.mounts[volConfig.Name])
	return nil
}

//Capabilities returns capabilities
func (d *S3fsDriver) Capabilities() *volume.CapabilitiesResponse {
	log.WithField("command", "driver").WithField("method", "capabilities").Debugf("scope: global")
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "global"}}
}
