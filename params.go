package mantra

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-zoo/bone"
	"github.com/oklog/ulid"
)

// ParamSlice gets all values for a parameter as strings
func ParamSlice(req *http.Request, key string) ([]string, error) {
	err := req.ParseForm()
	if err != nil {
		return nil, err
	}
	return req.Form[key], nil
}

// IntParamSlice gets all values for a parameter as integers
func IntParamSlice(req *http.Request, key string) ([]int, error) {
	ints := make([]int, 0)
	vals, err := ParamSlice(req, key)
	if err != nil {
		return ints, err
	}
	for _, val := range vals {
		iVal, err := strconv.Atoi(val)
		if err != nil {
			return ints, err
		}
		ints = append(ints, iVal)
	}
	return ints, nil
}

// ULIDParam gets and validates a ULID parameter
func ULIDParam(req *http.Request, key string) (string, error) {
	ul := strings.ToUpper(Param(req, key))
	id, err := ulid.Parse(ul)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// Param gets a single string parameter from either route params or query string
func Param(req *http.Request, key string) string {
	// try route param
	value := bone.GetValue(req, key)
	// try qs
	if value == "" {
		value = req.URL.Query().Get(key)
	}
	// do we have a value
	if value != "" {
		newValue, err := url.QueryUnescape(value)
		if err == nil {
			value = strings.Replace(newValue, "%20", " ", -1)
		}
	}
	return value
}

// IntParam gets a single integer parameter
func IntParam(req *http.Request, key string) (int, error) {
	return strconv.Atoi(Param(req, key))
}

// DateParam gets a date parameter in RFC3339Nano format
func DateParam(req *http.Request, key string) (time.Time, error) {
	var dt = Param(req, key)
	return time.Parse(time.RFC3339Nano, dt)
}

// ShortDateParam gets a date parameter in "20060102" format
func ShortDateParam(req *http.Request, key string) (time.Time, error) {
	var dt = Param(req, key)
	return time.Parse("20060102", dt)
}

// BoolParam gets a boolean parameter with multiple truthy values
func BoolParam(req *http.Request, key string) bool {
	val := Param(req, key)
	if val == "true" {
		return true
	}
	if val == "yes" {
		return true
	}
	if val == "1" {
		return true
	}
	if val == "y" {
		return true
	}
	if val == "✓" {
		return true
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return false
	}
	return b
}

// IntParamWithDefault gets an integer parameter with a default value
func IntParamWithDefault(req *http.Request, key string, deefault int) int {
	val := Param(req, key)
	if val == "" {
		return deefault // default
	}
	c, err := strconv.Atoi(Param(req, key))
	if err != nil {
		return deefault // default
	}
	return c
}
