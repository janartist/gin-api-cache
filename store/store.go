package store

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

type ResponseCache struct {
	Status int           `json:"status"`
	Header http.Header   `json:"header"`
	Data   []byte        `json:"data"`
	Expire time.Duration `json:"-"`
}

//cache write to response
func (c *ResponseCache) Write(w gin.ResponseWriter) {
	for k, val := range c.Header {
		for _, v := range val {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(c.Status)
	_, _ = w.Write(c.Data)
}
func (c *ResponseCache) AddCacheHeader(key string, source int8) {
	c.Header.Set("X-Cache-Key", key)
	c.Header.Set("Cache-Control", "max-age="+strconv.Itoa(int(c.Expire.Seconds()))+";must-revalidate")
	c.Header.Set("X-Cache-Source", strconv.Itoa(int(source)))
}
func (c ResponseCache) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}
func (c *ResponseCache) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, c)
}
