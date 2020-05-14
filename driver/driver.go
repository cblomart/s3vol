package driver

import (
	"fmt"

	"github.com/docker/go-plugins-helpers/volume"
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
