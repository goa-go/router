// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package router

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/goa-go/goa"
)

func TestParams(t *testing.T) {
	ps := goa.Params{
		goa.Param{"param1", "value1"},
		goa.Param{"param2", "value2"},
		goa.Param{"param3", "value3"},
	}
	for i := range ps {
		if val := ps.Get(ps[i].Key); val != ps[i].Value {
			t.Errorf("Wrong value for %s: Got %s; Want %s", ps[i].Key, val, ps[i].Value)
		}
	}
	if val := ps.Get("noKey"); val != "" {
		t.Errorf("Expected empty string for not found key; got: %s", val)
	}
}

func handle(c *goa.Context, req *http.Request, router Router) {
	c.Request = req
	c.Method = req.Method
	c.URL = req.URL
	c.Path = req.URL.Path
	router.Handle(c)
}

func TestRouter(t *testing.T) {
	router := New()

	routed := false
	router.Register("GET", "/user/:name", func(c *goa.Context) {
		routed = true
		want := goa.Params{goa.Param{"name", "gopher"}}
		if !reflect.DeepEqual(c.Params, want) {
			t.Fatalf("wrong wildcard values: want %v, got %v", want, c.Params)
		}
	}, goa.Middlewares{})

	c := &goa.Context{}

	req, _ := http.NewRequest("GET", "/user/gopher", nil)
	handle(c, req, *router)

	if !routed {
		t.Fatal("routing failed")
	}
}

func TestRouterAPI(t *testing.T) {
	var get, head, options, post, put, patch, delete, register bool

	router := New()
	router.GET("/GET", func(c *goa.Context) {
		get = true
	})
	router.HEAD("/GET", func(c *goa.Context) {
		head = true
	})
	router.OPTIONS("/GET", func(c *goa.Context) {
		options = true
	})
	router.POST("/POST", func(c *goa.Context) {
		post = true
	})
	router.PUT("/PUT", func(c *goa.Context) {
		put = true
	})
	router.PATCH("/PATCH", func(c *goa.Context) {
		patch = true
	})
	router.DELETE("/DELETE", func(c *goa.Context) {
		delete = true
	})
	router.Register("GET", "/Register", func(c *goa.Context) {
		register = true
	}, goa.Middlewares{})

	c := &goa.Context{}

	r, _ := http.NewRequest("GET", "/GET", nil)
	handle(c, r, *router)
	if !get {
		t.Error("routing GET failed")
	}

	r, _ = http.NewRequest("HEAD", "/GET", nil)
	handle(c, r, *router)
	if !head {
		t.Error("routing HEAD failed")
	}

	r, _ = http.NewRequest("OPTIONS", "/GET", nil)
	handle(c, r, *router)
	if !options {
		t.Error("routing OPTIONS failed")
	}

	r, _ = http.NewRequest("POST", "/POST", nil)
	handle(c, r, *router)
	if !post {
		t.Error("routing POST failed")
	}

	r, _ = http.NewRequest("PUT", "/PUT", nil)
	handle(c, r, *router)
	if !put {
		t.Error("routing PUT failed")
	}

	r, _ = http.NewRequest("PATCH", "/PATCH", nil)
	handle(c, r, *router)
	if !patch {
		t.Error("routing PATCH failed")
	}

	r, _ = http.NewRequest("DELETE", "/DELETE", nil)
	handle(c, r, *router)
	if !delete {
		t.Error("routing DELETE failed")
	}

	r, _ = http.NewRequest("GET", "/Register", nil)
	handle(c, r, *router)
	if !register {
		t.Error("routing Register failed")
	}
}

func TestRoutes(t *testing.T) {
	callNext := false
	c := &goa.Context{}
	router := New()
	routerMiddleware := router.Routes()

	next := func() {
		callNext = true
	}

	routerMiddleware(c, next)
	if !callNext {
		t.Error("router.Routes() failed")
	}
}

func TestRouterRoot(t *testing.T) {
	router := New()
	recv := catchPanic(func() {
		router.GET("noSlashRoot", nil)
	})
	if recv == nil {
		t.Fatal("registering path not beginning with '/' did not panic")
	}
}

