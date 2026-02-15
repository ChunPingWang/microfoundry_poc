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

//go:embed static/templates/*.html static/templates/partials/*.html
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
	}

	tmpl := template.Must(
		template.New("").Funcs(funcMap).ParseFS(
			templateFiles,
			"static/templates/*.html",
			"static/templates/partials/*.html",
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
