package api

import (
	"skynet/sn"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func APIGetPublicSetting(c *gin.Context, id uuid.UUID) (int, error) {
	ret := make(map[string]interface{})
	for _, v := range sn.DefaultSetting {
		if v.Public {
			ret[v.Name] = v.Value
		}
	}
	responseData(c, ret)
	return 0, nil
}