func TestRedirectTrailingSlash(t *testing.T) {
	c := &goa.Context{}
	router := New()

	// GET 301
	router.GET("/path", func(c *goa.Context) {})
	r, _ := http.NewRequest("GET", "/path/", nil)
	w := httptest.NewRecorder()
	c.ResponseWriter = w
	handle(c, r, *router)
	if !(w.Code == 301 && strings.Contains(fmt.Sprint(w.Header()), "Location:[/path]")) {
		t.Errorf("Redirect trailing slash failed with get method: Code=%d, Header=%v", w.Code, w.Header())
	}

	// other methods 307
	router.POST("/path", func(c *goa.Context) {})
	r, _ = http.NewRequest("POST", "/path/", nil)
	w = httptest.NewRecorder()
	c.ResponseWriter = w
	handle(c, r, *router)
	if !(w.Code == 307 && strings.Contains(fmt.Sprint(w.Header()), "Location:[/path]")) {
		t.Errorf("Redirect trailing slash failed with post method: Code=%d, Header=%v", w.Code, w.Header())
	}

	// delete trailing slash
	router.PUT("/path/", func(c *goa.Context) {})
	r, _ = http.NewRequest("PUT", "/path", nil)
	w = httptest.NewRecorder()
	c.ResponseWriter = w
	handle(c, r, *router)
	if !(w.Code == 307 && strings.Contains(fmt.Sprint(w.Header()), "Location:[/path/]")) {
		t.Errorf("Redirect trailing slash failed with redirecting /path to /path/: Code=%d, Header=%v", w.Code, w.Header())
	}
}

func TestRedirectFixedPath(t *testing.T) {
	c := &goa.Context{}
	router := New()

	router.GET("/path", func(c *goa.Context) {})
	r, _ := http.NewRequest("GET", "/..//path", nil)
	w := httptest.NewRecorder()
	c.ResponseWriter = w
	handle(c, r, *router)
	if !(w.Code == 301 && strings.Contains(fmt.Sprint(w.Header()), "Location:[/path]")) {
		t.Errorf("Redirect fixed path failed: Code=%d, Header=%v", w.Code, w.Header())
	}
}

func TestRouterChaining(t *testing.T) {
	router1 := New()
	router2 := New()
	router1.NotFound = router2.Handle

	fooHit := false
	router1.POST("/foo", func(c *goa.Context) {
		fooHit = true
	})

	barHit := false
	router2.POST("/bar", func(c *goa.Context) {
		barHit = true
	})

	c := &goa.Context{}

	r, _ := http.NewRequest("POST", "/foo", nil)
	handle(c, r, *router1)
	if !fooHit {
		t.Errorf("Regular routing failed with router chaining.")
		t.FailNow()
	}

	r, _ = http.NewRequest("POST", "/bar", nil)
	handle(c, r, *router1)
	if !barHit {
		t.Errorf("Chained routing failed with router chaining.")
		t.FailNow()
	}
}

func TestRouterOPTIONS(t *testing.T) {
	c := &goa.Context{}
	handlerFunc := func(c *goa.Context) {}

	router := New()
	router.POST("/path", handlerFunc)

	// test not allowed
	// * (server)
	r, _ := http.NewRequest("OPTIONS", "*", nil)
	w := httptest.NewRecorder()
	c.ResponseWriter = w
	handle(c, r, *router)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "POST, OPTIONS" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// path
	r, _ = http.NewRequest("OPTIONS", "/path", nil)
	w = httptest.NewRecorder()
	c.ResponseWriter = w
	handle(c, r, *router)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "POST, OPTIONS" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// add another method
	router.GET("/path", handlerFunc)

	// test again
	// * (server)
	r, _ = http.NewRequest("OPTIONS", "*", nil)
	c.ResponseWriter = w
	handle(c, r, *router)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "POST, GET, OPTIONS" && allow != "GET, POST, OPTIONS" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// path
	r, _ = http.NewRequest("OPTIONS", "/path", nil)
	w = httptest.NewRecorder()
	c.ResponseWriter = w
	handle(c, r, *router)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "POST, GET, OPTIONS" && allow != "GET, POST, OPTIONS" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// custom handler
	var custom bool
	router.OPTIONS("/path", func(c *goa.Context) {
		custom = true
	})

	// test again
	// * (server)
	r, _ = http.NewRequest("OPTIONS", "*", nil)
	c.ResponseWriter = w
	handle(c, r, *router)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "POST, GET, OPTIONS" && allow != "GET, POST, OPTIONS" {
		t.Error("unexpected Allow header value: " + allow)
	}
	if custom {
		t.Error("custom handler called on *")
	}

	// path
	r, _ = http.NewRequest("OPTIONS", "/path", nil)
	w = httptest.NewRecorder()
	c.ResponseWriter = w
	handle(c, r, *router)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	}
	if !custom {
		t.Error("custom handler not called")
	}
}

