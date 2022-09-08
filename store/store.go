package store

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type ResponseCache struct {
	Status int           `json:"status"`
	Header http.Header   `json:"header"`
	Data   []byte        `json:"data"`
	Ttl    time.Duration `json:"-"`
}

//cache write to response
func (c *ResponseCache) Write(w gin.ResponseWriter) {
	w.WriteHeader(c.Status)
	for k, val := range c.Header {
		for _, v := range val {
			w.Header().Add(k, v)
		}
	}
	_, _ = w.Write(c.Data)
}
