~~## Gin middleware/handler to enable Cache

### 用于缓存http接口内容的gin高性能中间件

- 得益于简单singleflight解决缓存击穿问题，优于官方 [gin-contrib/cache](https://github.com/gin-contrib/cache)
- 支持redis（适合集群）及memory（适合单机）驱动
- 支持自定key(默认route) 及参数k(默认requestPath)
- 使用hash数据结构来解决同个接口的数据组批量维护的问题
- 使用过期时间加删除缓存策略（不覆盖）
- mode参数可选择缓存http状态码为2xx的回包


### 击穿逻辑

![](https://github.com/janartist/gin-api-cache/blob/main/cache.jpg)

### Quick start

```shell
go get github.com/janartist/api-cache
```

```go

package main

import (
    apicache "github.com/janartist/api-cache"
    "github.com/gin-gonic/gin"
    "net/http/httptest"
    "github.com/janartist/api-cache/store"
    "testing"
)
func main()  {
	m := apicache.NewDefault(&store.RedisConf{
		Addr: "127.0.0.1:6379",
		Auth: "",
		DB:   0,
	})
	route(m)
}
func route(m *apicache.CacheManager) {
	app := gin.Default()
	app.GET("/test-cache-second", apicache.CacheFunc(m, apicache.Ttl(time.Second), apicache.Single(true)), func(c *gin.Context) {
		time.Sleep(time.Second)
		fmt.Print("[/test-cache-second] DB select ...\n")
		c.String(200, "test-cache-second-res")
	})
	if err := app.Run(":8080"); err != nil {
		panic(err)
	}
}

```