func TestRouterNotAllowed(t *testing.T) {
	handlerFunc := func(c *goa.Context) {}
	c := &goa.Context{}

	router := New()
	router.POST("/path", handlerFunc)

	// test not allowed
	r, _ := http.NewRequest("GET", "/path", nil)
	w := httptest.NewRecorder()
	c.ResponseWriter = w
	recv := catchPanic(func() {
		handle(c, r, *router)
	})

	if err, ok := recv.(goa.Error); !ok {
		if err.Code != http.StatusMethodNotAllowed {
			t.Errorf("NotAllowed handling failed: Code=%d, Header=%v", err.Code, w.Header())
		}
		t.Errorf("unexpected recv: %v", recv)
	} else if allow := w.Header().Get("Allow"); allow != "POST, OPTIONS" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// add another method
	router.DELETE("/path", handlerFunc)
	router.OPTIONS("/path", handlerFunc) // must be ignored

	// test again
	r, _ = http.NewRequest("GET", "/path", nil)
	w = httptest.NewRecorder()
	c.ResponseWriter = w
	recv = catchPanic(func() {
		handle(c, r, *router)
	})

	if err, ok := recv.(goa.Error); !ok {
		if err.Code != http.StatusMethodNotAllowed {
			t.Errorf("NotAllowed handling failed: Code=%d, Header=%v", err.Code, w.Header())
		}
		t.Errorf("unexpected recv: %v", recv)
	} else if allow := w.Header().Get("Allow"); allow != "POST, DELETE, OPTIONS" && allow != "DELETE, POST, OPTIONS" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// test custom handler
	customMethodNotAllowed := false
	router.MethodNotAllowed = func(c *goa.Context) {
		customMethodNotAllowed = true
	}

	handle(c, r, *router)
	if !customMethodNotAllowed {
		t.Error("coustom MethodNotAllowed handling failed")
	}
	if allow := w.Header().Get("Allow"); allow != "POST, DELETE, OPTIONS" && allow != "DELETE, POST, OPTIONS" {
		t.Error("unexpected Allow header value: " + allow)
	}
}

func TestRouterNotFound(t *testing.T) {
	c := &goa.Context{}

	// Test custom not found handler
	router := New()
	var notFound bool
	router.NotFound = func(c *goa.Context) {
		c.Status(404)
		notFound = true
	}

	r, _ := http.NewRequest("GET", "/nope", nil)
	handle(c, r, *router)
	if !(c.GetStatus() == 404 && notFound == true) {
		t.Errorf("Custom NotFound handler failed: Code=%d", c.GetStatus())
	}

	// Test other method than GET (want 307 instead of 301)
	router.PATCH("/path", func(c *goa.Context) {})
	r, _ = http.NewRequest("PATCH", "/path/", nil)
	w := httptest.NewRecorder()
	c.ResponseWriter = w
	handle(c, r, *router)
	if !(w.Code == 307 && fmt.Sprint(w.Header()) == "map[Location:[/path]]") {
		t.Errorf("Custom NotFound handler failed: Code=%d, Header=%v", w.Code, w.Header())
	}
}

type mockFileSystem struct {
	opened bool
}

func (mfs *mockFileSystem) Open(name string) (http.File, error) {
	mfs.opened = true
	return nil, errors.New("this is just a mock")
}

func TestRouterServeFiles(t *testing.T) {
	c := &goa.Context{}
	router := New()
	mfs := &mockFileSystem{}

	recv := catchPanic(func() {
		router.ServeFiles("/noFilepath", mfs)
	})
	if recv == nil {
		t.Fatal("registering path not ending with '*filepath' did not panic")
	}

	router.ServeFiles("/*filepath", mfs)
	r, _ := http.NewRequest("GET", "/favicon.ico", nil)
	w := httptest.NewRecorder()
	c.ResponseWriter = w
	handle(c, r, *router)
	if !mfs.opened {
		t.Error("serving file failed")
	}
}

func TestRouteMiddleware(t *testing.T) {
	c := &goa.Context{}
	calls := []int{}
	router := New()
	router.GET("/", func(c *goa.Context) {
	}, func(c *goa.Context, next func()) {
		calls = append(calls, 1)
		next()
		calls = append(calls, 5)
	}, func(c *goa.Context, next func()) {
		calls = append(calls, 2)
		next()
		calls = append(calls, 4)
	}, func(c *goa.Context, next func()) {
		calls = append(calls, 3)
		next()
	})

	r, _ := http.NewRequest("GET", "/", nil)
	handle(c, r, *router)

	for i, call := range calls {
		if i+1 != call {
			t.Error("Route use middleware fail")
		}
	}
}
