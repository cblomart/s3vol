package main

import (
	"os"

	"github.com/cblomart/s3vol/serve"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "s3vol",
		Usage: "s3fs docker volume plugin",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "socket",
				Aliases: []string{"s"},
				Value:   "/run/docker/plugins/s3vol.sock",
				EnvVars: []string{"S3VOL_SOCKET"},
				Usage:   "plugin socket",
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Value:   false,
				EnvVars: []string{"S3VOL_DEBUG"},
				Usage:   "debug logging",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "serve",
				Aliases: []string{"s"},
				Usage:   "start s3vol server",
				Action:  serve.Serve,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "endpoint",
						Aliases: []string{"e"},
						Value:   "http://localhost:9000",
						EnvVars: []string{"S3VOL_ENDPOINT"},
						Usage:   "s3 endpoint",
					},
					&cli.StringFlag{
						Name:     "accesskey",
						Aliases:  []string{"k"},
						Required: true,
						EnvVars:  []string{"S3VOL_ACCESSKEY"},
						Usage:    "s3 accesskey",
					},
					&cli.StringFlag{
						Name:     "secretkey",
						Aliases:  []string{"s"},
						Required: true,
						EnvVars:  []string{"S3VOL_SECRETKEY"},
						Usage:    "s3 accesskey",
					},
					&cli.StringFlag{
						Name:    "region",
						Aliases: []string{"r"},
						Value:   "us-east-1",
						EnvVars: []string{"S3VOL_REGION"},
						Usage:   "s3 region",
					},
					&cli.StringFlag{
						Name:    "mount",
						Aliases: []string{"m"},
						Value:   "/mnt",
						EnvVars: []string{"S3FS_ROOT"},
						Usage:   "s3fs mount root",
					},
					&cli.StringFlag{
						Name:    "defaults",
						Aliases: []string{"o", "opts"},
						Value:   "",
						EnvVars: []string{"S3VOL_DEFAULTS"},
						Usage:   "s3fs default options",
					},
				},
			},
			{
				Name:    "volume",
				Aliases: []string{"v"},
				Usage:   "volume actions",
				Subcommands: []*cli.Command{
					{
						Name:    "list",
						Aliases: []string{"l"},
						Usage:   "list volumes",
					},
				},
			},
		},
	}
	app.Run(os.Args)
}
