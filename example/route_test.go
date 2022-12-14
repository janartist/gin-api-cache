package example

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

var engine *gin.Engine

func init() {
	engine = RunWithRedis()
}

func performRequest(method, target string, router *gin.Engine) *httptest.ResponseRecorder {
	//router.Run(":8080")
	r := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}
func BenchmarkNoCache(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp := performRequest("GET", "/test", engine)
		if resp.Body.String() != "test-res" {
			b.Errorf("[/test] err is %s", resp.Body.String())
		}
	}
}
func BenchmarkCache(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp := performRequest("GET", "/test-cache-second", engine)
		if resp.Body.String() != "test-cache-second-res" {
			b.Errorf("[/test] err is %s", resp.Body.String())
		}
	}
}
func BenchmarkCacheWithSingle(b *testing.B) {
	b.SetParallelism(25000)
	b.N = 25000
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp := performRequest("GET", "/test-cache-second-single", engine)
			source := resp.Header().Get("x-cache-source")
			if source != "" {
				fmt.Printf("x-cache-source-%s \n", source)
			} else {
				fmt.Print("x-cache-source-no \n")
			}
			if resp.Body.String() != "test-cache-second-single-res" {
				b.Errorf("[/test] err is %s %v", resp.Body.String(), resp)
			}
		}
	})
}
func TestCacheWithRedis(t *testing.T) {
	outp := make(chan string, 50)
	outpCache := make(chan string, 50)
	outpCacheSingle := make(chan string, 50)
	for i := 0; i < 50; i++ {
		//default
		go func() {
			resp := performRequest("GET", "/test", engine)
			outp <- resp.Body.String()
		}()
		//with cache
		go func() {
			resp := performRequest("GET", "/test-cache-second", engine)
			outpCache <- resp.Body.String()
		}()
		//with cache single
		go func() {
			resp := performRequest("GET", "/test-cache-second-single", engine)
			outpCacheSingle <- resp.Body.String()
		}()
	}

	for i := 0; i < 50; i++ {
		o := <-outp
		if o != "test-res" {
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
