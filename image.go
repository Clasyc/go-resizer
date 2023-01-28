package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"sync"
)

type Meta struct {
	Format      string `json:"format"`
	ContentType string `json:"content_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// resize saves the resized images to the storage
func (a *Application) resize(req *ResizeRequestBody, ctx context.Context) error {
	var in *imagor.Blob
	err := retry.Do(
		func() error {
			var e error
			in = imagor.NewBlob(
				func() (reader io.ReadCloser, size int64, err error) {
					var resp *http.Response
					if resp, err = a.Client.Get(req.URL); err != nil {
						e = err
						return
					}
					reader = resp.Body
					size, _ = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
					return
				},
			)
			return e
		},
		retry.Attempts(RetryAttempts),
		retry.Delay(RetryDelay),
	)

	if err != nil {
		app.Logger.Info("failed to download image", zap.String("url", req.URL), zap.Error(err))
		app.UpFailures()
		return err
	}

	// retrieve meta data about original image
	blob, err := a.Imagor.ServeBlob(
		ctx, in, imagorpath.Params{
			Meta: true,
		},
	)

	r, _, err := blob.NewReader()
	if err != nil {
		app.Logger.Info("failed to read blob", zap.Error(err))
		app.UpFailures()
		return err
	}
	defer r.Close()

	meta := Meta{}
	err = json.NewDecoder(r).Decode(&meta)
	if err != nil {
		app.Logger.Info("failed to decode meta", zap.Error(err))
		app.UpFailures()
		return err
	}

	// save original image
	if req.SaveOriginal {
		out, err := a.Imagor.ServeBlob(
			ctx, in, imagorpath.Params{
				FitIn: true,
				Filters: []imagorpath.Filter{
					{"format", "webp"},
				},
			},
		)
		if err != nil {
			app.Logger.Info(
				"failed to serve image",
				zap.String("image", "original"),
				zap.String("key", req.Key),
				zap.Error(err),
			)
			app.UpFailures()
			return &ResizeError{
				Key: "original",
				Err: err,
			}
		}
		err = retry.Do(
			func() error {
				return a.Storage.Put(ctx, fmt.Sprintf("%s/%s/%s.webp", app.Prefix, req.Prefix, req.Key), out)
			},
			retry.Attempts(RetryAttempts),
			retry.Delay(RetryDelay),
		)
		if err != nil {
			app.Logger.Info(
				"failed to save image",
				zap.String("image", "original"),
				zap.String("key", req.Key),
				zap.Error(err),
			)
			app.UpFailures()
			return &ResizeError{
				Key: "original",
				Err: err,
			}
		}

		app.UpResized("original")
	}

	errs := NewResizeErrors()
	var wg sync.WaitGroup
	wg.Add(len(req.Sizes))

	// save resized images
	for _, size := range req.Sizes {
		go func(wg *sync.WaitGroup, mu *sync.Mutex, size *Size) {
			defer wg.Done()
			s3key := fmt.Sprintf("%s/%s/%s_%dx%d.webp", app.Prefix, req.Prefix, req.Key, size.Width, size.Height)
			sl := fmt.Sprintf("%dx%d", size.Width, size.Height)
			out, err := a.Imagor.ServeBlob(
				ctx, in, imagorpath.Params{
					Width:  size.Width,
					Height: size.Height,
					FitIn:  true,
					Filters: []imagorpath.Filter{
						{"format", "webp"},
					},
				},
			)
			if err != nil {
				mu.Lock()
				app.Logger.Info(
					"failed to serve image",
					zap.String("image", sl),
					zap.String("key", s3key),
					zap.Error(err),
				)
				app.UpFailures()
				errs.Add(
					&ResizeError{
						Key: sl,
						Err: err,
					},
				)
				mu.Unlock()
				return
			}

			// skip if the size is larger than the original image
			if meta.Width < size.Width && meta.Height < size.Height {
				return
			}

			err = retry.Do(
				func() error {
					return a.Storage.Put(
						ctx, s3key, out,
					)
				},
				retry.Attempts(RetryAttempts),
				retry.Delay(RetryDelay),
			)
			if err != nil {
				mu.Lock()
				app.Logger.Info(
					"failed to save image",
					zap.String("image", sl),
					zap.String("key", s3key),
					zap.Error(err),
				)
				app.UpFailures()
				errs.Add(
					&ResizeError{
						Key: sl,
						Err: err,
					},
				)
				mu.Unlock()
			}

			app.UpResized(sl)
		}(&wg, app.Mutex, size)
	}
	wg.Wait()

	if len(errs.Errors) > 0 {
		return errs
	}

	return nil
}
