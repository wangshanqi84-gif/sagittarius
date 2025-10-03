package server

import (
	"net/http"
)

type Group struct {
	cores    []core
	basePath string
	svr      *Engine
}

func (g *Group) Group(basePath string) *Group {
	path := g.calculateAbsolutePath(basePath)
	group := &Group{
		svr:      g.svr,
		cores:    nil,
		basePath: path,
	}
	if len(g.cores) > 0 {
		group.cores = append(group.cores, g.cores...)
	}
	return group
}

func (g *Group) calculateAbsolutePath(relativePath string) string {
	return joinPaths(g.basePath, relativePath)
}

func (g *Group) combineHandlers(cores ...core) []core {
	finalSize := len(g.cores) + len(cores)
	mergedHandlers := make([]core, finalSize)
	copy(mergedHandlers, g.cores)
	copy(mergedHandlers[len(g.cores):], cores)
	return mergedHandlers
}

func (g *Group) Use(cores ...core) *Group {
	g.cores = append(g.cores, cores...)
	return g
}

func (g *Group) handle(httpMethod, relativePath string, cores ...core) *Group {
	absolutePath := g.calculateAbsolutePath(relativePath)
	cores = g.combineHandlers(cores...)
	g.svr.addRoute(httpMethod, absolutePath, cores...)
	return g
}

func (g *Group) POST(relativePath string, cores ...core) *Group {
	return g.handle(http.MethodPost, relativePath, cores...)
}

func (g *Group) GET(relativePath string, cores ...core) *Group {
	return g.handle(http.MethodGet, relativePath, cores...)
}

func (g *Group) DELETE(relativePath string, cores ...core) *Group {
	return g.handle(http.MethodDelete, relativePath, cores...)
}

func (g *Group) PATCH(relativePath string, cores ...core) *Group {
	return g.handle(http.MethodPatch, relativePath, cores...)
}

func (g *Group) PUT(relativePath string, cores ...core) *Group {
	return g.handle(http.MethodPut, relativePath, cores...)
}
