package mantra

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-zoo/bone"
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
func Router(s *datastore.Datastore) *CustomRouter {
	customRouter := &CustomRouter{}
	r := bone.New()
	r.CaseSensitive = false
	customRouter.Mux = r
	customRouter.store = s
	return customRouter
}

// func Router(store *datastore.Datastore) *CustomRouter {
// 	customRouter := &CustomRouter{}
// 	r := chi.NewRouter()
// 	customRouter.Mux = r
// 	customRouter.store = store
// 	// customRouter.defaultLayout = pageLayout
// 	// customRouter.errLayout = errLayout

// 	// customRouter.Key = key
// 	// customRouter.AuthHandler = authenticate
// 	return customRouter
// }

type CustomRouter struct {
	// Router *mux.Router
	Mux   *bone.Mux
	store *datastore.Datastore

	// defaultLayout templDefaultLayout
	// errLayout     templErrLayout

	// AuthHandler func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore, fn CustomHandlerFunc, authMethod string)
}

// // CustomRouter wraps gorilla mux with database, redis and renderer
// type CustomRouter struct {
// 	// Router *mux.Router
// 	Mux      *bone.Mux
// 	Renderer *render.Render
// 	Store    *datastore.Datastore
// 	// AuthHandler func(w http.ResponseWriter, req *http.Request,  store *datastore.Datastore, fn CustomHandlerFunc, authMethod string)
// }

// type CustomHandlerFunc func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore)

// type CustomHandlerFuncTempl func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore) (data any, err error)
// type CustomHandlerFuncJSON func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore) ([]byte, error)

// func (customRouter *CustomRouter) Json(route string, routeFunc CustomHandlerFuncJSON, securityType AuthMethod) {
// 	customRouter.Mux.Get(route, customRouter.jsonHandler(routeFunc, securityType, true))
// }

// // func (customRouter *CustomRouter) Partial(route string, routeFunc CustomHandlerFuncTempl, securityType AuthMethod) {
// // 	customRouter.Mux.Get(route, customRouter.templhandler(routeFunc, securityType, false))
// // }

// // func (customRouter *CustomRouter) TemplPost(route string, routeFunc CustomHandlerFuncTempl, securityType AuthMethod) {
// // 	customRouter.Mux.Post(route, customRouter.templhandler(routeFunc, securityType, true))
// // }

// func (customRouter *CustomRouter) GET(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) {
// 	customRouter.Mux.Get(route, customRouter.handler(routeFunc, securityType))
// }
// func (customRouter *CustomRouter) POST(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) {
// 	customRouter.Mux.Post(route, customRouter.handler(routeFunc, securityType))
// }

// // POST - Post handler
// func (customRouter *CustomRouter) POST(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {

// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.PostFunc(route, customRouter.handler("POST", routeFunc, securityType))
// }

// // PST - Post handler with pst for tidier lines
// func (customRouter *CustomRouter) PST(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.PostFunc(route, customRouter.handler("POST", routeFunc, securityType))
// }

// // PUT - Put handler
// func (customRouter *CustomRouter) PUT(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.PutFunc(route, customRouter.handler("PUT", routeFunc, securityType))
// }

// // PATCH - Patch handler
// func (customRouter *CustomRouter) PATCH(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.PatchFunc(route, customRouter.handler("PATCH", routeFunc, securityType))
// }

// // OPTIONS - Options handler
// func (customRouter *CustomRouter) OPTIONS(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.OptionsFunc(route, customRouter.handler("OPTIONS", routeFunc, securityType))
// }

// // DELETE - Delete handler
// func (customRouter *CustomRouter) DELETE(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.DeleteFunc(route, customRouter.handler("DELETE", routeFunc, securityType))
// }

// // DEL - Delete handler
// func (customRouter *CustomRouter) DEL(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.DeleteFunc(route, customRouter.handler("DELETE", routeFunc, securityType))
// }

// func (customRouter *CustomRouter) handler(fn CustomHandlerFunc, authMethod AuthMethod) http.HandlerFunc {
// 	return func(w http.ResponseWriter, req *http.Request) {
// 		fn(w, req, customRouter.store)
// 	}
// }

