// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gin

import (
	"net/http"
	"path"
	"regexp"
	"strings"
)

type (
	RateLimitBlueprint struct {
		Tag    string
		Burst  int32
		Count  int32
		Period int32
	}
	RouteRate struct {
		Bp   *RateLimitBlueprint
		Cost int32
	}

	IRouter interface {
		IRoutes
		Group(string, ...HandlerFunc) *RouterGroup
	}

	IRoutes interface {
		Use(...HandlerFunc) IRoutes

		Handle(*RouteRate, string, string, ...HandlerFunc) IRoutes
		Any(*RouteRate, string, ...HandlerFunc) IRoutes
		GET(*RouteRate, string, ...HandlerFunc) IRoutes
		POST(*RouteRate, string, ...HandlerFunc) IRoutes
		DELETE(*RouteRate, string, ...HandlerFunc) IRoutes
		PATCH(*RouteRate, string, ...HandlerFunc) IRoutes
		PUT(*RouteRate, string, ...HandlerFunc) IRoutes
		OPTIONS(*RouteRate, string, ...HandlerFunc) IRoutes
		HEAD(*RouteRate, string, ...HandlerFunc) IRoutes

		StaticFile(*RouteRate, string, string) IRoutes
		Static(*RouteRate, string, string) IRoutes
		StaticFS(*RouteRate, string, http.FileSystem) IRoutes
	}

	// RouterGroup is used internally to configure router, a RouterGroup is associated with a prefix
	// and an array of handlers (middleware)
	RouterGroup struct {
		Handlers HandlersChain
		basePath string
		engine   *Engine
		root     bool
	}
)

var _ IRouter = &RouterGroup{}

// Use adds middleware to the group, see example code in github.
func (group *RouterGroup) Use(middleware ...HandlerFunc) IRoutes {
	group.Handlers = append(group.Handlers, middleware...)
	return group.returnObj()
}

// Group creates a new router group. You should add all the routes that have common middlwares or the same path prefix.
// For example, all the routes that use a common middlware for authorization could be grouped.
func (group *RouterGroup) Group(relativePath string, handlers ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		Handlers: group.combineHandlers(handlers),
		basePath: group.calculateAbsolutePath(relativePath),
		engine:   group.engine,
	}
}

func (group *RouterGroup) BasePath() string {
	return group.basePath
}

func (group *RouterGroup) handle(rld *RouteRate, httpMethod, relativePath string, handlers HandlersChain) IRoutes {
	absolutePath := group.calculateAbsolutePath(relativePath)
	handlers = group.combineHandlers(handlers)
	group.engine.addRoute(rld, httpMethod, absolutePath, handlers)
	return group.returnObj()
}

// Handle registers a new request handle and middleware with the given path and method.
// The last handler should be the real handler, the other ones should be middleware that can and should be shared among different routes.
// See the example code in github.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (group *RouterGroup) Handle(rld *RouteRate, httpMethod, relativePath string, handlers ...HandlerFunc) IRoutes {
	if matches, err := regexp.MatchString("^[A-Z]+$", httpMethod); !matches || err != nil {
		panic("http method " + httpMethod + " is not valid")
	}
	return group.handle(rld, httpMethod, relativePath, handlers)
}

// POST is a shortcut for router.Handle("POST", path, handle)
func (group *RouterGroup) POST(rld *RouteRate, relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(rld, "POST", relativePath, handlers)
}

// GET is a shortcut for router.Handle("GET", path, handle)
func (group *RouterGroup) GET(rld *RouteRate, relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(rld, "GET", relativePath, handlers)
}

// DELETE is a shortcut for router.Handle("DELETE", path, handle)
func (group *RouterGroup) DELETE(rld *RouteRate, relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(rld, "DELETE", relativePath, handlers)
}

// PATCH is a shortcut for router.Handle("PATCH", path, handle)
func (group *RouterGroup) PATCH(rld *RouteRate, relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(rld, "PATCH", relativePath, handlers)
}

// PUT is a shortcut for router.Handle("PUT", path, handle)
func (group *RouterGroup) PUT(rld *RouteRate, relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(rld, "PUT", relativePath, handlers)
}

// OPTIONS is a shortcut for router.Handle("OPTIONS", path, handle)
func (group *RouterGroup) OPTIONS(rld *RouteRate, relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(rld, "OPTIONS", relativePath, handlers)
}

// HEAD is a shortcut for router.Handle("HEAD", path, handle)
func (group *RouterGroup) HEAD(rld *RouteRate, relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(rld, "HEAD", relativePath, handlers)
}

// Any registers a route that matches all the HTTP methods.
// GET, POST, PUT, PATCH, HEAD, OPTIONS, DELETE, CONNECT, TRACE
func (group *RouterGroup) Any(rld *RouteRate, relativePath string, handlers ...HandlerFunc) IRoutes {
	group.handle(rld, "GET", relativePath, handlers)
	group.handle(rld, "POST", relativePath, handlers)
	group.handle(rld, "PUT", relativePath, handlers)
	group.handle(rld, "PATCH", relativePath, handlers)
	group.handle(rld, "HEAD", relativePath, handlers)
	group.handle(rld, "OPTIONS", relativePath, handlers)
	group.handle(rld, "DELETE", relativePath, handlers)
	group.handle(rld, "CONNECT", relativePath, handlers)
	group.handle(rld, "TRACE", relativePath, handlers)
	return group.returnObj()
}

// StaticFile registers a single route in order to server a single file of the local filesystem.
// router.StaticFile("favicon.ico", "./resources/favicon.ico")
func (group *RouterGroup) StaticFile(rld *RouteRate, relativePath, filepath string) IRoutes {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static file")
	}
	handler := func(c *Context) {
		c.File(filepath)
	}
	group.GET(rld, relativePath, handler)
	group.HEAD(rld, relativePath, handler)
	return group.returnObj()
}

// Static serves files from the given file system root.
// Internally a http.FileServer is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
// To use the operating system's file system implementation,
// use :
//     router.Static("/static", "/var/www")
func (group *RouterGroup) Static(rld *RouteRate, relativePath, root string) IRoutes {
	return group.StaticFS(rld, relativePath, Dir(root, false))
}

// StaticFS works just like `Static()` but a custom `http.FileSystem` can be used instead.
// Gin by default user: gin.Dir()
func (group *RouterGroup) StaticFS(rld *RouteRate, relativePath string, fs http.FileSystem) IRoutes {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static folder")
	}
	handler := group.createStaticHandler(relativePath, fs)
	urlPattern := path.Join(relativePath, "/*filepath")

	// Register GET and HEAD handlers
	group.GET(rld, urlPattern, handler)
	group.HEAD(rld, urlPattern, handler)
	return group.returnObj()
}

func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := group.calculateAbsolutePath(relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	_, nolisting := fs.(*onlyfilesFS)
	return func(c *Context) {
		if nolisting {
			c.Writer.WriteHeader(404)
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}

func (group *RouterGroup) combineHandlers(handlers HandlersChain) HandlersChain {
	finalSize := len(group.Handlers) + len(handlers)
	if finalSize >= int(abortIndex) {
		panic("too many handlers")
	}
	mergedHandlers := make(HandlersChain, finalSize)
	copy(mergedHandlers, group.Handlers)
	copy(mergedHandlers[len(group.Handlers):], handlers)
	return mergedHandlers
}

func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return joinPaths(group.basePath, relativePath)
}

func (group *RouterGroup) returnObj() IRoutes {
	if group.root {
		return group.engine
	}
	return group
}
