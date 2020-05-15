package driver

import (
	"fmt"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
)

func parseOptions(options string) (map[string]string, error) {
	defaults := make(map[string]string)
	if len(options) == 0 {
		return defaults, nil
	}
	opts := strings.Split(options, ",")
	for _, o := range opts {
		if !strings.Contains(o, "=") {
			defaults[o] = "true"
			continue
		}
		infos := strings.SplitN(o, "=", 1)
		if len(infos) != 2 {
			log.WithField("command", "driver").Errorf("could not parse default options: %s", o)
			return nil, fmt.Errorf("could not parse default options: %s", o)
		}
		if strings.ToLower(infos[1]) == "false" {
			continue
		}
		defaults[infos[0]] = infos[1]
	}
	return defaults, nil
}

func optionsToString(options map[string]string) string {
	//gather keys
	var keys []string
	for k := range options {
		keys = append(keys, k)
	}
	// sort keys
	sort.Strings(keys)
	var strOption []string
	// add options in alphabetical order
	for _, k := range keys {
		if len(options[k]) == 0 || strings.ToLower(options[k]) == "true" {
			strOption = append(strOption, k)
			continue
		}
		if strings.ToLower(options[k]) == "false" {
			continue
		}
		strOption = append(strOption, fmt.Sprintf("%s=%s", k, options[k]))
	}
	return strings.Join(strOption, ",")
}
