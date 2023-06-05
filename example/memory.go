package example

import (
	"github.com/gin-gonic/gin"
	apicache "github.com/janartist/gin-api-cache"
	"github.com/janartist/gin-api-cache/store"
)

func RunWithMemory() *gin.Engine {
	m := apicache.New(store.NewMemoryStore(), true, apicache.SuccessEnableMode, nil)
	return route(m)
}
