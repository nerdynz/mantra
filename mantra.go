package mantra

import (
	"context"
	"embed"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/lmittmann/tint"
	"github.com/nerdynz/datastore"
	"github.com/nerdynz/security"
	"github.com/urfave/negroni"
)

func New(log *slog.Logger) *negroni.Negroni {
	nL := &negroni.Logger{
		ALogger: &logSlog{
			log,
		},
	}
	nL.SetFormat(negroni.LoggerDefaultFormat)
	n := negroni.New(negroni.NewRecovery(), nL)
	return n
}

func SlogTintedLogger(isDevelopment bool) *slog.Logger {
	return slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
			NoColor:    isDevelopment,
		}),
	)
}

type logSlog struct {
	logger *slog.Logger
}

func (ls *logSlog) Printf(format string, v ...any) {
	ls.logger.Info(format)
}

func (ls *logSlog) Println(v ...any) {
	output, ok := v[0].(string)
	if ok {
		logLevel := slog.LevelInfo

		// split into something useful
		spl := strings.Split(output, " | ")
		statusStr := spl[1]
		if status, err := strconv.Atoi(statusStr); err == nil {
			if status >= 400 {
				logLevel = slog.LevelError
			}
		}

		duration := strings.TrimPrefix(spl[2], "\t ")
		endpoint := strings.Split(spl[4], " ")
		// ls.logger.Info(output)
		ls.logger.Log(context.Background(), logLevel, "RPC", "duration", duration, "status", statusStr, "verb", endpoint[0], "endpoint", endpoint[1])
	} else {
		ls.logger.Debug("NOT IMPLEMENTED PRINT LINE NEGRONI")
	}
}

// type templDefaultLayout func(contents ...templ.Component) templ.Component
// type templErrLayout func(err error) templ.Component

func Router(store *datastore.Datastore, dist embed.FS) *CustomRouter {
	customRouter := &CustomRouter{}
	r := chi.NewRouter()
	customRouter.Mux = r
	customRouter.Store = store
	customRouter.fs = dist
	// customRouter.defaultLayout = pageLayout
	// customRouter.errLayout = errLayout

	// customRouter.Key = key
	// customRouter.AuthHandler = authenticate
	return customRouter
}

type CustomRouter struct {
	// Router *mux.Router
	Mux   *chi.Mux
	Store *datastore.Datastore
	fs    embed.FS

	// defaultLayout templDefaultLayout
	// errLayout     templErrLayout

	// AuthHandler func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore, fn CustomHandlerFunc, authMethod string)
}

type CustomHandlerFunc func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore) error

type CustomHandlerFuncTempl func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore) (data any, err error)
type CustomHandlerFuncJSON func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore) ([]byte, error)

func (customRouter *CustomRouter) Render(route string, routeFunc CustomHandlerFuncTempl, securityType AuthMethod) {
	customRouter.Mux.Get(route, customRouter.renderHandler(routeFunc, securityType, true))
}

func (customRouter *CustomRouter) Json(route string, routeFunc CustomHandlerFuncJSON, securityType AuthMethod) {
	customRouter.Mux.Get(route, customRouter.jsonHandler(routeFunc, securityType, true))
}

// func (customRouter *CustomRouter) Partial(route string, routeFunc CustomHandlerFuncTempl, securityType AuthMethod) {
// 	customRouter.Mux.Get(route, customRouter.templhandler(routeFunc, securityType, false))
// }

// func (customRouter *CustomRouter) TemplPost(route string, routeFunc CustomHandlerFuncTempl, securityType AuthMethod) {
// 	customRouter.Mux.Post(route, customRouter.templhandler(routeFunc, securityType, true))
// }

func (customRouter *CustomRouter) GET(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) {
	customRouter.Mux.Get(route, customRouter.handler(routeFunc, securityType))
}
func (customRouter *CustomRouter) POST(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) {
	customRouter.Mux.Post(route, customRouter.handler(routeFunc, securityType))
}

// // POST - Post handler
// func (customRouter *CustomRouter) POST(route string, routeFunc CustomHandlerFunc, securityType string) *bone.Route {

// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.PostFunc(route, customRouter.handler("POST", routeFunc, securityType))
// }

// // PST - Post handler with pst for tidier lines
// func (customRouter *CustomRouter) PST(route string, routeFunc CustomHandlerFunc, securityType string) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.PostFunc(route, customRouter.handler("POST", routeFunc, securityType))
// }

