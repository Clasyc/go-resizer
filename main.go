package main

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"github.com/cshum/imagor/storage/s3storage"
	"go.uber.org/zap"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	HttpClientTimeout = 5 * time.Second
	RetryAttempts     = 3
	RetryDelay        = 500 * time.Millisecond
)

var app *Application

type Application struct {
	Storage  *s3storage.S3Storage
	Imagor   *imagor.Imagor
	Client   *http.Client
	Logger   *zap.Logger
	Counters *Counters
	Mutex    *sync.Mutex
	Fallback Fallback
	Format   string
	DryRun   bool
}

func NewApplication(ctx context.Context) *Application {
	b := os.Getenv("S3_BUCKET")
	r := os.Getenv("S3_REGION")
	ff := os.Getenv("FALLBACK_FORMAT")
	fs := os.Getenv("FALLBACK_SIZE")
	fo := os.Getenv("FORMAT")
	e := os.Getenv("S3_ENDPOINT")
	dry := false

	var endpoint *string
	if e != "" {
		endpoint = &e
	}
	if fo == "" {
		fo = DefaultFormat
	}
	if v, err := strconv.ParseBool(os.Getenv("DRY_RUN")); err == nil && v {
		dry = v
	}

	img := imagor.New(
		imagor.WithLoaders(httploader.New()),
		imagor.WithProcessors(vipsprocessor.NewProcessor()),
	)
	if err := img.Startup(ctx); err != nil {
		panic(err)
	}
	img.AutoWebP = true
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	return &Application{
		Storage: NewStorage(b, r, endpoint),
		Imagor:  img,
		Client: &http.Client{
			Timeout: HttpClientTimeout,
		},
		Logger: logger,
		Counters: &Counters{
			Resized: make(map[string]int),
		},
		Mutex: &sync.Mutex{},
		Fallback: Fallback{
			Format: ff,
			Size:   NewSize(fs),
		},
		Format: fo,
		DryRun: dry,
	}
}

func main() {
	ctx := context.Background()
	app = NewApplication(ctx)
	server := NewServer()
	server.Start(ctx, "8000")
}
