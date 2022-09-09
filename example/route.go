package example

import (
	"fmt"
	"github.com/gin-gonic/gin"
	apicache "github.com/janartist/api-cache"
	"net/http"
	"time"
)

func route(m *apicache.CacheManager) *gin.Engine {
	r := gin.Default()
	r.GET("/test", func(c *gin.Context) {
		time.Sleep(time.Second)
		fmt.Print("[/test-cache] DB select ...\n")
		c.String(200, "test")
	})
	r.GET("/test-cache-second", apicache.CacheFunc(m, apicache.Ttl(time.Second)), func(c *gin.Context) {
		time.Sleep(time.Second)
		fmt.Print("[/test-cache-second] DB select ...\n")
		c.String(200, "test-cache-second-res")
	})
	r.GET("/test-cache-second-single", apicache.CacheFunc(m, apicache.Ttl(time.Second), apicache.Single(true)), func(c *gin.Context) {
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
