package driver

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"strings"
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
}

//VolConfig represents the configuration of a volume
type VolConfig struct {
	Name    string
	Bucket  string
	Options map[string]string
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
	configbucketname := c.String("configbucket")
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

func (d *S3fsDriver) createBucket(bucket string) error {
	ok, err := d.s3client.BucketExists(bucket)
	if err != nil {
		log.WithField("command", "driver").Errorf("could not check existance of bucket %s: %s", bucket, err)
		return fmt.Errorf("could not check existance of bucket %s: %s", bucket, err)
	}
	if !ok {
		// create bucket
		err = d.s3client.MakeBucket(bucket, d.Region)
		if err != nil {
			log.WithField("command", "driver").Errorf("could not create bucket %s: %s", bucket, err)
			return fmt.Errorf("could not create bucket %s: %s", bucket, err)
		}
	}
	return nil
}

func (d *S3fsDriver) getVolumesConfig() ([]*VolConfig, error) {
	// get the config object
	obj, err := d.s3client.GetObject(d.ConfigBucketName, configObject, minio.GetObjectOptions{})
	if err != nil {
		log.WithField("command", "driver").Errorf("could not get config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
		return nil, fmt.Errorf("could not get config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
	}
	volConfigs := make([]*VolConfig, 0)
	// read the object
	scanner := bufio.NewScanner(obj)
	for scanner.Scan() {
		// skip comments
		if strings.HasPrefix(scanner.Text(), "#") {
			continue
		}
		// slit ";" and 3 max (volumename;bucket;options)
		parts := strings.SplitN(scanner.Text(), ";", 3)
		if len(parts) != 3 {
			log.WithField("command", "driver").Warn("wrong line in config: %s", scanner.Text())
			continue
		}
		name := parts[0]
		bucket := parts[1]
		options, err := parseOptions(parts[2])
		if err != nil {
			log.WithField("command", "driver").Warn("wrong options in config for %s: %s", name, err)
			continue
		}
		volConfigs = append(volConfigs, &VolConfig{Name: name, Bucket: bucket, Options: options})
	}
	if err := scanner.Err(); err != nil {
		log.WithField("command", "driver").Errorf("could not read config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
		return nil, fmt.Errorf("could not read config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
	}
	return volConfigs, nil
}

func (d *S3fsDriver) getVolumeConfig(volumeName string) (*VolConfig, error) {
	// get the config object
	obj, err := d.s3client.GetObject(d.ConfigBucketName, configObject, minio.GetObjectOptions{})
	if err != nil {
		log.WithField("command", "driver").Errorf("could not get config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
		return nil, fmt.Errorf("could not get config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
	}
	var volConfig *VolConfig
	// read the object
	scanner := bufio.NewScanner(obj)
	for scanner.Scan() {
		// skip comments
		if strings.HasPrefix(scanner.Text(), "#") {
			continue
		}
		// check that ; is in line
		if !strings.Contains(scanner.Text(), ";") {
			log.WithField("command", "driver").Warn("wrong line in config: %s", scanner.Text())
			continue
		}
		// slit ";" and 3 max (volumename;bucket;options)
		parts := strings.SplitN(scanner.Text(), ";", 3)
		if len(parts) != 3 {
			log.WithField("command", "driver").Warn("wrong line in config: %s", scanner.Text())
			continue
		}
		name := parts[0]
		if name != volumeName {
			continue
		}
		bucket := parts[1]
		options, err := parseOptions(parts[2])
		if err != nil {
			log.WithField("command", "driver").Warn("wrong options in config for %s: %s", name, err)
			continue
		}
		volConfig = &VolConfig{Name: name, Bucket: bucket, Options: options}
		break
	}
	if err := scanner.Err(); err != nil {
		log.WithField("command", "driver").Errorf("could not read config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
		return nil, fmt.Errorf("could not read config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
	}
	if volConfig == nil {
		log.WithField("command", "driver").Errorf("could not find config for '%s': %s", volumeName, err)
		return nil, fmt.Errorf("could not find config for '%s': %s", volumeName, err)
	}
	return volConfig, nil
}

func (d *S3fsDriver) addVolumeConfig(volConfig *VolConfig) error {
	// create config line
	options := optionsToString(volConfig.Options)
	config := fmt.Sprintf("%s;%s;%s\n", volConfig.Name, volConfig.Bucket, options)
	// get volumes config
	vols, err := d.getVolumesConfig()
	if err != nil {
		log.WithField("command", "driver").Errorf("could not get volumes config: %s", err)
		return fmt.Errorf("could not get volumes config: %s", err)
	}
	for _, v := range vols {
		if v.Name != volConfig.Name {
			continue
		}
		opts := optionsToString(v.Options)
		if opts != options {
			log.WithField("command", "driver").Errorf("the same volume already exists with different options")
			return fmt.Errorf("the same volume already exists with different options")
		}
		return nil
	}
	// get the config object
	obj, err := d.s3client.GetObject(d.ConfigBucketName, configObject, minio.GetObjectOptions{})
	if err != nil {
		log.WithField("command", "driver").Errorf("could not get config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
		return fmt.Errorf("could not get config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
	}
	// read all config
	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(obj)
	if err != nil {
		log.WithField("command", "driver").Errorf("could not read config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
		return fmt.Errorf("could not read config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
	}
	_, err = buf.WriteString(config)
	if err != nil {
		log.WithField("command", "driver").Errorf("could not write config '%s': %s", configObject, err)
		return fmt.Errorf("could not write config '%s': %s", configObject, err)
	}
	reader := bytes.NewReader(buf.Bytes())
	_, err = d.s3client.PutObject(d.ConfigBucketName, configObject, reader, reader.Size(), minio.PutObjectOptions{})
	if err != nil {
		log.WithField("command", "driver").Errorf("could not write config '%s' to bucket '%s': %s", configObject, d.ConfigBucketName, err)
		return fmt.Errorf("could not write config '%s' to bucket '%s': %s", configObject, d.ConfigBucketName, err)
	}
	return nil
}

func (d *S3fsDriver) removeVolumeConfig(volumeName string) error {
	// get the config object
	obj, err := d.s3client.GetObject(d.ConfigBucketName, configObject, minio.GetObjectOptions{})
	if err != nil {
		log.WithField("command", "driver").Errorf("could not get config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
		return fmt.Errorf("could not get config '%s' from bucket '%s': %s", configObject, d.ConfigBucketName, err)
	}
	// read the object
	buf := bytes.Buffer{}
	scanner := bufio.NewScanner(obj)
	for scanner.Scan() {
		// skip comments
		if strings.HasPrefix(scanner.Text(), "#") {
			_, err := buf.Write(scanner.Bytes())
			if err != nil {
				log.WithField("command", "driver").Warn("wrong cannot write to buffer: %s", err)
			}
			continue
		}
		// check that ; is in line
		if !strings.Contains(scanner.Text(), ";") {
			log.WithField("command", "driver").Warn("wrong line in config: %s", scanner.Text())
			continue
		}
		if strings.HasPrefix(fmt.Sprintf("%s;", volumeName), scanner.Text()) {
			continue
		}
		_, err := buf.Write(scanner.Bytes())
		if err != nil {
			log.WithField("command", "driver").Warn("wrong cannot write to buffer: %s", err)
		}
	}
	// write the config to bucket
	reader := bytes.NewReader(buf.Bytes())
	_, err = d.s3client.PutObject(d.ConfigBucketName, configObject, reader, reader.Size(), minio.PutObjectOptions{})
	if err != nil {
		log.WithField("command", "driver").Errorf("could not write config '%s' to bucket '%s': %s", configObject, d.ConfigBucketName, err)
		return fmt.Errorf("could not write config '%s' to bucket '%s': %s", configObject, d.ConfigBucketName, err)
	}
	return nil
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
	return fmt.Errorf("not implemented")
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
		log.WithField("command", "driver").Errorf("could not get volume config for '%s': %s", req.Name, err)
		return nil, fmt.Errorf("could not get volume config for '%s': %s", req.Name, err)
	}
	// get bucket infos
	bucketInfos, err := d.s3client.ListBuckets()
	if err != nil {
		log.WithField("command", "driver").Errorf("could not get bucket infos: %s", err)
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
