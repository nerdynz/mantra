package mantra

import (
	"context"
	"log/slog"
	"net/http"
	"os"
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

func TintedLogger(colorize bool) *slog.Logger {
	return slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
			NoColor:    !colorize,
		}),
	)
}

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

func Router(s *datastore.Datastore) *CustomRouter {
	customRouter := &CustomRouter{}
	r := bone.New()
	r.CaseSensitive = false
	customRouter.Mux = r
	customRouter.store = s
	return customRouter
}

type CustomRouter struct {
	Mux   *bone.Mux
	store *datastore.Datastore
}

type AuthMethod string

const (
	TODO   AuthMethod = "TODO"
	SECURE AuthMethod = "SECURE"
	OPEN   AuthMethod = "OPEN"
)

func WithAuthorization(base http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	slog.Info("Twirp", "url registered", twirpserver.PathPrefix())
	if securityType == OPEN {
		customRouter.Mux.Handle(twirpserver.PathPrefix(), twirpserver)
		return
	}

	if securityType == TODO && customRouter.store.Settings.IsDevelopment() {
		slog.Warn("Twirp", "URL marked as TODO", twirpserver.PathPrefix())
		customRouter.Mux.Handle(twirpserver.PathPrefix(), twirpserver)
		return
	}

	customRouter.Mux.Handle(twirpserver.PathPrefix(), WithAuthorization(twirpserver))
}

type CustomHandlerFunc func(w http.ResponseWriter, req *http.Request, store *datastore.Datastore)

func (customRouter *CustomRouter) GET(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	return customRouter.Mux.GetFunc(route, customRouter.handler("GET", routeFunc, securityType))
}

func (customRouter *CustomRouter) POST(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	return customRouter.Mux.PostFunc(route, customRouter.handler("POST", routeFunc, securityType))
}

func (customRouter *CustomRouter) PST(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	return customRouter.Mux.PostFunc(route, customRouter.handler("POST", routeFunc, securityType))
}

func (customRouter *CustomRouter) PUT(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	return customRouter.Mux.PutFunc(route, customRouter.handler("PUT", routeFunc, securityType))
}

func (customRouter *CustomRouter) PATCH(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	return customRouter.Mux.PatchFunc(route, customRouter.handler("PATCH", routeFunc, securityType))
}

func (customRouter *CustomRouter) OPTIONS(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	return customRouter.Mux.OptionsFunc(route, customRouter.handler("OPTIONS", routeFunc, securityType))
}

func (customRouter *CustomRouter) DELETE(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	return customRouter.Mux.DeleteFunc(route, customRouter.handler("DELETE", routeFunc, securityType))
}

func (customRouter *CustomRouter) DEL(route string, routeFunc CustomHandlerFunc, securityType AuthMethod) *bone.Route {
	return customRouter.Mux.DeleteFunc(route, customRouter.handler("DELETE", routeFunc, securityType))
}

func (customRouter *CustomRouter) handler(reqType string, fn CustomHandlerFunc, authMethod AuthMethod) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		authenticate(w, req, customRouter.store, fn, authMethod)
	}
}

func authenticate(w http.ResponseWriter, req *http.Request, store *datastore.Datastore, fn CustomHandlerFunc, authMethod AuthMethod) {
	canonical := store.Settings.Get("CANNONICAL_URL")
	if canonical != "" && store.Settings.IsProduction() {
		root := strings.ToLower(req.Host)
		if !strings.HasSuffix(root, "/") {
			root += "/"
		}
		if !strings.HasSuffix(canonical, "/") {
			canonical += "/"
		}
		if canonical != root {
			redirectURL := "http://"
			if store.Settings.GetBool("IS_HTTPS") {
				redirectURL = "https://"
			}
			redirectURL += strings.TrimRight(canonical, "/")
			if req.URL.Path != "" {
				redirectURL += req.URL.Path
			}
			if req.URL.RawQuery != "" {
				redirectURL += "?" + req.URL.RawQuery
			}
			if req.URL.Fragment != "" {
				redirectURL += "#" + req.URL.Fragment
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
		view := NewViewBucket(w, req, store)
		view.Error(http.StatusForbidden, "Login Expired", err)
		return
	}

	loggedInUser, _, err := padlock.LoggedInUser()
	if err != nil {
		view := NewViewBucket(w, req, store)
		if err.Error() == "redis: nil" {
			view.Error(http.StatusForbidden, "Login Expired", err)
		} else {
			view.Error(http.StatusForbidden, "Auth Failure", err)
		}
		return
	}

	if loggedInUser != nil {
		fn(w, req, store)
		return
	}

	if authMethod == security.Disallow {
		view := NewViewBucket(w, req, store)
		view.Error(http.StatusForbidden, "You're not currently logged in", err)
		return
	}
	if authMethod == security.Redirect {
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}
}
