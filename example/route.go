package example

import (
	"fmt"
	"github.com/gin-gonic/gin"
	apicache "github.com/janartist/api-cache"
	"github.com/janartist/api-cache/store"
	"net/http"
	"time"
)

func route() *gin.Engine {
	r := gin.Default()
	_ = apicache.NewDefault(&store.RedisConf{
		Addr: "127.0.0.1:6379",
		Auth: "",
		DB:   0,
	})

	r.GET("/test", func(c *gin.Context) {
		time.Sleep(time.Second)
		fmt.Print("[/test-cache] DB select ...\n")
		c.String(200, "test")
	})
	r.GET("/test-cache-second", apicache.CacheFunc(apicache.Ttl(time.Second)), func(c *gin.Context) {
		time.Sleep(time.Second)
		fmt.Print("[/test-cache-second] DB select ...\n")
		c.String(200, "test-cache-second-res")
	})
	r.GET("/test-cache-second-single", apicache.CacheFunc(apicache.Ttl(time.Second), apicache.Single(true)), func(c *gin.Context) {
		time.Sleep(time.Second)
		fmt.Print("[/test-cache-second-single] DB select ...\n")
		c.String(200, "test-cache-second-single-res")
	})
	r.GET("/test-cache-clear", func(c *gin.Context) {
		_ = apicache.Remove("/test")
		_ = apicache.Remove("/test-cache-second")
		_ = apicache.Remove("/test-cache-second-single")
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	return r
}
func Run() {
	err := route().Run()
	if err != nil {
		return
	}
}
