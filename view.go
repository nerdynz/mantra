package mantra

import (
	"bytes"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	pdf "github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/nerdynz/datastore"

	"github.com/unrolled/render"
)

// ViewBucket handles template rendering and data management
type ViewBucket struct {
	renderer *render.Render
	store    *datastore.Datastore
	w        http.ResponseWriter
	req      *http.Request
	data     map[string]interface{}
}

var funcmap = map[string]any{}

func AddTemplateFunc(name string, fn any) {
	funcmap[name] = fn
}

// NewViewBucket creates a new ViewBucket instance
func NewViewBucket(w http.ResponseWriter, req *http.Request, store *datastore.Datastore) *ViewBucket {
	vb := &ViewBucket{
		w:     w,
		req:   req,
		store: store,
		data:  make(map[string]interface{}),
	}

	// Create a new renderer instance with default options
	vb.renderer = render.New(render.Options{
		Directory:  "templates", // default templates directory
		Extensions: []string{".html", ".tmpl"},
		Layout:     "", // default layout file
		Funcs: []template.FuncMap{
			helperFuncs,
			funcmap,
		},
	})

	vb.Add("Now", time.Now())
	vb.Add("Year", time.Now().Year())
	return vb
}

// Add adds a key-value pair to the view data
func (vb *ViewBucket) Add(key string, value interface{}) {
	vb.data[key] = value
}

// HTML renders an HTML template with the current data
func (vb *ViewBucket) HTML(status int, templateName string, layout ...string) {
	if vb.renderer == nil {
		vb.ErrorHTML(http.StatusInternalServerError, "Renderer not set", errors.New("renderer not set"))
		return
	}
	if vb.store == nil {
		vb.ErrorHTML(http.StatusInternalServerError, "Store not set", errors.New("store not set"))
		return
	}
	if len(layout) > 0 {
		vb.renderer.HTML(vb.w, status, templateName, vb.data, render.HTMLOptions{
			Layout: layout[0],
		})
	} else {
		vb.renderer.HTML(vb.w, status, templateName, vb.data)
	}
}

// Text renders plain text
func (vb *ViewBucket) Text(status int, text string) {
	if vb.renderer == nil {
		vb.ErrorText(http.StatusInternalServerError, "Renderer not set", errors.New("renderer not set"))
		return
	}
	if vb.store == nil {
		vb.ErrorText(http.StatusInternalServerError, "Store not set", errors.New("store not set"))
		return
	}
	vb.renderer.Text(vb.w, status, text)
}

// JSON renders JSON data
func (vb *ViewBucket) JSON(status int, data interface{}) {
	vb.renderer.JSON(vb.w, status, data)
}

// createErrorData creates a standardized error data structure
func (vb *ViewBucket) createErrorData(friendly string, errs ...error) *errorData {
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
		// notice that we're using 1, so it will actually log the where
		// the error happened, 0 = this function, we don't want that.
		pc, file, line, _ := runtime.Caller(2)
		lineNumber = line
		funcName = runtime.FuncForPC(pc).Name()
		fileName = file
	}

	return &errorData{
		friendly,
		errStr,
		lineNumber,
		funcName,
		fileName,
	}
}

// Error handles errors with appropriate response type based on Accept header
func (vb *ViewBucket) Error(status int, friendly string, errs ...error) {
	accept := vb.req.Header.Get("Accept")
	switch {
	case strings.Contains(accept, "text/html"):
		vb.ErrorHTML(status, friendly, errs...)
	case strings.Contains(accept, "application/json"):
		vb.ErrorJSON(status, friendly, errs...)
	default:
		vb.ErrorText(status, friendly, errs...)
	}
}

// ErrorHTML renders an error template
func (vb *ViewBucket) ErrorHTML(status int, friendly string, errs ...error) {
	data := vb.createErrorData(friendly, errs...)
	slog.Error(data.nicelyFormatted())

	vb.Add("FriendlyError", data.Friendly)
	vb.Add("NastyError", data.Error)
	vb.Add("LineNumber", data.LineNumber)
	vb.Add("FuncName", data.FunctionName)
	vb.Add("FileName", data.FileName)
	vb.Add("ErrorCode", status)
	vb.HTML(status, "error")
}

// ErrorText renders error as plain text
func (vb *ViewBucket) ErrorText(status int, friendly string, errs ...error) {
	data := vb.createErrorData(friendly, errs...)
	slog.Error(data.nicelyFormatted())
	vb.Text(status, data.nicelyFormatted())
}