type AuthMethod string

const (
	TODO   AuthMethod = "TODO"
	SECURE AuthMethod = "SECURE"
	OPEN   AuthMethod = "OPEN"
)

// func (customRouter *CustomRouter) jsonHandler(fn CustomHandlerFuncJSON, authMethod AuthMethod, useLayout bool) http.HandlerFunc {
// 	return func(w http.ResponseWriter, req *http.Request) {
// 		store := customRouter.store
// 		ctx := req.Context()
// 		ctx = context.WithValue(ctx, "SiteUlid", "01EDG1D97AWN9V0Q87E4SJ13C7")
// 		w.Header().Set("Content-Type", "application/json")
// 		json, err := fn(w, req.WithContext(ctx), store)
// 		if err != nil {
// 			w.Write([]byte(err.Error()))
// 			return
// 		}
// 		w.Write(json)
// 	}
// }

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
		customRouter.Mux.Handle(twirpserver.PathPrefix(), WithAuthorization(twirpserver))
	} else {
		// TODO todo

		customRouter.Mux.Handle(twirpserver.PathPrefix(), twirpserver)
	}
}

type CustomHandlerFunc func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore)

// // GET - Get handler
// func (customRouter *CustomRouter) Application(route string, path string) *bone.Route {
// 	//route = strings.ToLower(route)
// 	return customRouter.Mux.GetFunc(route, customRouter.appHandler(path))
// }

func (customRouter *CustomRouter) GET(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	//route = strings.ToLower(route)
	return customRouter.Mux.GetFunc(route, customRouter.handler("GET", routeFunc, securityType))
}

// POST - Post handler
func (customRouter *CustomRouter) POST(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {

	//route = strings.ToLower(route)
	return customRouter.Mux.PostFunc(route, customRouter.handler("POST", routeFunc, securityType))
}

// PST - Post handler with pst for tidier lines
func (customRouter *CustomRouter) PST(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	//route = strings.ToLower(route)
	return customRouter.Mux.PostFunc(route, customRouter.handler("POST", routeFunc, securityType))
}

// PUT - Put handler
func (customRouter *CustomRouter) PUT(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	//route = strings.ToLower(route)
	return customRouter.Mux.PutFunc(route, customRouter.handler("PUT", routeFunc, securityType))
}

// PATCH - Patch handler
func (customRouter *CustomRouter) PATCH(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	//route = strings.ToLower(route)
	return customRouter.Mux.PatchFunc(route, customRouter.handler("PATCH", routeFunc, securityType))
}

// OPTIONS - Options handler
func (customRouter *CustomRouter) OPTIONS(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	//route = strings.ToLower(route)
	return customRouter.Mux.OptionsFunc(route, customRouter.handler("OPTIONS", routeFunc, securityType))
}

// DELETE - Delete handler
func (customRouter *CustomRouter) DELETE(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	//route = strings.ToLower(route)
	return customRouter.Mux.DeleteFunc(route, customRouter.handler("DELETE", routeFunc, securityType))
}

// DEL - Delete handler
func (customRouter *CustomRouter) DEL(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	//route = strings.ToLower(route)
	return customRouter.Mux.DeleteFunc(route, customRouter.handler("DELETE", routeFunc, securityType))
}

func (customRouter *CustomRouter) handler(reqType string, fn CustomHandlerFunc, authMethod AuthMethod) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		authenticate(w, req, customRouter.store, fn, authMethod)
		// customRouter.AuthHandler(w, req, flow, customRouter.Store, fn, authMethod)
	}
}

