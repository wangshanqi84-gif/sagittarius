package server

import (
	"log"
	"net/http"
	"strings"
)

const (
	pathParamX = "[+-*/]"
)

type node struct {
	path     string
	name     string
	children map[string]*node
	isRoot   bool
	cores    []core
}

type trees map[string]*node

func newTree() trees {
	return map[string]*node{
		http.MethodPost:   nil,
		http.MethodGet:    nil,
		http.MethodDelete: nil,
		http.MethodPatch:  nil,
		http.MethodPut:    nil,
	}
}

func (t trees) addRoute(method string, path string, cores ...core) {
	if len(path) == 0 {
		panic("path length is zero")
	}
	if t[method] == nil {
		t[method] = &node{
			isRoot:   true,
			children: make(map[string]*node),
		}
	}
	if len(path) == 1 && path[0] == '/' {
		t[method].cores = append(t[method].cores, cores...)
	} else {
		ss := strings.Split(path, "/")
		var ns []string
		for _, s := range ss {
			if s != "" {
				ns = append(ns, s)
			}
		}
		current := t[method]
		for _, s := range ns {
			if len(s) > 2 && s[0] == '{' && s[len(s)-1] == '}' {
				p := pathParamX
				name := s[1 : len(s)-1]
				if _, has := current.children[p]; has || len(ns) == 1 {
					log.Print("addRoute path error, path:", path)
					return
				}
				current.children[p] = &node{
					path:     p,
					name:     name,
					isRoot:   false,
					children: make(map[string]*node),
				}
				current = current.children[p]
			} else {
				if _, has := current.children[s]; !has {
					current.children[s] = &node{
						path:     s,
						name:     s,
						isRoot:   false,
						children: make(map[string]*node),
					}
				}
				current = current.children[s]
			}
		}
		current.cores = append(current.cores, cores...)
	}
}
