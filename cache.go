package gin_api_cache

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/janartist/gin-api-cache/store"
)

const (
	DefaultExpire           = time.Minute
	ForeverExpire           = time.Duration(-1)
	CachePrefix             = "url_data_cache:"
	ResponseCacheContextKey = "responseCache"
)
const (
	SuccessEnableMode Mode = iota + 1
	ALLEnableMode
	CustomEnableMode
)
const (
	CacheSource Source = iota + 1
	LocalSource
)

var (
	responseCacheNotFoundError = fmt.Errorf("responseCache notFound")
)

func New(c CacheStore, ac bool, mode Mode, fn func(*store.ResponseCache) bool) *CacheManager {
	return &CacheManager{c, &Group{}, ac, mode, fn}
}
func NewDefault(conf *store.RedisConf) *CacheManager {
	cache := New(store.NewRedisStoreDefault(conf), true, SuccessEnableMode, nil)
	return cache
}

func Key(key string) CeOpt {
	return func(c *Cache) { c.Key = key }
}
func KMap(kMap map[string]string) CeOpt {
	return func(c *Cache) {
		c.KMap = kMap
	}
}
func Ttl(ttl time.Duration) CeOpt {
	return func(c *Cache) { c.Ttl = ttl }
}
func Single(s bool) CeOpt {
	return func(c *Cache) { c.Single = s }
}
func Remove(ctx context.Context, m *CacheManager, key string) error {
	err := m.Store.Remove(ctx, CachePrefix+key)
	return err
}

// CacheFunc Cache Decorator
func CacheFunc(m *CacheManager, opts ...CeOpt) gin.HandlerFunc {
	ce := &Cache{Single: true}
	for _, opt := range opts {
		opt(ce)
	}
	if ce.Ttl == 0 {
		ce.Ttl = DefaultExpire
	}
	return func(ctx *gin.Context) {
		ce2 := *ce
		if ce2.Key == "" {
			ce2.Key = ctx.Request.URL.Path
		}
		if ce2.KMap != nil {
			ce2.Key = ce2.Key + makeMapSortToString(ce2.KMap)
		}
		cc := &CacheContext{ctx, m, ce2, ctx.Request.RequestURI}

		cache := store.ResponseCache{Header: make(map[string][]string)}
		//from cache
		if err := m.Store.Get(ctx, CachePrefix+cc.Cache.Key, cc.requestPath, &cache); err == nil {
			if m.AddCacheHeader {
				cache.AddCacheHeader(cc.requestPath, int8(CacheSource))
			}
			cache.Write(ctx.Writer)
			ctx.Abort()
			return
		}

		doFunc := func(c *gin.Context) (interface{}, error) {
			w := c.Writer
			// replace writer, add set context ResponseCacheContextKey
			c.Writer = newCacheWriter(c.Writer, cc)
			//此处Next()应执行业务逻辑,执行完后response应写入完毕,或c.Writer被篡改
			c.Next()
			//get context ResponseCacheContextKey read response
			cFromContext, ok := c.Get(ResponseCacheContextKey)
			if !ok {
				return nil, responseCacheNotFoundError
			}
			c.Writer = w
			responseCache := cFromContext.(*store.ResponseCache)
			if m.IsOK(responseCache) {
				if err := m.Store.Set(c, CachePrefix+cc.Cache.Key, cc.requestPath, responseCache, ce2.Ttl); err != nil {
					log.Printf("cache store set err:%v", err)
				}
			}
			if m.AddCacheHeader {
				responseCache.AddCacheHeader(cc.requestPath, int8(LocalSource))
			}
			return responseCache, nil
		}

		if !ce2.Single {
			// replace writer
			doFunc(ctx)
			return
		}
		isWrite := false
		val, err := m.group.Do(cc.requestPath, func() (interface{}, error) {
			isWrite = true
			//TODO add MutexLock
			return doFunc(ctx)
		})
		// from localCache
		if !isWrite && err == nil {
			v := val.(*store.ResponseCache)
			v.Write(ctx.Writer)
		}
		if err != nil {
			log.Printf("local cache get err:%v", err)
		}
		ctx.Abort()
		return
	}
}

type Mode int8
type Source int8

type CacheManager struct {
	Store                CacheStore
	group                *Group
	AddCacheHeader       bool
	Mode                 Mode
	CustomEnableModeFunc func(*store.ResponseCache) bool
}

// IsOK 是否OK需要缓存
func (m *CacheManager) IsOK(c *store.ResponseCache) bool {
	if m.Mode == ALLEnableMode {
		return true
	}
	if m.Mode == SuccessEnableMode {
		return c.Status < http.StatusMultipleChoices && c.Status >= http.StatusOK
	}
	if m.Mode == CustomEnableMode && m.CustomEnableModeFunc != nil {
		return m.CustomEnableModeFunc(c)
	}
	return false
}

type CacheStore interface {
	Get(context.Context, string, string, *store.ResponseCache) error
	Set(context.Context, string, string, *store.ResponseCache, time.Duration) error
	Remove(context.Context, string) error
}

// CeOpt is an application option.
type CeOpt func(*Cache)

type Cache struct {
	Key    string
	KMap   map[string]string
	Ttl    time.Duration
	Single bool
}

type CacheContext struct {
	*gin.Context
	*CacheManager
	Cache
	requestPath string
}

func (c *CacheContext) Remove() error {
	return Remove(c.Context, c.CacheManager, c.Key)
}

type cacheWriter struct {
	gin.ResponseWriter
	cc *CacheContext
}

//重写底层write方法，获取response存入上下文，其余不更改
func (w *cacheWriter) Write(data []byte) (int, error) {
	ret, err := w.ResponseWriter.Write(data)
	if err == nil {
		//cache response
		val := &store.ResponseCache{
			Status: w.Status(),
			Header: w.Header(),
			Data:   data,
		}
		w.cc.Context.Set(ResponseCacheContextKey, val)
	}
	return ret, err
}
func newCacheWriter(writer gin.ResponseWriter, cc *CacheContext) *cacheWriter {
	return &cacheWriter{writer, cc}
}

//func urlEscape(prefix string, u string) string {
//	key := url.QueryEscape(u)
//	if len(key) > 200 {
//		h := sha1.New()
//		_, err := io.WriteString(h, u)
//		if err != nil {
//			return ""
//		}
//		key = string(h.Sum(nil))
//	}
//	var buffer bytes.Buffer
//	buffer.WriteString(prefix)
//	buffer.WriteString(":")
//	buffer.WriteString(key)
//	return buffer.String()
//}

//map至有序字符串
func makeMapSortToString(m map[string]string) string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Sort(sort.StringSlice(keys))

	var buffer bytes.Buffer

	for _, k := range keys {
		v, _ := m[k]
		buffer.WriteString("&")
		buffer.WriteString(k)
		buffer.WriteString("=")
		buffer.WriteString(v)
	}
	return buffer.String()
}