// // PUT - Put handler
// func (customRouter *CustomRouter) PUT(route string, routeFunc CustomHandlerFunc, securityType string) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.PutFunc(route, customRouter.handler("PUT", routeFunc, securityType))
// }

// // PATCH - Patch handler
// func (customRouter *CustomRouter) PATCH(route string, routeFunc CustomHandlerFunc, securityType string) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.PatchFunc(route, customRouter.handler("PATCH", routeFunc, securityType))
// }

// // OPTIONS - Options handler
// func (customRouter *CustomRouter) OPTIONS(route string, routeFunc CustomHandlerFunc, securityType string) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.OptionsFunc(route, customRouter.handler("OPTIONS", routeFunc, securityType))
// }

// // DELETE - Delete handler
// func (customRouter *CustomRouter) DELETE(route string, routeFunc CustomHandlerFunc, securityType string) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.DeleteFunc(route, customRouter.handler("DELETE", routeFunc, securityType))
// }

// // DEL - Delete handler
// func (customRouter *CustomRouter) DEL(route string, routeFunc CustomHandlerFunc, securityType string) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.DeleteFunc(route, customRouter.handler("DELETE", routeFunc, securityType))
// }

func (customRouter *CustomRouter) handler(fn CustomHandlerFunc, authMethod AuthMethod) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// req.ParseForm()
		err := fn(w, req, customRouter.Store)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			// customRouter.errLayout(err).Render(req.Context(), w)
		}
		// customRouter.AuthHandler(w, req, flow, customRouter.Store, fn, authMethod)
	}
}

type AuthMethod string

const (
	TODO   AuthMethod = "TODO"
	SECURE AuthMethod = "SECURE"
	OPEN   AuthMethod = "OPEN"
)

func (customRouter *CustomRouter) renderHandler(fn CustomHandlerFuncTempl, authMethod AuthMethod, useLayout bool) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		lock, err := security.New(req)
		if err != nil {
			// customRouter.errLayout(err).Render(req.Context(), w)
			return
		}
		store := customRouter.Store
		// isAuthed, err := CheckAuth(ctx, req)

		if authMethod == OPEN || lock.IsLoggedIn() {
			ctx := req.Context()
			ctx = context.WithValue(ctx, "SiteUlid", "01EDG1D97AWN9V0Q87E4SJ13C7")
			ctx = context.WithValue(ctx, "CurrentUrlPath", req.URL.Path)
			ctx = context.WithValue(ctx, "IsLayoutSkipped", req.Header.Get("HX-Boosted") == "true" || req.Header.Get("HX-request") == "true")
			ctx = context.WithValue(ctx, "IsHxBoosted", req.Header.Get("HX-Boosted") == "true")
			ctx = context.WithValue(ctx, "IsHxRequest", req.Header.Get("HX-request") == "true")
			ctx = context.WithValue(ctx, "HxCurrentUrl", req.Header.Get("hx-current-url"))

			c, err := fn(w, req.WithContext(ctx), store)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				// customRouter.errLayout(err).Render(ctx, w)
			} else if c != nil {
				rnd, err := NewRenderer(customRouter.fs)
				if err != nil {
					w.Write([]byte(err.Error()))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				output, err := rnd.Render(req.URL.Path, c)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "text/html")
				// w.Write([]byte(strings.Replace(string(rnd.indexHTML), "<!--app-html-->", output, 1)))
				w.Write([]byte(output))
			}
			return
		}
		http.Redirect(w, req, "/login", http.StatusTemporaryRedirect)
	}
}

func (customRouter *CustomRouter) jsonHandler(fn CustomHandlerFuncJSON, authMethod AuthMethod, useLayout bool) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		store := customRouter.Store
		ctx := req.Context()
		ctx = context.WithValue(ctx, "SiteUlid", "01EDG1D97AWN9V0Q87E4SJ13C7")
		ctx = context.WithValue(ctx, "CurrentUrlPath", req.URL.Path)
		ctx = context.WithValue(ctx, "IsLayoutSkipped", req.Header.Get("HX-Boosted") == "true" || req.Header.Get("HX-request") == "true")
		ctx = context.WithValue(ctx, "IsHxBoosted", req.Header.Get("HX-Boosted") == "true")
		ctx = context.WithValue(ctx, "IsHxRequest", req.Header.Get("HX-request") == "true")
		ctx = context.WithValue(ctx, "HxCurrentUrl", req.Header.Get("hx-current-url"))

		w.Header().Set("Content-Type", "application/json")

		json, err := fn(w, req.WithContext(ctx), store)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		w.Write(json)
	}
}
