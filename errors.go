package mantra

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
)

// errorData represents a standardized error response
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
	str += "FunctionName: \n\t" + e.FunctionName + "\n"
	return str
}

// errorOut handles error responses in a standardized way
func errorOut(w http.ResponseWriter, req *http.Request, status int, friendly string, errs ...error) {
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

	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
