package utils

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ztrue/tracerr"
	"golang.org/x/exp/constraints"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var randLetter = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// RandString returns n length random string [a-zA-Z0-9]+
func RandString(n int) string {
	s := make([]byte, n)
	for i := range s {
		s[i] = randLetter[rand.Intn(len(randLetter))]
	}
	return string(s)
}

// MD5 returns md5 hash of str.
func MD5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
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
		return tracerr.Wrap(err)
	}
	req = req.WithContext(ctx)
	client := new(http.Client)
	finish := make(chan error, 1)
	go func() {
		resp, err := client.Do(req)
		if err != nil {
			finish <- tracerr.Wrap(err)
			return
		}
		defer resp.Body.Close()
		file, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			finish <- tracerr.Wrap(err)
			return
		}
		err = ioutil.WriteFile(path, file, 0755)
		finish <- tracerr.Wrap(err)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-finish:
		return err
	}
}

func MustMarshal(v any) string {
	ret, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(ret)
}

func Min[T constraints.Ordered](a T, b T) T {
	if a > b {
		return b
	}
	return a
}

func Max[T constraints.Ordered](a T, b T) T {
	if a > b {
		return a
	}
	return b
}

func CalcPage(page int, size int, len int) (int, int, bool) {
	min := Max(0, (page-1)*size)
	max := Min(len, page*size)
	min = Min(min, len)
	max = Max(max, 0)
	if min == len || max == 0 || min >= max {
		return -1, -1, false
	}
	return min, max, true
}
