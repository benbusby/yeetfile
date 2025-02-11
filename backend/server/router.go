package server

import (
	"log"
	"net/http"
	"strings"
	"yeetfile/shared"
	"yeetfile/shared/endpoints"
)

type Route struct {
	Method string
	Path   string
}

type RouteDef struct {
	Methods HttpMethod
	Path    endpoints.Endpoint
	Handler http.HandlerFunc
}

type router struct {
	routes   map[Route]http.HandlerFunc
	reserved []string
}

func (r *router) AddRoute(method string, path string, handler http.HandlerFunc) {
	route := Route{Path: path, Method: method}
	r.routes[route] = handler

	// Reserve endpoint to help pattern matching on new requests
	endpoint := strings.Split(route.Path, "/")[1]
	if len(endpoint) > 0 && endpoint != "*" {
		r.reserved = append(r.reserved, endpoint)
	}
}

func (r *router) AddRoutes(routes []RouteDef) {
	for _, route := range routes {
		for methodInt, methodStr := range MethodMap {
			if route.Methods&methodInt == 0 {
				continue
			}

			path := string(route.Path)
			handler := DefaultHeadersMiddleware(route.Handler)

			// Check for paths with optional segments
			if strings.Contains(path, "/?") {
				optPath := strings.Replace(path, "/?", "", 1)
				wildPath := strings.Replace(path, "/?", "/*", 1)
				r.AddRoute(methodStr, optPath, handler)
				r.AddRoute(methodStr, wildPath, handler)
			} else {
				r.AddRoute(methodStr, path, handler)
			}
		}
	}
}

// ServeHTTP finds the proper routing handler for the provided path.
func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for el, handler := range r.routes {
		if r.matchPath(el.Path, req.URL.Path) && el.Method == req.Method {
			if req.URL.Path != string(endpoints.Up) {
				log.Printf("%s %s\n", req.Method, req.URL)
			}

			handler(w, req)
			return
		}
	}

	log.Printf("Error: %s %s", req.Method, req.URL)
	http.NotFound(w, req)
}

// matchPath takes a URL path and determines if it's a match for a particular
// handler. This allows differentiating between two paths where the only
// difference is a wildcard (i.e. "/u" and "/u/*" for uploadInit and uploadData)
func (r *router) matchPath(pattern, path string) bool {
	parts := strings.Split(pattern, "/")
	segments := strings.Split(path, "/")

	isWildcard := parts[1] == "*"
	isEndpoint := shared.ArrayContains(r.reserved, segments[1])

	if len(parts) != len(segments) || (isWildcard && isEndpoint) {
		return false
	}

	for i, part := range parts {
		if part == "*" && len(path) > 1 {
			continue
		}

		if part != segments[i] {
			return false
		}
	}

	return true
}
