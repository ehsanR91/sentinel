package api

import (
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi/v5"
)

// RegisterPprofRoutes exposes runtime profiling handlers behind existing auth
// and role middleware. These routes are intended for operator-driven profiling.
func RegisterPprofRoutes(r chi.Router) {
	r.Get("/api/v1/admin/pprof", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/api/v1/admin/pprof/", http.StatusPermanentRedirect)
	})
	r.HandleFunc("/api/v1/admin/pprof/", pprof.Index)
	r.HandleFunc("/api/v1/admin/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/api/v1/admin/pprof/profile", pprof.Profile)
	r.HandleFunc("/api/v1/admin/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/api/v1/admin/pprof/trace", pprof.Trace)
	r.Handle("/api/v1/admin/pprof/allocs", pprof.Handler("allocs"))
	r.Handle("/api/v1/admin/pprof/block", pprof.Handler("block"))
	r.Handle("/api/v1/admin/pprof/goroutine", pprof.Handler("goroutine"))
	r.Handle("/api/v1/admin/pprof/heap", pprof.Handler("heap"))
	r.Handle("/api/v1/admin/pprof/mutex", pprof.Handler("mutex"))
	r.Handle("/api/v1/admin/pprof/threadcreate", pprof.Handler("threadcreate"))
}
