package example

import (
	"github.com/gin-gonic/gin"
	apicache "github.com/janartist/gin-api-cache"
	"github.com/janartist/gin-api-cache/store"
)

func RunWithRedis() *gin.Engine {
	m := apicache.NewDefault(&store.RedisConf{
		Addr: "127.0.0.1:6379",
		Auth: "",
		DB:   0,
	})
	return route(m)
}
