package example

import (
	"github.com/gin-gonic/gin"
	apicache "github.com/janartist/gin-api-cache"
	"github.com/janartist/gin-api-cache/store"
)

func RunWithRedis() *gin.Engine {
	m := apicache.NewDefault(&store.RedisConf{
		Addr: "10.1.2.7:6379",
		Auth: "",
		DB:   3,
	})
	return route(m)
}
