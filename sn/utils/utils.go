package utils

import (
	"crypto/md5"
	"encoding/hex"
	"math"
	"math/rand"
	"os"
	"skynet/sn"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
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

func SplitPage(n int, p int) int {
	if n == 0 {
		return 1
	}
	if n%p == 0 {
		return n / p
	}
	return int(n/p) + 1
}

func GetSplitPage(l int, p int, s int) (int, int) {
	if l <= 0 {
		return -1, -1
	}
	p = int(math.Min(float64(p), float64(SplitPage(l, s))))
	low := int(math.Max(float64(p-1), 0)) * s
	high := int(math.Min(float64(p*s), float64(l)))
	return low, high
}

func PreSplitFunc(c *gin.Context, item *sn.SNPageItem, length int, size int, sizeList []int) (int, int, bool) {
	item.Param["_page"] = c.DefaultQuery("page", "1")
	reqPage, err := strconv.Atoi(item.Param["_page"].(string))
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(500)
		return -1, -1, false
	}
	item.Param["_size"] = c.Query("size")
	if item.Param["_size"] != "" {
		reqSize, err := strconv.Atoi(item.Param["_size"].(string))
		if err != nil {
			log.Error(err)
			c.AbortWithStatus(500)
			return -1, -1, false
		}
		if ContainsInt(sizeList, reqSize) {
			size = reqSize
		}
	}
	item.Param["_size"] = size
	item.Param["_page"] = reqPage
	item.Param["_totpage"] = SplitPage(length, size)
	if reqPage < 1 {
		reqPage = 1
		item.Param["_page"] = 1
	}
	if reqPage > item.Param["_totpage"].(int) {
		reqPage = item.Param["_totpage"].(int)
		item.Param["_page"] = item.Param["_totpage"]
	}
	low, high := GetSplitPage(length, reqPage, size)
	return low, high, true
}

func ContainsInt(a []int, x int) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
