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
	"strings"
	"sync"
	"time"
)

const DefaultFormat = "webp"

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

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Fallback struct {
	Format string
	Size   *Size
}

type ResizePlan struct {
	Blueprint []*ResizeParams
}

type ResizeParams struct {
	Size   *Size
	Format string
}

func NewSize(d string) *Size {
	s := strings.Split(d, "x")
	if len(s) != 2 {
		return nil
	}
	return &Size{
		Width:  atoi(s[0]),
		Height: atoi(s[1]),
	}
}

func (m *Meta) SortKeys() {
	sort.Slice(
		m.Keys, func(i, j int) bool {
			return m.Keys[i].Key < m.Keys[j].Key
		},
	)
}

// Resize saves the resized images to the storage
func (a *Application) Resize(req *ResizeRequestBody, ctx context.Context) (*Meta, error) {
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

	p := &ResizePlan{
		Blueprint: make([]*ResizeParams, 0),
	}

	// save fallback image
	if a.Fallback.Format != "" {
		p.Blueprint = append(
			p.Blueprint, &ResizeParams{
				Size:   a.Fallback.Size,
				Format: a.Fallback.Format,
			},
		)
	}

	// save webp image
	if req.SaveOriginal {
		p.Blueprint = append(
			p.Blueprint, &ResizeParams{
				Format: DefaultFormat,
			},
		)
	}

	// save resized images
	errs := NewResizeErrors()
	for _, size := range req.Sizes {
		p.Blueprint = append(
			p.Blueprint, &ResizeParams{
				Size:   size,
				Format: DefaultFormat,
			},
		)
	}

	// do resize and s3 API calls in go routines
	var wg sync.WaitGroup
	wg.Add(len(p.Blueprint))
	for _, params := range p.Blueprint {
		s3key := a.creates3Key(params.Size, params.Format, req.Key, req.Prefix)
		if params.Size == nil {
			params.Size = &Size{
				Width:  meta.Width,
				Height: meta.Height,
			}
		}
		meta.Keys = append(
			meta.Keys, &MetaKey{
				Width:  params.Size.Width,
				Height: params.Size.Height,
				Key:    s3key,
			},
		)
		go func(wg *sync.WaitGroup, mu *sync.Mutex, pr *ResizeParams) {
			defer wg.Done()
			if err := a.save(in, s3key, pr.Format, pr.Size, ctx); err != nil {
				mu.Lock()
				errs.Add(err.(*ResizeError))
				mu.Unlock()
			}
		}(&wg, app.Mutex, params)
	}
	wg.Wait()

	if len(errs.Errors) > 0 {
		return nil, errs
	}
	meta.SortKeys()

	return meta, nil
}

func (a *Application) creates3Key(size *Size, format, name, prefix string) string {
	var s3key string
	if size != nil {
		s3key = join(prefix, fmt.Sprintf("%s_%dx%d.%s", name, size.Width, size.Height, format))
	} else {
		s3key = join(prefix, name+"."+format)
	}
	return s3key
}

func (a *Application) save(
	in *imagor.Blob, s3key string, format string, size *Size, ctx context.Context,
) error {
	label, err := app.resize(ctx, in, size, format, s3key)

	if err != nil {
		app.Logger.Info(
			"failed to resize image",
			zap.String("image", label),
			zap.String("key", s3key),
			zap.Error(err),
		)
		return err
	}

	app.UpResized(label)

	return nil
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

func (a *Application) resize(ctx context.Context, in *imagor.Blob, size *Size, format, key string) (string, error) {
	label := "original"
	params := imagorpath.Params{
		FitIn: true,
		Filters: []imagorpath.Filter{
			{"format", format},
		},
	}

	if size != nil {
		label = fmt.Sprintf("%dx%d", size.Width, size.Height)
		params.Width = size.Width
		params.Height = size.Height
	}

	start := time.Now()
	out, err := a.Imagor.ServeBlob(ctx, in, params)
	if err != nil {
		return label, &ResizeError{
			Key: label,
			Err: err,
		}
	}
	elapsed := time.Since(start)
	app.Logger.Info(fmt.Sprintf("resize on %s, took %s", key, elapsed))
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

func join(s1, s2 string) string {
	var p string
	if s1 != "" {
		p += "/" + s1
	}
	if s2 != "" {
		p += "/" + s2
	}

	if p == "" {
		return ""
	}

	return strings.TrimPrefix(p, "/")
}

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