func authenticate(w http.ResponseWriter, req *http.Request, store *datastore.Datastore, fn CustomHandlerFunc, authMethod AuthMethod) {
	// canonical host
	canonical := store.Settings.Get("CANNONICAL_URL")
	if canonical != "" && store.Settings.IsProduction() { // set in ENV
		root := strings.ToLower(req.Host)
		if !strings.HasSuffix(root, "/") {
			root += "/"
		}
		if !strings.HasSuffix(canonical, "/") {
			canonical += "/"
		}
		// logrus.Info("root", root)
		// logrus.Info("root", canonical)
		if canonical != root {
			redirectURL := "http://"
			if store.Settings.GetBool("IS_HTTPS") {
				redirectURL = "https://"
			}
			redirectURL += strings.TrimRight(canonical, "/")
			if req.URL.Path != "" {
				redirectURL += req.URL.Path
				// logrus.Info("0", redirectURL)
			}
			if req.URL.RawQuery != "" {
				redirectURL += "?" + req.URL.RawQuery
				// logrus.Info("2", redirectURL)
			}
			if req.URL.Fragment != "" {
				redirectURL += "#" + req.URL.Fragment
				// logrus.Info("2", redirectURL)
			}

			http.Redirect(w, req, redirectURL, http.StatusMovedPermanently)
			return
		}
	}

	if authMethod == OPEN {
		fn(w, req, store)
		return
	}
	padlock, err := security.New(req)
	if err != nil {
		errorOut(w, req, http.StatusForbidden, "Login Expired", err)
		return
	}

	// if we are at this point then we want a login
	loggedInUser, _, err := padlock.LoggedInUser()
	if err != nil {
		if err.Error() == "redis: nil" {
			// ignore it, its expired from cache
			errorOut(w, req, http.StatusForbidden, "Login Expired", err)
		} else {
			errorOut(w, req, http.StatusForbidden, "Auth Failure", err)
		}
		return
	}

	if loggedInUser != nil {
		fn(w, req, store)
		return
	}

	// if we have reached this point then the user doesn't have access
	if authMethod == security.Disallow {
		errorOut(w, req, http.StatusForbidden, "You're not currently logged in", err)
		return
	}
	if authMethod == security.Redirect {
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}
}

func errorOut(w http.ResponseWriter, req *http.Request, status int, friendly string, errs ...error) {
	isText := false
	//https: //stackoverflow.com/questions/24809287/how-do-you-get-a-golang-program-to-print-the-line-number-of-the-error-it-just-ca
	errStr := ""
	lineNumber := -1
	funcName := "Not Specified"
	fileName := "Not Specified"

	if len(errs) > 0 {
		for _, err := range errs {
			if err != nil {
				errStr += err.Error() + "\n"
			} else {
				errStr += "No Error Specified \n"
			}
		}
		// notice that we're using 1, so it will actually log the where // actually one is errorOut
		// the error happened, 0 = this function, we don't want that.
		pc, file, line, _ := runtime.Caller(2)
		lineNumber = line
		funcName = runtime.FuncForPC(pc).Name()
		fileName = file
	}

	data := &errorData{
		friendly,
		errStr,
		lineNumber,
		funcName,
		fileName,
	}

	slog.Error(data.nicelyFormatted())

	if isText {
		w.WriteHeader(status)
		// w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(data.nicelyFormatted()))
		return
	}

	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

type errorData struct {
	Friendly     string
	Error        string
	LineNumber   int
	FunctionName string
	FileName     string
}

func (e *errorData) nicelyFormatted() string {
	str := ""
	str += "Friendly Message: \n\t" + e.Friendly + "\n"
	str += "Error: \n\t" + e.Error + "\n"
	str += "File: \n\t" + e.FileName + ":" + strconv.Itoa(e.LineNumber) + "\n"
	// str += "LineNumber: \n\t" + strconv.Itoa(e.LineNumber) + "\n"
	str += "FunctionName: \n\t" + e.FunctionName + "\n"
	return str
}

// func (customRouter *CustomRouter) appHandler(file string) func(w http.ResponseWriter, req *http.Request) {
// 	return func(w http.ResponseWriter, req *http.Request) {
// 		flw := flow.New(w, req, customRouter.Renderer, customRouter.Store, customRouter.Key)
// 		fullpath, err := os.Getwd()
// 		if err != nil {
// 			errorOut(w, req, false, http.StatusInternalServerError, "Failed to get current working directory", err)
// 		}
// 		fullpath += file
// 		flw.StaticFile(200, fullpath, "text/html; charset=utf-8")
// 	}
// }
