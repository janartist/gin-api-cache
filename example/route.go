package example

import (
	"github.com/gin-gonic/gin"
	apicache "github.com/janartist/api-cache"
	"github.com/janartist/api-cache/store"
	"net/http"
	"time"
)

func main() {
	r := gin.Default()
	_ = apicache.NewDefault(&store.RedisConf{
		Addr: "127.0.0.1:6379",
		Auth: "",
		DB:   0,
	})

	r.GET("/test-cache1", apicache.CacheFunc(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	r.GET("/test-cache2", apicache.CacheFunc(apicache.Key("test-cache2")), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	r.GET("/test-cache3", apicache.CacheFunc(apicache.Key("test-cache3"), apicache.Ttl(time.Second*10)), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	r.GET("/test-cache-clear", func(c *gin.Context) {

		_ = apicache.Remove("test-cache1")
		_ = apicache.Remove("test-cache2")
		_ = apicache.Remove("test-cache3")
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	r.Run()
}
