package driver

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/minio/minio-go/v6"
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
			_, err := buf.WriteString(fmt.Sprintf("%s\n", scanner.Text()))
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
		_, err := buf.WriteString(fmt.Sprintf("%s\n", scanner.Text()))
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
