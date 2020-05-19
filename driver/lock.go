package driver

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/minio/minio-go/v6"
	log "github.com/sirupsen/logrus"
)

const (
	lockExt     = ".ext.lock"
	lockWait    = 50 * time.Millisecond
	lockTimeOut = 100
)

// Lock locks an object
func (d *S3fsDriver) Lock(bucket string, object string) error {
	log.WithField("object", "minio").WithField("mehtod", "lock").WithField("bucket", bucket).WithField("object", object).Debugf("locking object")
	lock := fmt.Sprintf("%s%s", object, lockExt)
	hostname, err := os.Hostname()
	if err != nil {
		log.WithField("object", "minio").WithField("mehtod", "lock").WithField("bucket", bucket).WithField("object", object).Errorf("could not get hostname: %s", err)
		return fmt.Errorf("could not get hostname: %s", err)
	}
	// loop while stat works - assume no stat means no file
	count := 0
	for {
		_, err = d.s3client.StatObject(bucket, lock, minio.StatObjectOptions{})
		if err != nil {
			log.WithField("object", "minio").WithField("mehtod", "lock").WithField("bucket", bucket).WithField("object", lock).Debugf("could not stat lock: %s", err)
			break
		}
		// lock does exist
		obj, err := d.s3client.GetObject(bucket, lock, minio.GetObjectOptions{})
		if err != nil {
			log.WithField("object", "minio").WithField("mehtod", "lock").WithField("bucket", bucket).WithField("object", lock).Errorf("could not get lock: %s", err)
			return fmt.Errorf("could not get lock: %s", err)
		}
		// read the object
		buf := bytes.Buffer{}
		_, err = buf.ReadFrom(obj)
		if err != nil {
			log.WithField("object", "minio").WithField("mehtod", "lock").WithField("bucket", bucket).WithField("object", lock).Errorf("could not read lock: %s", err)
			return fmt.Errorf("could not read lock: %s", err)
		}
		log.WithField("object", "minio").WithField("mehtod", "lock").WithField("bucket", bucket).WithField("object", lock).Infof("lock is held by %s, waiting 50ms", buf.String())
		// increase tried count
		count++
		if count > lockTimeOut {
			log.WithField("object", "minio").WithField("mehtod", "lock").WithField("bucket", bucket).WithField("object", lock).Errorf("lock didn't disapear for 5s")
			return fmt.Errorf("lock didn't disapear for 5s")
		}
		time.Sleep(lockWait)
	}
	reader := strings.NewReader(hostname)
	_, err = d.s3client.PutObject(bucket, lock, reader, reader.Size(), minio.PutObjectOptions{})
	if err != nil {
		log.WithField("object", "minio").WithField("mehtod", "lock").WithField("bucket", bucket).WithField("object", object).Errorf("could not put lock: %s", err)
		return fmt.Errorf("could not put lock: %s", err)
	}
	// obtained the lock
	log.WithField("object", "minio").WithField("mehtod", "lock").WithField("bucket", bucket).WithField("object", object).Infof("locked")
	return nil
}

// UnLock unlocks an object
func (d *S3fsDriver) UnLock(bucket string, object string) error {
	log.WithField("object", "minio").WithField("mehtod", "unlock").WithField("bucket", bucket).WithField("object", object).Debugf("unlocking object")
	lock := fmt.Sprintf("%s%s", object, lockExt)
	hostname, err := os.Hostname()
	if err != nil {
		log.WithField("object", "minio").WithField("mehtod", "unlock").WithField("bucket", bucket).WithField("object", object).Errorf("could not get hostname: %s", err)
		return fmt.Errorf("could not get hostname: %s", err)
	}
	// check existance of the lock
	_, err = d.s3client.StatObject(bucket, lock, minio.StatObjectOptions{})
	if err != nil {
		log.WithField("object", "minio").WithField("mehtod", "ulock").WithField("bucket", bucket).WithField("object", object).Warnf("could not stat lock: %s", err)
		return nil
	}
	// lock does exist
	obj, err := d.s3client.GetObject(bucket, lock, minio.GetObjectOptions{})
	if err != nil {
		log.WithField("object", "minio").WithField("mehtod", "unlock").WithField("bucket", bucket).WithField("object", lock).Errorf("could not get lock: %s", err)
		return fmt.Errorf("could not get lock: %s", err)
	}
	// read the object
	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(obj)
	if err != nil {
		log.WithField("object", "minio").WithField("mehtod", "unlock").WithField("bucket", bucket).WithField("object", lock).Errorf("could not read lock: %s", err)
		return fmt.Errorf("could not read lock: %s", err)
	}
	if hostname != buf.String() {
		log.WithField("object", "minio").WithField("mehtod", "unlock").WithField("bucket", bucket).WithField("object", lock).Errorf("lock not generated by this server")
		return fmt.Errorf("could not generated by this server")
	}
	// remove the lock
	err = d.s3client.RemoveObject(bucket, lock)
	if err != nil {
		log.WithField("object", "minio").WithField("mehtod", "unlock").WithField("bucket", bucket).WithField("object", lock).Errorf("could not remove lock: %s", err)
		return fmt.Errorf("could not remove lock: %s", err)
	}
	// unlocked
	log.WithField("object", "minio").WithField("mehtod", "unlock").WithField("bucket", bucket).WithField("object", object).Infof("unlocked")
	return nil
}
