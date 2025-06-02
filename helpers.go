package mantra

import (
	"encoding/json"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/nerdynz/helpers"
)

var helperFuncs = template.FuncMap{
	"javascript":      javascriptTag,
	"stylesheet":      stylesheetTag,
	"javascriptAsync": javascriptTagAsync,
	"stylesheetAsync": stylesheetTagAsync,
	"image":           imageTag,
	"imagepath":       imagePath,
	"content":         content,
	"plainToHtml":     plainToHtml,
	"slugify":         slugify,
	"address":         address,
	"JSON": func(v interface{}) template.JS {
		a, _ := json.Marshal(v)
		return template.JS(a)
	},
	// "link":  link,
	"title": title,
	"year":  year,
	// "getPartialName": getPartialName,
	"hasValue":         hasValue,
	"isBlank":          isBlank,
	"isNotBlank":       isNotBlank,
	"formatDate":       formatDate,
	"formatDateLoc":    formatDateLoc,
	"formatDateStrLoc": formatDateStrLoc,
	"htmlsafe":         htmlSafe,
	"gt":               greaterThan,
	"replace":          replace,
	"pictureBox":       pictureBox,
	"icon":             icon,
	"currency":         currency,
	"padIntLeft":       padIntLeft,
	"padLeft":          padLeft,
	"firstName":        firstName,
}

func address(name string) template.HTML {
	return template.HTML(strings.Join(strings.Split(name, ","), "<br>"))
}

func firstName(name string) string {
	return strings.Split(name, " ")[0]
}

func currency(dec float64) string {
	return helpers.Currency(dec)
}

func padLeft(str string, len int, padChar string) string {
	return helpers.PadLeft(str, len, padChar)
}

func padIntLeft(num int, len int, padChar string) string {
	return helpers.PadIntLeft(num, len, padChar)
}

func pictureBox(str string, index int, w int, h int) template.HTML {
	if str == "" {
		return template.HTML("")
	}
	index = index - 1

	html := `	<div class="column is-4">
		<a href="javascript:void(0)" onclick="gallery.open(` + strconv.Itoa(index) + `);return false;" class="image" data-galleryitem='{"src": "` + imagePath(str) + `", "w": ` + strconv.Itoa(w) + `, "h": ` + strconv.Itoa(h) + `}'>
			<img src="` + imagePath(str) + `" />
		</a>
	</div>`
	return template.HTML(html)
}

func icon(str string, size string) template.HTML {
	html := `<span class="icon ` + size + `"><i class="far fa-` + str + `"></i></span>`
	return template.HTML(html)
}

func replace(str string, old string, new string) string {
	return strings.Replace(str, old, new, -1)
}

// func getPartialName(block *models.Block) string {
// 	return helpers.KebabCase(block.Type)
// }

func year() string {
	return strconv.Itoa(time.Now().Year())
}

func greaterThan(num int, amt int) bool {
	return num > amt
}

// func content(contents ...string) template.HTML {
// 	var str string
// 	for _, content := range contents {
// 		str += "<div class='standard'>" + content + "</standard>"
// 	}
// 	return template.HTML(str)
// }

func javascriptTag(names ...string) template.HTML {
	var str string
	for _, name := range names {
		if strings.HasPrefix(name, "http") {
			str += "<script src='" + name + ".js' type='text/javascript'></script>"
		} else {
			str += "<script src='/js/" + name + ".js' type='text/javascript'></script>"
		}
	}
	return template.HTML(str)
}

func javascriptTagAsync(names ...string) template.HTML {
	var str string
	for _, name := range names {
		href := ""
		if strings.HasPrefix(name, "http") {
			href = name + ".js"
		} else {
			href = "/js/" + name + ".js"
		}
		str += `<script type="text/javascript">(function () { head.load('` + href + `');})() </script>`
	}
	return template.HTML(str)
}

func stylesheetTagAsync(names ...string) template.HTML {
	var str string
	for _, name := range names {
		href := "/css/" + name + ".css"
		//str += `<script type="text/javascript">(function () { var rl = document.createElement('link'); rl.rel = 'stylesheet';rl.href = '` + href + `';var rh = document.getElementsByTagName('head')[0]; rh.parentNode.insertBefore(rl, rh);})();</script>`
		str += `<script type="text/javascript">(function () { head.load('` + href + `');})() </script>`
	}
	return template.HTML(str)
}

func stylesheetTag(names ...string) template.HTML {
	var str string
	for _, name := range names {
		str += "<link rel='stylesheet' href='/css/" + name + ".css' type='text/css' media='screen'  />\n"
	}
	return template.HTML(str)
}

func imagePath(n interface{}) string {
	name := n.(string)
	if strings.HasPrefix(name, "data:") {
		return name
	}
	if strings.HasPrefix(name, "/attachments") {
		return name
	}
	if strings.HasPrefix(name, "/images") {
		return name
	}
	if strings.HasPrefix(name, "/") {
		return name
	}
	return "/images/" + name
}

func imageTag(name interface{}, alt interface{}, class string) template.HTML {
	return template.HTML("<image src='" + imagePath(name) + "' alt='" + alt.(string) + "' class='" + class + "' />")
}

func plainToHtml(str string) template.HTML {
	str = strings.Replace(str, "\n", "<br>", -1)
	return template.HTML(str)
}
func htmlSafe(str string) template.HTML {
	return template.HTML(str)
}

func content(str string) template.HTML {
	return template.HTML("<div class='content'>" + str + "</div>")
}

func title(text string) string {
	return strings.Title(text)
}

func hasValue(val interface{}) bool {
	return val != nil && val != ""
}

func isBlank(str string) bool {
	return str == ""
}

func isNotBlank(str string) bool {
	return !isBlank(str)
}

func formatDate(t time.Time, layout string) string {
	return t.Format(layout)
}

func formatDateLoc(t time.Time, location string, layout string) string {
	loc, _ := time.LoadLocation(location)
	return t.In(loc).Format(layout)
}

func formatDateStrLoc(tStr string, location string, layout string) string {
	t, err := time.Parse(time.RFC3339, tStr)
	if err != nil {
		return err.Error()
	}
	return formatDateLoc(t, location, layout)
}

func slugify(str string) string {
	return helpers.Slugify(str)
}
