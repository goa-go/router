// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package router

import (
	"net/http"

	"github.com/goa-go/goa"
)

// Handler is the type of goa-router handle function.
type Handler func(*goa.Context)

// Router is a http.Handler which can be used to dispatch requests to different
// handler functions via configurable routes
type Router struct {
	trees map[string]*node

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 307 for all other request methods.
	RedirectTrailingSlash bool

	// If enabled, the router tries to fix the current request path, if no
	// handler is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the router does a case-insensitive lookup of the cleaned path.
	// If a handler can be found for this route, the router makes a redirection
	// to the corrected path with status code 301 for GET requests and 307 for
	// all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// RedirectTrailingSlash is independent of this option.
	RedirectFixedPath bool

	// If enabled, the router checks if another method is allowed for the
	// current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed'
	// and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound
	// handler.
	HandleMethodNotAllowed bool

	// If enabled, the router automatically replies to OPTIONS requests.
	// Custom OPTIONS handlers take priority over automatic replies.
	HandleOPTIONS bool

	// Configurable http.Handler which is called when no matching route is found.
	NotFound Handler

	// Configurable http.Handler which is called when a request
	// cannot be routed and HandleMethodNotAllowed is true.
	// If it is not set, http.Error with http.StatusMethodNotAllowed is used.
	// The "Allow" header with allowed request methods is set before the handler
	// is called.
	MethodNotAllowed Handler

	// All allow methods
	Methods []string
}

// New returns a new initialized Router.
// Path auto-correction, including trailing slashes, is enabled by default.
func New() *Router {
	return &Router{
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleMethodNotAllowed: true,
		HandleOPTIONS:          true,
	}
}

// GET registers a new request handle with the given path and get method.
func (r *Router) GET(path string, handler Handler, middlewares ...goa.Middleware) {
	r.Register("GET", path, handler, middlewares)
}

// HEAD registers a new request handle with the given path and head method.
func (r *Router) HEAD(path string, handler Handler, middlewares ...goa.Middleware) {
	r.Register("HEAD", path, handler, middlewares)
}

// OPTIONS registers a new request handle with the given path and options method.
func (r *Router) OPTIONS(path string, handler Handler, middlewares ...goa.Middleware) {
	r.Register("OPTIONS", path, handler, middlewares)
}

// POST registers a new request handle with the given path and post method.
func (r *Router) POST(path string, handler Handler, middlewares ...goa.Middleware) {
	r.Register("POST", path, handler, middlewares)
}

// PUT registers a new request handle with the given path and put method.
func (r *Router) PUT(path string, handler Handler, middlewares ...goa.Middleware) {
	r.Register("PUT", path, handler, middlewares)
}

// PATCH registers a new request handle with the given path and patch method.
func (r *Router) PATCH(path string, handler Handler, middlewares ...goa.Middleware) {
	r.Register("PATCH", path, handler, middlewares)
}

// DELETE registers a new request handle with the given path and delete method.
func (r *Router) DELETE(path string, handler Handler, middlewares ...goa.Middleware) {
	r.Register("DELETE", path, handler, middlewares)
}

// Register registers a new request handle with the given path and method.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *Router) Register(method, path string, handler Handler, middlewares goa.Middlewares) {
	if path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}

	if r.trees == nil {
		r.trees = make(map[string]*node)
	}

	root := r.trees[method]
	if root == nil {
		root = new(node)
		r.trees[method] = root
	}

	if len(middlewares) > 0 {
		middlewareHandler := func(c *goa.Context) {
			compose(middlewares)(c)
			handler(c)
		}
		root.addRoute(path, middlewareHandler)
	} else {
		root.addRoute(path, handler)
	}
}

func compose(m goa.Middlewares) Handler {
	var dispatch func(i int)
	return func(c *goa.Context) {
		dispatch = func(i int) {
			if i == len(m) {
				return
			}
			fn := m[i]
			fn(c, func() {
				dispatch(i + 1)
			})
		}
		dispatch(0)
	}
}

