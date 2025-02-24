package mantra

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"strings"

	"golang.org/x/net/html"
	"rogchap.com/v8go"
)

// Renderer renders a React application to HTML.
type Renderer struct {
	pool          *IsolatePool
	ssrScriptName string

	clientScript string
	clientCss    string
	serverEntry  string
}

// NewRenderer creates a new server side renderer for a given script.
func NewRenderer(dist embed.FS) (*Renderer, error) {
	slog.Info("asdf", "xx", "xx")

	err := fs.WalkDir(dist, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		slog.Info("file", "path", path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	frontendDist, err := fs.Sub(dist, "dist/client")
	if err != nil {
		return nil, err
	}

	serverDist, err := fs.Sub(dist, "dist/server")
	if err != nil {
		return nil, err
	}
	ssrScriptName := "server.js"

	indexHTML, err := readFSFile(frontendDist, "index.html")
	if err != nil {
		return nil, err
	}

	script, css, err := extractAssets(string(indexHTML))
	if err != nil {
		return nil, err
	}
	slog.Info("asdf", "script", script)
	slog.Info("asdf", "css", css)

	serverEntry, err := readFSFile(serverDist, "server.js")
	if err != nil {
		return nil, err
	}

	return &Renderer{
		pool:          NewIsolatePool(string(serverEntry), ssrScriptName),
		ssrScriptName: ssrScriptName,
		clientScript:  script,
		clientCss:     css,
	}, nil
}

// func (rnd *Renderer) JSON(w http.ResponseWriter, r *http.Request, ctx *flow.Flow, store *datastore.Datastore) {
// 	slog.Info("URL", "r.URL.Path", r.URL.Path)
// 	ph := page.NewPageHelper(store)
// 	page, err := ph.LoadFullPage("01EHZXH0YBCM8Q8PEFDZB8K3WW", ctx.URLParam("slug"))
// 	if err != nil {
// 		ctx.ErrorHTML(500, "No page avaliable", err)
// 		return
// 	}
// 	ctx.JSON(http.StatusOK, page)
// }

// func (rnd *Renderer) Metadata(w http.ResponseWriter, r *http.Request, ctx *flow.Flow, store *datastore.Datastore) {
// 	slog.Info("URL", "r.URL.Path", r.URL.Path)
// 	ph := page.NewPageHelper(store)
// 	meta, err := ph.LoadMetadata()
// 	if err != nil {
// 		ctx.ErrorHTML(500, "No page avaliable", err)
// 		return
// 	}
// 	ctx.JSON(http.StatusOK, meta)
// }

// func (rnd *Renderer) Handle(w http.ResponseWriter, r *http.Request, store *datastore.Datastore) {
// 	// ph := page.NewPageHelper(store)

// 	// meta, err := ph.LoadMetadata()
// 	// if err != nil {
// 	// 	log.Println(err)
// 	// 	w.WriteHeader(http.StatusInternalServerError)
// 	// 	return
// 	// }

// 	// slug := ctx.URLParam("slug")
// 	// if slug == "" {
// 	// 	slug = "home"
// 	// }
// 	// page, err := ph.LoadFullPage("01EHZXH0YBCM8Q8PEFDZB8K3WW", slug)
// 	// if err != nil {
// 	// 	ctx.ErrorHTML(500, "No page avaliable", err)
// 	// 	return
// 	// }

// 	output, err := rnd.Render(r.URL.Path, data)
// 	if err != nil {
// 		log.Println(err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "text/html")
// 	// w.Write([]byte(strings.Replace(string(rnd.indexHTML), "<!--app-html-->", output, 1)))
// 	w.Write([]byte(output))
// }

// Render renders the provided path to HTML.
func (r *Renderer) Render(urlPath string, data interface{}) (string, error) {
	iso := r.pool.Get()
	defer r.pool.Put(iso)

	ctx := v8go.NewContext(iso.Isolate)
	defer ctx.Close()

	_, err := iso.RenderScript.Run(ctx)
	if err != nil {
		if jsErr, ok := err.(*v8go.JSError); ok {
			err = fmt.Errorf("%v", jsErr.StackTrace)
			if err != nil {
				return "", err
			}
		}
		return "", nil
	}

	b, err := json.Marshal(data)

	if err != nil {
		return "", err
	}
	ctx.Global().Set("payload", string(b))

	renderCmd := fmt.Sprintf("ssrRender(`%s`, `%s`, `%s`)", urlPath, r.clientScript, r.clientCss)
	val, err := ctx.RunScript(renderCmd, r.ssrScriptName)
	if err != nil {
		if jsErr, ok := err.(*v8go.JSError); ok {
			err = fmt.Errorf("%v", jsErr.StackTrace)
			if err != nil {
				return "", err
			}
		}
		return "", nil
	}
	result, err := resolvePromise(ctx, val, err)
	if err != nil {
		if jsErr, ok := err.(*v8go.JSError); ok {
			err = fmt.Errorf("%v", jsErr.StackTrace)
			if err != nil {
				return "", err
			}
		}
		return "", err
	}
	return result.String(), nil
}

func resolvePromise(ctx *v8go.Context, val *v8go.Value, err error) (*v8go.Value, error) {
	if err != nil || !val.IsPromise() {
		return val, err
	}
	for {
		switch p, _ := val.AsPromise(); p.State() {
		case v8go.Fulfilled:
			return p.Result(), nil
		case v8go.Rejected:
			return nil, errors.New(p.Result().DetailString())
		case v8go.Pending:
			ctx.PerformMicrotaskCheckpoint() // run VM to make progress on the promise
			// go round the loop again...
		default:
			return nil, fmt.Errorf("illegal v8go.Promise state %d", p) // unreachable
		}
	}
}

func readFSFile(f fs.FS, name string) ([]byte, error) {
	file, err := f.Open(name)
	if err != nil {
		return nil, err
	}

	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return contents, nil
}

// Function to extract the script src from the provided HTML
func extractScriptSrc(htmlData string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlData))
	if err != nil {
		return "", err
	}

	var scriptSrc string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					scriptSrc = attr.Val
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if scriptSrc == "" {
		return "", fmt.Errorf("script src not found")
	}

	return scriptSrc, nil
}

// Function to extract script src and link href from the provided HTML
func extractAssets(htmlData string) (string, string, error) {
	doc, err := html.Parse(strings.NewReader(htmlData))
	if err != nil {
		return "", "", err
	}

	var scriptSrc, cssHref string
	var f func(*html.Node)
	f = func(n *html.Node) {
		// Extract <script> tag with "src" attribute
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					scriptSrc = attr.Val
				}
			}
		}

		// Extract <link> tag with "href" attribute for stylesheets
		if n.Type == html.ElementNode && n.Data == "link" {
			isStylesheet := false
			for _, attr := range n.Attr {
				if attr.Key == "rel" && attr.Val == "stylesheet" {
					isStylesheet = true
				}
				if isStylesheet && attr.Key == "href" {
					cssHref = attr.Val
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if scriptSrc == "" {
		return "", "", fmt.Errorf("no script found")
	}

	return scriptSrc, cssHref, nil
}
