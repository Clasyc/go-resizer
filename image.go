package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"go.uber.org/zap"
	"io"
	"net/http"
	"sort"
	"strconv"
	"sync"
)

type MetaKey struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Key    string `json:"key"`
}

type Meta struct {
	Format      string     `json:"format"`
	ContentType string     `json:"content_type"`
	Width       int        `json:"width"`
	Height      int        `json:"height"`
	Keys        []*MetaKey `json:"keys"`
}

func (m *Meta) SortKeys() {
	sort.Slice(
		m.Keys, func(i, j int) bool {
			return m.Keys[i].Key < m.Keys[j].Key
		},
	)
}

// resize saves the resized images to the storage
func (a *Application) resize(req *ResizeRequestBody, ctx context.Context) (*Meta, error) {
	// download image
	in, err := a.blob(req.URL)
	if err != nil {
		app.Logger.Info("failed to download image", zap.String("url", req.URL), zap.Error(err))
		return nil, err
	}

	// retrieve meta data about original image
	meta, err := a.metadata(ctx, in)
	if err != nil {
		app.Logger.Info("failed to retrieve image metadata", zap.String("url", req.URL), zap.Error(err))
		return nil, err
	}

	orgKey := a.path(req)
	meta.Keys = append(
		meta.Keys, &MetaKey{
			Width:  meta.Width,
			Height: meta.Height,
			Key:    orgKey,
		},
	)

	// save original image
	if req.SaveOriginal {
		label, err := app.save(ctx, in, nil, orgKey)

		if err != nil {
			app.Logger.Info(
				"failed to save image",
				zap.String("image", label),
				zap.String("key", req.Key),
				zap.Error(err),
			)
			return nil, err
		}

		app.UpResized(label)
	}

	errs := NewResizeErrors()
	var wg sync.WaitGroup
	wg.Add(len(req.Sizes))

	// save resized images
	for _, size := range req.Sizes {
		go func(wg *sync.WaitGroup, mu *sync.Mutex, size *Size) {
			defer wg.Done()
			s3key := fmt.Sprintf("%s/%s/%s_%dx%d.webp", app.Prefix, req.Prefix, req.Key, size.Width, size.Height)

			if meta.Width < size.Width && meta.Height < size.Height {
				return
			}

			label, err := app.save(ctx, in, size, s3key)
			if err != nil {
				mu.Lock()
				errs.Add(err.(*ResizeError))
				mu.Unlock()
				return
			}

			meta.Keys = append(
				meta.Keys, &MetaKey{
					Width:  size.Width,
					Height: size.Height,
					Key:    s3key,
				},
			)

			app.UpResized(label)
		}(&wg, app.Mutex, size)
	}
	wg.Wait()

	if len(errs.Errors) > 0 {
		return nil, errs
	}
	meta.SortKeys()

	return meta, nil
}

func (a *Application) metadata(ctx context.Context, in *imagor.Blob) (*Meta, error) {
	blob, err := a.Imagor.ServeBlob(
		ctx, in, imagorpath.Params{
			Meta: true,
		},
	)

	r, _, err := blob.NewReader()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	meta := Meta{
		Keys: make([]*MetaKey, 0),
	}
	err = json.NewDecoder(r).Decode(&meta)
	if err != nil {
		return nil, err
	}

	return &meta, nil
}

func (a *Application) blob(url string) (*imagor.Blob, error) {
	var in *imagor.Blob
	err := retry.Do(
		func() error {
			var e error
			in = imagor.NewBlob(
				func() (reader io.ReadCloser, size int64, err error) {
					var resp *http.Response
					if resp, err = a.Client.Get(url); err != nil {
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

	return in, err
}

func (a *Application) save(ctx context.Context, in *imagor.Blob, size *Size, key string) (string, error) {
	label := "original"
	params := imagorpath.Params{
		FitIn: true,
		Filters: []imagorpath.Filter{
			{"format", "webp"},
		},
	}

	if size != nil {
		label = fmt.Sprintf("%dx%d", size.Width, size.Height)
		params.Width = size.Width
		params.Height = size.Height
	}

	out, err := a.Imagor.ServeBlob(ctx, in, params)
	if err != nil {
		return label, &ResizeError{
			Key: label,
			Err: err,
		}
	}
	err = retry.Do(
		func() error {
			return a.Storage.Put(ctx, key, out)
		},
		retry.Attempts(RetryAttempts),
		retry.Delay(RetryDelay),
	)
	if err != nil {
		return label, &ResizeError{
			Key: label,
			Err: err,
		}
	}

	return label, nil
}

func (a *Application) imageToBase64(ctx context.Context, url string, size *Size) (string, error) {
	blob, err := app.blob(url)
	if err != nil {
		app.Logger.Info("failed to download image", zap.String("url", url), zap.Error(err))
		return "", err
	}

	out, err := app.Imagor.ServeBlob(
		ctx, blob, imagorpath.Params{
			FitIn:  true,
			Width:  size.Width,
			Height: size.Height,
			Filters: []imagorpath.Filter{
				{"format", "webp"},
			},
		},
	)

	if err != nil {
		app.Logger.Info("failed to resize image for base64", zap.String("url", url), zap.Error(err))
		return "", err
	}

	reader, _, err := out.NewReader()
	if err != nil {
		panic(err)
	}

	bytes, err := io.ReadAll(reader)
	if err != nil {
		app.Logger.Info("failed to read bytes", zap.String("url", url), zap.Error(err))
		return "", err
	}

	base64Str := base64.StdEncoding.EncodeToString(bytes)
	app.UpResized("base64")

	return base64Str, nil
}

func (a *Application) path(req *ResizeRequestBody) string {
	return fmt.Sprintf("%s/%s/%s.webp", a.Prefix, req.Prefix, req.Key)
}
