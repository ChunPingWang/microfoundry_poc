package admin

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

//go:embed static/templates/*.html static/templates/partials/*.html static/templates/tabs/*.html
var templateFiles embed.FS

//go:embed static/css/*
var cssFiles embed.FS

func staticFS() http.FileSystem {
	sub, _ := fs.Sub(cssFiles, "static")
	return http.FS(sub)
}

type TemplateRenderer struct {
	templates *template.Template
}

func NewTemplateRenderer() *TemplateRenderer {
	funcMap := template.FuncMap{
		"stateColor": func(state string) string {
			switch strings.ToLower(state) {
			case "started", "running":
				return "green"
			case "stopped", "down":
				return "red"
			case "starting", "staging", "deploying", "uploading":
				return "yellow"
			case "crashed", "failed":
				return "red"
			default:
				return "gray"
			}
		},
		"stateBg": func(state string) string {
			switch strings.ToLower(state) {
			case "started", "running":
				return "bg-green-100 text-green-800"
			case "stopped", "down":
				return "bg-red-100 text-red-800"
			case "starting", "staging", "deploying", "uploading":
				return "bg-yellow-100 text-yellow-800"
			case "crashed", "failed":
				return "bg-red-100 text-red-800"
			default:
				return "bg-gray-100 text-gray-800"
			}
		},
		"lifecycleBadge": func(lcType string) string {
			switch strings.ToLower(lcType) {
			case "docker":
				return "bg-blue-100 text-blue-800"
			case "buildpack":
				return "bg-green-100 text-green-800"
			case "cnb":
				return "bg-purple-100 text-purple-800"
			default:
				return "bg-gray-100 text-gray-800"
			}
		},
		"timeAgo": func(t time.Time) string {
			if t.IsZero() {
				return "-"
			}
			d := time.Since(t)
			switch {
			case d < time.Minute:
				return fmt.Sprintf("%ds ago", int(d.Seconds()))
			case d < time.Hour:
				return fmt.Sprintf("%dm ago", int(d.Minutes()))
			case d < 24*time.Hour:
				return fmt.Sprintf("%dh ago", int(d.Hours()))
			default:
				return fmt.Sprintf("%dd ago", int(d.Hours()/24))
			}
		},
		"join": strings.Join,
		"memFmt": func(mb int) string {
			if mb >= 1024 {
				return fmt.Sprintf("%.1fG", float64(mb)/1024)
			}
			return fmt.Sprintf("%dM", mb)
		},
		"isSensitive": func(key string) bool {
			upper := strings.ToUpper(key)
			for _, s := range []string{"PASSWORD", "SECRET", "KEY", "TOKEN", "CREDENTIALS"} {
				if strings.Contains(upper, s) {
					return true
				}
			}
			return false
		},
		"dict": func(values ...any) map[string]any {
			if len(values)%2 != 0 {
				return nil
			}
			d := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				d[key] = values[i+1]
			}
			return d
		},
	}

	tmpl := template.Must(
		template.New("").Funcs(funcMap).ParseFS(
			templateFiles,
			"static/templates/*.html",
			"static/templates/partials/*.html",
			"static/templates/tabs/*.html",
		),
	)

	return &TemplateRenderer{templates: tmpl}
}

func (tr *TemplateRenderer) Render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tr.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, fmt.Sprintf("template error: %v", err), http.StatusInternalServerError)
	}
}

func (tr *TemplateRenderer) RenderPartial(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tr.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, fmt.Sprintf("template error: %v", err), http.StatusInternalServerError)
	}
}
