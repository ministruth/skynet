package utils

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letter = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

func Restart() {
	log.Warn("Restart triggered")
	syscall.Kill(os.Getpid(), syscall.SIGHUP)
}

func MD5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func IntMin(a int, b int) int {
	if a <= b {
		return a
	} else {
		return b
	}
}

func IntMax(a int, b int) int {
	if a >= b {
		return a
	} else {
		return b
	}
}

func FileExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func DownloadTempFile(ctx context.Context, url string, path string, hash string) error {
	dir, _ := filepath.Split(path)
	os.MkdirAll(dir, 0755)
	if FileExist(path) {
		file, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		if hash != "" && fmt.Sprintf("%x", sha256.Sum256(file)) == hash {
			return nil
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	tr := &http.Transport{}
	client := &http.Client{Transport: tr}
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
