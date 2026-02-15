package admin

import (
	"bufio"
	"fmt"
	"html"
	"net/http"
)

func (s *Server) LogStreamHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	stream, err := s.k8sClient.GetAppLogs(ctx, name, true)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", html.EscapeString(err.Error()))
		flusher.Flush()
		return
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			line := html.EscapeString(scanner.Text())
			fmt.Fprintf(w, "data: <div class=\"py-0.5\">%s</div>\n\n", line)
			flusher.Flush()
		}
	}
}