func (r *Router) allowed(path, reqMethod string) (allow string) {
	if path == "*" { // server-wide
		for method := range r.trees {
			if method == "OPTIONS" {
				continue
			}

			// add request method to list of allowed methods
			if len(allow) == 0 {
				allow = method
			} else {
				allow += ", " + method
			}
		}
	} else { // specific path
		for method := range r.trees {
			// Skip the requested method - we already tried this one
			if method == reqMethod || method == "OPTIONS" {
				continue
			}

			handle, _, _ := r.trees[method].getValue(path)
			if handle != nil {
				// add request method to list of allowed methods
				if len(allow) == 0 {
					allow = method
				} else {
					allow += ", " + method
				}
			}
		}
	}
	if len(allow) > 0 {
		allow += ", OPTIONS"
	}
	return
}

// ServeFiles serves files from the given file system root.
// The path must end with "/*filepath", files are then served from the local
// path /defined/root/dir/*filepath.
// For example if root is "/etc" and *filepath is "passwd", the local file
// "/etc/passwd" would be served.
// Internally a http.FileServer is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
// To use the operating system's file system implementation,
// use http.Dir:
// router.ServeFiles("/src/*filepath", http.Dir("/var/www"))
func (r *Router) ServeFiles(path string, root http.FileSystem) {
	if len(path) < 10 || path[len(path)-10:] != "/*filepath" {
		panic("path must end with /*filepath in path '" + path + "'")
	}

	fileServer := http.FileServer(root)

	r.GET(path, func(c *goa.Context) {
		c.URL.Path = c.Param("filepath")
		fileServer.ServeHTTP(c.ResponseWriter, c.Request)
		// c.Handled = true
	})
}

// Handle is goa-router's handle function.
func (r *Router) Handle(c *goa.Context) {
	path := c.Path

	if root := r.trees[c.Method]; root != nil {
		if Handler, ps, tsr := root.getValue(path); Handler != nil {
			c.Params = ps
			Handler(c)
			return
		} else if c.Method != "CONNECT" && path != "/" {
			code := 301 // Permanent redirect, request with GET method
			if c.Method != "GET" {
				// Temporary redirect, request with same method
				// As of Go 1.3, Go does not support status code 308.
				code = 307
			}

			if tsr && r.RedirectTrailingSlash {
				if len(path) > 1 && path[len(path)-1] == '/' {
					c.URL.Path = path[:len(path)-1]
					c.Path = path[:len(path)-1]
				} else {
					c.URL.Path = path + "/"
					c.Path = path + "/"
				}
				c.Redirect(code, c.URL.String())
				return
			}

			// Try to fix the request path
			if r.RedirectFixedPath {
				fixedPath, found := root.findCaseInsensitivePath(
					CleanPath(path),
					r.RedirectTrailingSlash,
				)
				if found {
					c.Path = string(fixedPath)
					c.Redirect(code, c.URL.String())
					return
				}
			}
		}
	}

	if c.Method == "OPTIONS" && r.HandleOPTIONS {
		// Handle OPTIONS requests
		if allow := r.allowed(path, c.Method); len(allow) > 0 {
			c.SetHeader("Allow", allow)
			return
		}
	} else {
		// Handle 405
		if r.HandleMethodNotAllowed {
			if allow := r.allowed(path, c.Method); len(allow) > 0 {
				c.SetHeader("Allow", allow)
				if r.MethodNotAllowed != nil {
					r.MethodNotAllowed(c)
				} else {
					c.Error(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
				}
				return
			}
		}
	}

	// Handle 404
	if r.NotFound != nil {
		r.NotFound(c)
	}
}

// Routes returns a goa.Middleware.
// app.Use(router.Routes())
func (r *Router) Routes() goa.Middleware {
	return func(c *goa.Context, next func()) {
		r.Handle(c)
		next()
	}
}
