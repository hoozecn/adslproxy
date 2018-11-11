package adslproxy

import (
	"encoding/json"
	"net/http"
	"regexp"
	"time"
)

type forwardPojo struct {
	Name  string `json:"name"`
	Left  string `json:"left"`
	Right string `json:"right"`
}

type nodePojo struct {
	// id of the agent
	Id string `json:"id"`
	// Name of the agent
	Name        string        `json:"name"`
	RemoteIp    string        `json:"remote_ip"`
	ForwardList []forwardPojo `json:"forward_list"`
	// Heartbeat is the time of last heartbeat
	Heartbeat time.Time `json:"heartbeat"`
}

type route struct {
	pattern *regexp.Regexp
	handler http.Handler
}

type RegexpHandler struct {
	routes []*route
}

func (h *RegexpHandler) Handler(pattern *regexp.Regexp, handler http.Handler) {
	h.routes = append(h.routes, &route{pattern, handler})
}

func (h *RegexpHandler) HandleFunc(pattern *regexp.Regexp, handler func(http.ResponseWriter, *http.Request)) {
	h.routes = append(h.routes, &route{pattern, http.HandlerFunc(handler)})
}

func (h *RegexpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range h.routes {
		if route.pattern.MatchString(r.URL.Path) {
			route.handler.ServeHTTP(w, r)
			return
		}
	}
	// no pattern matched; send 404 response
	http.NotFound(w, r)
}

func (s *Server) ListNodesApi() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		nodes := s.ListNodes()
		var data []nodePojo

		for _, node := range nodes {
			var forwardList []forwardPojo
			for _, forward := range node.ForwardList {
				forwardList = append(forwardList, forwardPojo{
					Name:  forward.Name,
					Left:  forward.Left.String(),
					Right: forward.Right,
				})
			}

			data = append(data, nodePojo{
				Id:          node.Id,
				Name:        node.Name,
				RemoteIp:    node.RemoteIp,
				Heartbeat:   node.Heartbeat,
				ForwardList: forwardList,
			})
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(data)
	}
}

func (s *Server) UpdateNodesApi() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)

		json.NewEncoder(w).Encode("")
	}
}

func (s *Server) apiHandler() http.Handler {
	handler := &RegexpHandler{}
	handler.HandleFunc(regexp.MustCompile("/api/nodes/$"), s.ListNodesApi())
	handler.HandleFunc(regexp.MustCompile("/api/nodes/[^/]+/$"), s.UpdateNodesApi())
	return handler
}
