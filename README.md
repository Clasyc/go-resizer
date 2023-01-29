# Go image resizer based on [Imagor](https://github.com/cshum/imagor)
This app will resize image to multiple images for each given size. 
Output images will be saved to S3 bucket as `webp` format.
> NOTE: If original image size is smaller than given size, it will be skipped.


## How to use

| Env variable          | Example                    | Description                                                                                                                                    |
|-----------------------|----------------------------|------------------------------------------------------------------------------------------------------------------------------------------------|
| S3_BUCKET             | my-bucket                  | S3 bucket name where content will be saved                                                                                                     |
| S3_REGION             | eu-west-1                  | AWS S3 bucket region name                                                                                                                      |
| AWS_ACCESS_KEY_ID     | AKIAEXAMPLEVALUE           | AWS credentials ID                                                                                                                             |
| AWS_SECRET_ACCESS_KEY | BypKqExaMpLeExamPleExample | AWS credentials secret                                                                                                                         |
| FALLBACK_FORMAT       | jpg                        | The image format which will be saved as alternative to webp for fallback support, if not set it will be skipped. Possible values: `jpg`, `png` |
| FALLBACK_SIZE         | original                   | The size which will be used for fallback image, example `512x512`, to keep original size use `original`                                        |

### Resize
1. Run `docker build -t resizer .`
2. Run 
    ```
    docker run -p 8001:8000 -e S3_BUCKET=bucket-name -e S3_REGION=eu-west-1 -e AWS_ACCESS_KEY_ID=access-key-id -e AWS_SECRET_ACCESS_KEY=secret-access-key -e FALLBACK_FORMAT=jpg -e FALLBACK_SIZE=original resizer
    ```
3. Send a POST request:

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
                    "key": "gopher/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f.webp"
                },
                {
                    "width": 512,
                    "height": 512,
                    "key": "gopher/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f_512x512.webp"
                },
                {
                    "width": 100,
                    "height": 100,
                    "key": "gopher/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f_100x100.webp"
                },
                {
                    "width": 200,
                    "height": 200,
                    "key": "gopher/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f_200x200.webp"
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