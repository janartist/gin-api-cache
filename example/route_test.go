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
	b.N = 500
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for i := 0; i < 10; i++ {
				resp := performRequest("GET", "/test-cache-second-single", engine)
				source := resp.Header().Get("X-Cache-Source")
				if source != "" {
					fmt.Printf("X-cache-source-%s \n", source)
				} else {
					fmt.Print("X-cache-source-no \n")
				}
				if resp.Body.String() != "test-cache-second-single-res" {
					b.Errorf("[/test] err is %s %v", resp.Body.String(), resp)
				}
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
		//go func() {
		//	resp := performRequest("GET", "/test", engine)
		//	outp <- resp.Body.String()
		//}()
		for j := 0; j < 50; j++ {
			//with cache
			go func() {
				resp := performRequest("GET", "/test-cache-second?uid=1", engine)
				source := resp.Header().Get("X-Cache-Source")
				outpCache <- resp.Body.String() + "-" + source
			}()
		}
		//with cache single
		//go func() {
		//	resp := performRequest("GET", "/test-cache-second-single", engine)
		//	source := resp.Header().Get("X-Cache-Source")
		//	outpCache <- resp.Body.String() + "-" + source
		//}()
	}
	for i := 0; i < 200; i++ {
		select {
		case o := <-outp:
			if o != "test-res" {
				t.Errorf("[/test] err is %s", o)
			}
		case oc := <-outpCache:
			fmt.Print(oc, "\n")
		case ocs := <-outpCacheSingle:
			fmt.Print(ocs, "\n")

		}
	}
}
