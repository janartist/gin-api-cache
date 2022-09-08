package api_cache

import (
	"bytes"
	"crypto/sha1"
	"github.com/gin-gonic/gin"
	"github.com/janartist/api-cache/store"
	"io"
	"log"
	"net/url"
	"strconv"
	"time"
)

const (
	DEFAULT_EXPIRE = time.Minute * 2
	FOREVER_EXPIRE = time.Duration(-1)
	CACHE_PREFIX   = "url_data_cache:"
)

var CM *CacheManager

func New(c CacheStore, group *Group, eh bool) *CacheManager {
	cm := &CacheManager{c, group, eh}
	CM = cm
	return cm
}
func NewDefault(conf *store.RedisConf) *CacheManager {
	cache := New(store.NewRedisStoreDefault(conf), &Group{}, true)
	return cache
}

type CacheManager struct {
	Store        CacheStore
	group        *Group
	AddCacheInfo bool
}

type CacheStore interface {
	Get(string, string, *store.ResponseCache) error
	Set(string, string, *store.ResponseCache, time.Duration) error
	Remove(string) error
}

// CeOpt is an application option.
type CeOpt func(*Cache)

type Cache struct {
	Key string
	Ttl time.Duration
}

func Key(key string) CeOpt {
	return func(c *Cache) { c.Key = key }
}
func Ttl(ttl time.Duration) CeOpt {
	return func(c *Cache) { c.Ttl = ttl }
}

type CacheContext struct {
	*gin.Context
	*Cache
	requestPath string
}

func Remove(key string) error {
	err := CM.Store.Remove(CACHE_PREFIX + key)
	return err
}
func (c *CacheContext) Remove() error {
	return Remove(c.Key)
}

// CacheFunc Cache Decorator
func CacheFunc(co ...CeOpt) gin.HandlerFunc {
	ce := &Cache{}
	for _, c := range co {
		c(ce)
	}
	if ce.Ttl == 0 {
		ce.Ttl = DEFAULT_EXPIRE
	}
	return func(c *gin.Context) {
		var cache *store.ResponseCache
		if ce.Key == "" {
			ce.Key = c.Request.URL.Path
		}
		cc := &CacheContext{c, ce, urlEscape("", c.Request.RequestURI)}
		//from cache
		if err := CM.Store.Get(CACHE_PREFIX+ce.Key, cc.requestPath, cache); err == nil {
			if CM.AddCacheInfo {
				cache.Header.Set("Cache-Control", "max-age="+strconv.Itoa(int(cache.Ttl.Seconds()))+";must-revalidate")
			}
			cache.Write(c.Writer)
			c.Abort()
			return
		}
		val, _ := CM.group.Do(cc.requestPath, func() (interface{}, error) {
			// replace writer
			c.Writer = newCachedWriter(c.Writer, CM.Store, cc)
			c.Next()
			responseCache, _ := c.Get("ResponseCache")
			return responseCache.(*store.ResponseCache), nil
		})
		val.(*store.ResponseCache).Write(c.Writer)
		c.Abort()
		return
	}
}

type cachedWriter struct {
	gin.ResponseWriter
	store CacheStore
	cc    *CacheContext
}

func newCachedWriter(writer gin.ResponseWriter, store CacheStore, cc *CacheContext) *cachedWriter {
	return &cachedWriter{writer, store, cc}
}

func (w *cachedWriter) Write(data []byte) (int, error) {
	ret, err := w.ResponseWriter.Write(data)
	if err == nil {
		//cache response
		val := &store.ResponseCache{
			Status: w.Status(),
			Header: w.Header(),
			Data:   data,
			Ttl:    w.cc.Ttl,
		}
		err = w.store.Set(CACHE_PREFIX+w.cc.Key, w.cc.requestPath, val, w.cc.Ttl)
		if err != nil {
			log.Printf("cache store set err:%v", err)
		}
		w.cc.Context.Set("ResponseCache", val)
	}
	return ret, err
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
