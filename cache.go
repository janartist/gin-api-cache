package gin_api_cache

import (
	"bytes"
	"crypto/sha1"
	"github.com/gin-gonic/gin"
	"github.com/janartist/gin-api-cache/store"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"time"
)

const (
	DefaultExpire           = time.Minute * 2
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
const (
	maxConcurrencyDefault = 100000
)

func New(c CacheStore, group *Group, ac bool, mode Mode, fn func(*store.ResponseCache) bool) *CacheManager {
	return &CacheManager{c, group, ac, mode, fn}
}
func NewDefault(conf *store.RedisConf) *CacheManager {
	cache := New(store.NewRedisStoreDefault(conf), &Group{}, true, SuccessEnableMode, nil)
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
func MaxConcurrency(maxConcurrency int64) CeOpt {
	return func(c *Cache) { c.C = make(chan struct{}, maxConcurrency) }
}
func Remove(m *CacheManager, key string) error {
	err := m.Store.Remove(CachePrefix + key)
	return err
}

// CacheFunc Cache Decorator
func CacheFunc(m *CacheManager, co ...CeOpt) gin.HandlerFunc {
	ce := &Cache{Single: true, C: make(chan struct{}, maxConcurrencyDefault)}
	for _, c := range co {
		c(ce)
	}
	if ce.Ttl == 0 {
		ce.Ttl = DefaultExpire
	}
	return func(c *gin.Context) {
		ce := &*ce
		ce.C <- struct{}{}
		defer func() {
			<-ce.C
		}()
		cache := &store.ResponseCache{}
		cache.Header = make(map[string][]string)
		if ce.Key == "" {
			ce.Key = c.Request.URL.Path
		}
		if ce.KMap != nil {
			ce.Key = ce.Key + makeMapSortToString(ce.KMap)
		}

		cc := &CacheContext{c, m, ce, c.Request.RequestURI}

		//from cache
		if err := m.Store.Get(CachePrefix+cc.Cache.Key, cc.requestPath, cache); err == nil {
			if m.AddCacheHeader {
				cache.AddCacheHeader(cc.requestPath, int8(CacheSource))
			}
			cache.Write(c.Writer)
			c.Abort()
			return
		}

		if !ce.Single {
			// replace writer
			c.Writer = newCacheWriter(c.Writer, cc)
			return
		}
		isWrite := false
		val, _ := m.group.Do(cc.requestPath, func() (interface{}, error) {
			doFunc := func() (interface{}, error) {
				isWrite = true
				// replace writer, add set context ResponseCacheContextKey
				c.Writer = newCacheWriter(c.Writer, cc)
				//此处Next()应执行业务逻辑,执行完后response应写入完毕
				c.Next()
				//get context ResponseCacheContextKey read response
				cFromContext, _ := c.Get(ResponseCacheContextKey)
				responseCache := cFromContext.(*store.ResponseCache)
				if m.IsOK(responseCache) {
					if err := m.Store.Set(CachePrefix+ce.Key, cc.requestPath, responseCache, ce.Ttl); err != nil {
						log.Printf("cache store set err:%v", err)
					}
				}
				if m.AddCacheHeader {
					responseCache.AddCacheHeader(cc.requestPath, int8(LocalSource))
				}
				return responseCache, nil
			}
			//TODO add MutexLock
			return doFunc()
		})
		//from localCache
		if !isWrite {
			v := *val.(*store.ResponseCache)
			v.Write(c.Writer)
		}
		c.Abort()
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
	Get(string, string, *store.ResponseCache) error
	Set(string, string, *store.ResponseCache, time.Duration) error
	Remove(string) error
}

// CeOpt is an application option.
type CeOpt func(*Cache)

type Cache struct {
	Key    string
	KMap   map[string]string
	Ttl    time.Duration
	Single bool
	C      chan struct{}
}

type CacheContext struct {
	*gin.Context
	*CacheManager
	*Cache
	requestPath string
}

func (c *CacheContext) Remove() error {
	return Remove(c.CacheManager, c.Key)
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
func urlEscape(prefix string, u string) string {
	key := url.QueryEscape(u)
	if len(key) > 200 {
		h := sha1.New()
		_, err := io.WriteString(h, u)
		if err != nil {
			return ""
		}
		key = string(h.Sum(nil))
	}
	var buffer bytes.Buffer
	buffer.WriteString(prefix)
	buffer.WriteString(":")
	buffer.WriteString(key)
	return buffer.String()
}

//map至有序字符串
func makeMapSortToString(m map[string]string) string {
	var keys []string
	for k, _ := range m {
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
