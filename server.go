package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type Server struct {
	Exporter *Exporter
}

type ResizeRequestBody struct {
	URL          string  `json:"url"`
	Sizes        []*Size `json:"sizes"`
	Key          string  `json:"key"`
	Prefix       string  `json:"prefix"`
	SaveOriginal bool    `json:"save_original,omitempty"`
}

type ResponseBody struct {
	Status string               `json:"status"`
	Errors []*ResponseBodyError `json:"errors"`
}

type ResponseBodyError struct {
	Key     string `json:"key"`
	Message string `json:"error"`
}

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func NewResizeRequestBody() ResizeRequestBody {
	return ResizeRequestBody{
		SaveOriginal: true,
	}
}

func NewResponseError() *ResponseBody {
	return &ResponseBody{
		Status: "error",
	}
}

func NewServer() *Server {
	return &Server{
		Exporter: NewExporter(),
	}
}

func (s *Server) Start(ctx context.Context, port string) {
	prometheus.MustRegister(s.Exporter)

	http.HandleFunc("/resize", handler)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle(
		"/health", http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		),
	)
	app.Logger.Info(fmt.Sprintf("Starting resize server on port %s", port))
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	b := NewResizeRequestBody()
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		re := NewResponseError()
		re.Errors = []*ResponseBodyError{
			{
				Key:     "request",
				Message: fmt.Sprintf("error decoding JSON body: %s", err.Error()),
			},
		}
		jsonError(w, re, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	err = app.resize(&b, ctx)
	if err != nil {
		re := NewResponseError()
		switch err.(type) {
		case *ResizeErrors:
			for _, e := range err.(*ResizeErrors).Errors {
				re.Errors = append(
					re.Errors, &ResponseBodyError{
						Key:     e.Key,
						Message: e.Err.Error(),
					},
				)
			}
			jsonError(w, re, http.StatusConflict)
			return
		default:
			re := &ResponseBody{
				Status: "error",
				Errors: []*ResponseBodyError{
					{
						Key:     "unknown",
						Message: err.Error(),
					},
				},
			}
			jsonError(w, re, http.StatusConflict)
		}
		return
	}

	fmt.Fprintf(w, fmt.Sprintf("image '%s' resized successfully", b.Key))
}

func jsonError(w http.ResponseWriter, re *ResponseBody, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(re)
}
