package driver

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

func parseOptions(options string) (map[string]interface{}, error) {
	defaults := make(map[string]interface{})
	if len(options) == 0 {
		return defaults, nil
	}
	opts := strings.Split(options, ",")
	for _, o := range opts {
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
	return defaults, nil
}
