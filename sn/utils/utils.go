package utils

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var randLetter = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// GetIP returns real ip for gin
// BUG: https://github.com/gin-gonic/gin/issues/2697
func GetIP(c *gin.Context) string {
	if !viper.GetBool("proxy.enable") {
		return c.ClientIP()
	} else {
		return c.GetHeader(viper.GetString("proxy.header"))
	}
}

// RandString return n length random string [a-zA-Z0-9]+
func RandString(n int) string {
	s := make([]byte, n)
	for i := range s {
		s[i] = randLetter[rand.Intn(len(randLetter))]
	}
	return string(s)
}

// Restart restart skynet itself, never returns.
func Restart() {
	log.Warn("Restart triggered")
	syscall.Kill(os.Getpid(), syscall.SIGHUP)
}

// MD5 return md5 hash of str.
func MD5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// IntMin returns smaller one between a and b.
func IntMin(a int, b int) int {
	if a <= b {
		return a
	} else {
		return b
	}
}

// IntMin returns bigger one between a and b.
func IntMax(a int, b int) int {
	if a >= b {
		return a
	} else {
		return b
	}
}

// FileExist returns whether file in path exist.
func FileExist(path string) bool {
	var exist = true
	if _, err := os.Stat(path); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

// DownloadFile download url to path.
func DownloadFile(ctx context.Context, url string, path string) error {
	dir, _ := filepath.Split(path)
	os.MkdirAll(dir, 0755)
	if FileExist(path) {
		return nil
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	tr := &http.Transport{}
	client := &http.Client{
		Transport: tr,
	}
	finish := make(chan error, 1)
	go func() {
		resp, err := client.Do(req)
		if err != nil {
			finish <- err
			return
		}
		defer resp.Body.Close()
		file, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			finish <- err
			return
		}
		err = ioutil.WriteFile(path, file, 0755)
		finish <- err
	}()

	select {
	case <-ctx.Done():
		tr.CancelRequest(req)
		return ctx.Err()
	case err := <-finish:
		return err
	}
}
