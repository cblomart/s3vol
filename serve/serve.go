package serve

import (
	"github.com/cblomart/s3vol/driver"
	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Serve serves the requests from docker
func Serve(c *cli.Context) error {
	log.WithField("command", "serve").Infof("s3vol - docker volume driver for s3fs")
	log.WithField("command", "serve").Infof("socket: %s", c.String("socket"))
	log.WithField("command", "serve").Infof("endpoint: %s", c.String("endpoint"))
	log.WithField("command", "serve").Infof("accesskey: %s", c.String("accesskey"))
	log.WithField("command", "serve").Infof("region: %s", c.String("region"))
	log.WithField("command", "serve").Infof("default options: %s", c.String("defaults"))
	volDriver := &driver.S3fsDriver{}
	volHandler := volume.NewHandler(volDriver)
	log.WithField("command", "driver").Infof("listening on %s", c.String("socket"))
	log.Error(volHandler.ServeUnix(c.String("socket"), 0))
	return nil
}
