package serve

import (
	"os"

	"github.com/cblomart/s3vol/driver"
	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Serve serves the requests from docker
func Serve(c *cli.Context) error {
	log.WithField("command", "serve").Infof("s3vol - docker volume driver for s3fs")
	volDriver, err := driver.NewDriver(c)
	if err != nil {
		log.WithField("command", "serve").Errorf("cannot instantiate driver: %s", err)
		return err
	}
	volHandler := volume.NewHandler(volDriver)
	log.WithField("command", "serve").Infof("listening on %s", c.String("socket"))
	defer func() {
		err := os.Remove(c.String("socket"))
		if err != nil {
			log.WithField("command", "serve").Warnf("couldn't remove socket: %s", c.String("socket"))
		}
	}()
	log.Error(volHandler.ServeUnix(c.String("socket"), 0))
	return nil
}
