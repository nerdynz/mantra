package mantra

import (
	"context"
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
	"github.com/twitchtv/twirp"
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
		ls.logger.Log(context.Background(), logLevel, "RPC", "duration", duration, "status", statusStr, "verb", endpoint[0], "endpoint", endpoint[1])
	} else {
		ls.logger.Debug("NOT IMPLEMENTED PRINT LINE NEGRONI")
	}
}

// type templDefaultLayout func(contents ...templ.Component) templ.Component
// type templErrLayout func(err error) templ.Component

func Router(store *datastore.Datastore) *CustomRouter {
	customRouter := &CustomRouter{}
	r := chi.NewRouter()
	customRouter.Mux = r
	customRouter.store = store
	// customRouter.defaultLayout = pageLayout
	// customRouter.errLayout = errLayout

	// customRouter.Key = key
	// customRouter.AuthHandler = authenticate
	return customRouter
}

type CustomRouter struct {
	// Router *mux.Router
	Mux   *chi.Mux
	store *datastore.Datastore

	// defaultLayout templDefaultLayout
	// errLayout     templErrLayout

	// AuthHandler func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore, fn CustomHandlerFunc, authMethod string)
}

type CustomHandlerFunc func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore)

type CustomHandlerFuncTempl func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore) (data any, err error)
type CustomHandlerFuncJSON func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore) ([]byte, error)

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
		fn(w, req, customRouter.store)
	}
}

type AuthMethod string

const (
	TODO   AuthMethod = "TODO"
	SECURE AuthMethod = "SECURE"
	OPEN   AuthMethod = "OPEN"
)

func (customRouter *CustomRouter) jsonHandler(fn CustomHandlerFuncJSON, authMethod AuthMethod, useLayout bool) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		store := customRouter.store
		ctx := req.Context()
		ctx = context.WithValue(ctx, "SiteUlid", "01EDG1D97AWN9V0Q87E4SJ13C7")
		w.Header().Set("Content-Type", "application/json")
		json, err := fn(w, req.WithContext(ctx), store)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		w.Write(json)
	}
}

func WithAuthorization(base http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if slices.Contains(bypassUrls, r.URL.Path) {
		// 	base.ServeHTTP(w, r)
		// 	return
		// }
		slog.Info("asdfasdf")

		ctx := r.Context()
		auth := r.Header.Get("Authorization")

		padlock, err := security.New(r)
		if err != nil {
			twirp.WriteError(w, twirp.NewError(twirp.Unauthenticated, "not logged in"))
			return
		}

		ctx = context.WithValue(ctx, "authorization", auth)
		if !padlock.IsLoggedIn() {
			twirp.WriteError(w, twirp.NewError(twirp.Unauthenticated, "not logged in"))
			return
		}
		ctx = context.WithValue(ctx, "authorization", auth)
		siteUlid, err := padlock.SiteULID()
		if err != nil {
			twirp.WriteError(w, twirp.NewError(twirp.Unauthenticated, err.Error()))
			return
		}
		ctx = context.WithValue(ctx, "site_ulid", siteUlid)
		r = r.WithContext(ctx)
		base.ServeHTTP(w, r)
	})
}

type TwirpServer interface {
	http.Handler
	PathPrefix() string
}

func (customRouter *CustomRouter) Twirp(twirpserver TwirpServer, securityType AuthMethod) {
	if securityType == SECURE {
		customRouter.Mux.Mount(twirpserver.PathPrefix(), WithAuthorization(twirpserver))
	} else {
		// TODO todo

		customRouter.Mux.Mount(twirpserver.PathPrefix(), twirpserver)
	}
}
