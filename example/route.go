package example

import (
	"fmt"
	"github.com/gin-gonic/gin"
	apicache "github.com/janartist/gin-api-cache"
	"net/http"
	"time"
)

func route(m *apicache.CacheManager) *gin.Engine {
	r := gin.Default()
	r.GET("/test", func(c *gin.Context) {
		time.Sleep(time.Second)
		fmt.Print("[/test-cache] DB select ...\n")
		c.String(200, "test-res")
	})
	r.GET("/test-cache-second", apicache.CacheFunc(m, apicache.Ttl(time.Minute), apicache.Single(false)), func(c *gin.Context) {
		time.Sleep(time.Second)
		fmt.Print("[/test-cache-second] DB select ...\n")
		c.String(200, "test-cache-second-res")
	})
	r.GET("/test-cache-second-single", apicache.CacheFunc(m, apicache.Ttl(time.Second*5), apicache.Single(true)), func(c *gin.Context) {
		time.Sleep(time.Second)
		fmt.Print("[/test-cache-second-single] DB select ...\n")
		c.String(200, "test-cache-second-single-res")
	})
	r.GET("/test-cache-clear", func(c *gin.Context) {
		_ = apicache.Remove(m, "/test")
		_ = apicache.Remove(m, "/test-cache-second")
		_ = apicache.Remove(m, "/test-cache-second-single")
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	return r
}
func Run(m *apicache.CacheManager) {
	err := route(m).Run()
	if err != nil {
		return
	}
}

//添加用户ID到key
func cacheFuncByUid(m *apicache.CacheManager, co ...apicache.CeOpt) gin.HandlerFunc {
	return func(context *gin.Context) {
		uid := context.MustGet("uid").(string)
		co = append(co, apicache.KMap(map[string]string{"uid": uid}))
		apicache.CacheFunc(m, co...)(context)
	}
}
