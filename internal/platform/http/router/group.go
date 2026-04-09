package router

import "net/http"

type Middleware func(http.Handler) http.Handler

type Group struct {
	mux         *http.ServeMux
	prefix      string
	middlewares []Middleware
}

func NewGroup(mux *http.ServeMux, prefix string, mws ...Middleware) *Group {
	return &Group{
		mux:         mux,
		prefix:      prefix,
		middlewares: mws,
	}
}

func (g *Group) Handle(method string, path string, h http.Handler) {
	final := h

	for i := len(g.middlewares) - 1; i >= 0; i-- {
		final = g.middlewares[i](final)
	}

	g.mux.Handle(method+" "+g.prefix+path, final)
}

func (g *Group) Group(mws ...Middleware) *Group {
	combined := make([]Middleware, 0, len(g.middlewares)+len(mws))
	combined = append(combined, g.middlewares...)
	combined = append(combined, mws...)

	return &Group{
		mux:         g.mux,
		prefix:      g.prefix,
		middlewares: combined,
	}
}
