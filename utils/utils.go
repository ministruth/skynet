package utils

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
	"golang.org/x/exp/constraints"
)

var randLetter = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// RandString returns n length random string [a-zA-Z0-9]+
func RandString(n int) string {
	s := make([]byte, n)
	for i := range s {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(randLetter))))
		if err != nil {
			panic(err)
		}
		s[i] = randLetter[num.Int64()]
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

// DownloadFile downloads url to path.
// If override is set, override existing file, otherwise do nothing.
func DownloadFile(ctx context.Context, url string, path string, override bool) error {
	if !override && FileExist(path) {
		return nil
	}
	dir, _ := filepath.Split(path)
	os.MkdirAll(dir, 0755)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return tracerr.Wrap(err)
	}
	req = req.WithContext(ctx)
	client := new(http.Client)
	rsp, err := client.Do(req)
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer rsp.Body.Close()
	buf, err := io.ReadAll(rsp.Body)
	if err != nil {
		return tracerr.Wrap(err)
	}
	return os.WriteFile(path, buf, 0755)
}

// MustMarshal marshals v to string, panic if error.
func MustMarshal(v any) string {
	ret, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(ret)
}

// Max returns the smaller one in a and b.
func Min[T constraints.Ordered](a T, b T) T {
	if a > b {
		return b
	}
	return a
}

// Max returns the bigger one in a and b.
func Max[T constraints.Ordered](a T, b T) T {
	if a > b {
		return a
	}
	return b
}

// SliceUnique deduplicates slice.
func SliceUnique[T comparable](s []T) (ret []T) {
	keys := make(map[T]bool)
	for _, item := range s {
		if _, value := keys[item]; !value {
			keys[item] = true
			ret = append(ret, item)
		}
	}
	return
}

// MapKeyToSlice converts map key to slice.
func MapKeyToSlice[K comparable, V any](s map[K]V) (ret []K) {
	for k := range s {
		ret = append(ret, k)
	}
	return
}

// MapValueToSlice converts map value to slice.
func MapValueToSlice[K comparable, V any](s map[K]V) (ret []V) {
	for _, v := range s {
		ret = append(ret, v)
	}
	return
}

// SliceToMap converts slice to map.
func SliceToMap[V comparable](s []V) (ret map[V]bool) {
	ret = make(map[V]bool)
	for _, v := range s {
		ret[v] = true
	}
	return
}

// MapContains checks whether map contains a key.
func MapContains[K comparable, V any](m map[K]V, v K) bool {
	_, ok := m[v]
	return ok
}

// ParseUUIDSlice parses string array to uuid array.
func ParseUUIDSlice(uid []string) ([]uuid.UUID, error) {
	var ret []uuid.UUID
	for _, id := range uid {
		tmp, err := uuid.Parse(id)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		ret = append(ret, tmp)
	}
	return ret, nil
}

// SliceRemove removes slice from another slice.
func SliceRemove[T comparable](base []T, remove []T) []T {
	keys := SliceToMap(base)
	for _, v := range remove {
		delete(keys, v)
	}
	return MapKeyToSlice(keys)
}

func SlicePagination[T any](s []T, page int, size int) []T {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 1
	}
	start := (page - 1) * size
	end := Min(page*size, len(s))
	if start > end {
		return []T{}
	}
	return s[start:end]
}

func Unzip(buf []byte, dir string) error {
	if FileExist(dir) {
		return tracerr.New("folder already exists")
	}
	reader := bytes.NewReader(buf)
	r, err := zip.NewReader(reader, reader.Size())
	if err != nil {
		return tracerr.Wrap(err)
	}
	fc := func() error {
		if err := tracerr.Wrap(os.Mkdir(dir, 0755)); err != nil {
			return err
		}
		for _, f := range r.File {
			if f.FileInfo().IsDir() {
				if err := tracerr.Wrap(os.MkdirAll(filepath.Join(dir, f.Name), 0755)); err != nil {
					return err
				}
			} else {
				out, err := f.Open()
				if err != nil {
					return tracerr.Wrap(err)
				}
				defer out.Close()
				dst, err := os.OpenFile(filepath.Join(dir, f.Name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
				if err != nil {
					return tracerr.Wrap(err)
				}
				defer dst.Close()
				if _, err := io.Copy(dst, out); err != nil {
					return tracerr.Wrap(err)
				}
			}
		}
		return nil
	}

	if err := fc(); err != nil {
		os.RemoveAll(dir)
		return err
	}

	return nil
}
