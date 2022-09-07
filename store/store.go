package store

import (
	"net/http"
	"time"
)

type ResponseCache struct {
	Status int         `json:"status"`
	Header http.Header `json:"header"`
	Data   []byte      `json:"data"`
	Ttl    time.Duration
}
