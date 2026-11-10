package maxbot

type MiddlewareFunc func(HandlerFunc) HandlerFunc

func appendMiddleware(a, b []MiddlewareFunc) []MiddlewareFunc {
	if len(a) == 0 {
		return b
	}

	m := make([]MiddlewareFunc, 0, len(a)+len(b))
	return append(m, append(a, b...)...)
}

func applyMiddleware(h HandlerFunc, m ...MiddlewareFunc) HandlerFunc {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

type Group struct {
	b          *Api
	middleware []MiddlewareFunc
}

func (g *Group) Use(middleware ...MiddlewareFunc) {
	g.middleware = append(g.middleware, middleware...)
}

func (g *Group) Handle(endpoint string, h HandlerFunc, m ...MiddlewareFunc) {
	g.b.Handle(endpoint, h, appendMiddleware(g.middleware, m)...)
}
