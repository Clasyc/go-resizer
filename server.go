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
	Key          string  `json:"key,omitempty"`
	Prefix       string  `json:"prefix,omitempty"`
	SaveOriginal bool    `json:"save_original,omitempty"`
}

type ResponseBody struct {
	Status string               `json:"status"`
	Data   interface{}          `json:"data,omitempty"`
	Errors []*ResponseBodyError `json:"errors,omitempty"`
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

	http.HandleFunc("/resize", resizeHandler())
	http.HandleFunc("/base64", base64Handler())
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

func jsonError(w http.ResponseWriter, re *ResponseBody, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(re)
}

func jsonResponse(w http.ResponseWriter, re *ResponseBody, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(re)
}
