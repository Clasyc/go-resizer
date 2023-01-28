package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func resizeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		meta, err := app.resize(&b, ctx)
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

		jsonResponse(
			w, &ResponseBody{
				Status: "ok",
				Data:   meta,
			}, http.StatusOK,
		)
	}
}

func base64Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		b64, err := app.imageToBase64(ctx, b.URL, b.Sizes[0])
		if err != nil {
			re := NewResponseError()
			re.Errors = []*ResponseBodyError{
				{
					Key:     "request",
					Message: err.Error(),
				},
			}
			jsonError(w, re, http.StatusBadRequest)
			return
		}

		jsonResponse(
			w, &ResponseBody{
				Status: "ok",
				Data:   b64,
			}, http.StatusOK,
		)
	}
}
