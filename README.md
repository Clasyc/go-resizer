# Go image resizer based on [Imagor](https://github.com/cshum/imagor)
This app will resize image to multiple images for each given size. 
Output images will be saved to S3 bucket as `webp` format.
> NOTE: If original image size is smaller than given size, it will be skipped.


## How to use

### Resize
1. Run `docker build -t resizer .`
2. Run `docker run -p 8000:8000 -e S3_BUCKET=bucket-name -e S3_REGION=eu-west-1 -e S3_BUCKET_PREFIX=images -e AWS_ACCESS_KEY_ID=access-key-id -e AWS_SECRET_ACCESS_KEY=secret-access-key resizer`
3. Send a POST request to:

```
curl --location --request POST 'http://localhost:8000/resize' \
--header 'Content-Type: application/json' \
--data-raw '{
    "url": "https://technology.riotgames.com/sites/default/files/articles/116/golangheader.png",
    "sizes": [
        {
            "width": 100,
            "height": 100
        },
        {
            "width": 200,
            "height": 200
        },
        {
            "width": 512,
            "height": 512
        },
        {
            "width": 2048,
            "height": 2048
        },
        {
            "width": 4096,
            "height": 4096
        }
    ],
    "key": "e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f",
    "prefix": "gopher",
    "save_original": true
}'
```

```
{
    "status": "ok",
    "data": {
        "format": "png",
        "content_type": "image/png",
        "width": 1999,
        "height": 758,
        "keys": [
            {
                "width": 1999,
                "height": 758,
                "key": "gotest/gopher/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f.webp"
            },
            {
                "width": 512,
                "height": 512,
                "key": "gotest/gopher/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f_512x512.webp"
            },
            {
                "width": 100,
                "height": 100,
                "key": "gotest/gopher/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f_100x100.webp"
            },
            {
                "width": 200,
                "height": 200,
                "key": "gotest/gopher/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f_200x200.webp"
            }
        ]
    }
}
```

This will resize given image to 100x100 and 200x200 and save it to S3 bucket with the following paths:
* `images/example/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f.webp`
* `images/example/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f_100x100.webp`
* `images/example/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f_200x200.webp`
* `images/example/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f_512x512.webp`

### Prometheus metrics

Metrics are exposed on `/metrics` endpoint.