package example

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func performRequest(method, target string, router *gin.Engine) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

func TestCacheSecond(t *testing.T) {
	r := route()
	//r.Run()
	outp := make(chan string, 50)
	outpCache := make(chan string, 50)
	outpCacheSingle := make(chan string, 50)
	for i := 0; i < 50; i++ {
		//default
		go func() {
			resp := performRequest("GET", "/test", r)
			outp <- resp.Body.String()
		}()
		//with cache
		go func() {
			resp := performRequest("GET", "/test-cache-second", r)
			outpCache <- resp.Body.String()
		}()
		//with cache single
		go func() {
			resp := performRequest("GET", "/test-cache-second-single", r)
			outpCacheSingle <- resp.Body.String()
		}()
	}

	for i := 0; i < 50; i++ {
		o := <-outp
		if o != "test" {
			t.Errorf("[/test] err is %s", o)
		}
		oc := <-outpCache
		if oc != "test-cache-second-res" {
			t.Errorf("[/test-cache-second] err is %s", oc)
		}
		ocs := <-outpCacheSingle
		if ocs != "test-cache-second-single-res" {
			t.Errorf("[/test-cache-second-single] err is %s", oc)
		}
	}
}