// ErrorJSON renders error as JSON
func (vb *ViewBucket) ErrorJSON(status int, friendly string, errs ...error) {
	data := vb.createErrorData(friendly, errs...)
	slog.Error(data.nicelyFormatted())
	vb.JSON(status, data)
}

// Redirect performs an HTTP redirect
func (vb *ViewBucket) Redirect(newUrl string, status int) {
	if status == 301 || status == 302 || status == 303 || status == 304 || status == 401 {
		http.Redirect(vb.w, vb.req, newUrl, status)
		return
	}
	vb.ErrorHTML(http.StatusInternalServerError, "Invalid Redirect", nil)
}

// SendData sends raw data with a specific content type
func (vb *ViewBucket) SendData(status int, bytes []byte, mime string) {
	vb.w.Header().Add("content-type", mime)
	vb.w.WriteHeader(status)
	vb.w.Write(bytes)
}

// File sends a file as an attachment
func (vb *ViewBucket) File(bytes []byte, filename string, mime string) {
	vb.w.Header().Set("Content-Type", mime)
	vb.w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	vb.w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
	vb.w.Write(bytes)
}

// InlineFile sends a file to be displayed inline
func (vb *ViewBucket) InlineFile(bytes []byte, filename string, mime string) {
	vb.w.Header().Set("Content-Type", mime)
	vb.w.Header().Set("Content-Disposition", `inline; filename="`+filename+`"`)
	vb.w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
	vb.w.Write(bytes)
}

// PDF sends a PDF file
func (vb *ViewBucket) PDF(bytes []byte) {
	vb.w.Header().Set("Content-Type", "application/PDF")
	vb.w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
	vb.w.Write(bytes)
}

// Excel sends an Excel file
func (vb *ViewBucket) Excel(bytes []byte, filename string) {
	vb.w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	vb.w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	vb.w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
	vb.w.Write(bytes)
}

type PDFParams struct {
	Url                 string `protobuf:"bytes,1,opt,name=url,proto3" json:"url,omitempty"`
	Delay               int32  `protobuf:"varint,2,opt,name=delay,proto3" json:"delay,omitempty"`
	JavascriptReadyFlag string `protobuf:"bytes,3,opt,name=javascriptReadyFlag,proto3" json:"javascriptReadyFlag,omitempty"`
	IsDebug             bool   `protobuf:"varint,4,opt,name=isDebug,proto3" json:"isDebug,omitempty"`
	IsMarginless        bool   `protobuf:"varint,5,opt,name=isMarginless,proto3" json:"isMarginless,omitempty"`
	IsLandscape         bool   `protobuf:"varint,6,opt,name=isLandscape,proto3" json:"isLandscape,omitempty"`
}

func (vb *ViewBucket) TemplatePDF(templateName string, params *PDFParams) {
	// write to a buffer
	buf := bytes.NewBuffer(nil)
	err := vb.renderer.HTML(buf, http.StatusOK, templateName, vb.data)
	if err != nil {
		vb.ErrorHTML(http.StatusInternalServerError, "Error rendering template", err)
		return
	}

	pdfg, err := pdf.NewPDFGenerator()
	if err != nil {
		vb.ErrorHTML(http.StatusInternalServerError, "Error creating PDF generator", err)
		return
	}

	pdfg.Dpi.Set(300)
	pdfg.NoCollate.Set(false)
	pdfg.PageSize.Set(pdf.PageSizeA4)

	if params.IsMarginless {
		pdfg.MarginTop.Set(0)
		pdfg.MarginLeft.Set(0)
		pdfg.MarginBottom.Set(0)
		pdfg.MarginRight.Set(0)
	}

	if params.IsLandscape {
		pdfg.Orientation.Set("landscape")
	}

	page := pdf.NewPage(buf.String())
	if params.IsDebug {
		page.DebugJavascript.Set(true)
	}

	readyFlag := params.JavascriptReadyFlag
	if readyFlag == "" {
		delay := params.Delay
		if delay == 0 {
			page.JavascriptDelay.Set(250)
		} else {
			page.JavascriptDelay.Set(uint(delay))
		}
	} else {
		page.WindowStatus.Set(readyFlag)
	}
	page.NoStopSlowScripts.Set(true)
	pdfg.AddPage(page)

	// Create PDF document in internal buffer
	err = pdfg.Create()
	if err != nil {
		vb.ErrorHTML(http.StatusInternalServerError, "Error creating PDF generator", err)
		return
	}

	bts := pdfg.Bytes()

	vb.w.Header().Set("Content-Type", "application/PDF")
	vb.w.Header().Set("Content-Length", strconv.Itoa(len(bts)))
	vb.w.Write(bts)
}
