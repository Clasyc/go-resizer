# Go image resizer based on [Imagor](https://github.com/cshum/imagor)
This app will resize image to multiple images for each given size. 
Output images will be saved to S3 bucket as `webp` format.
> NOTE: If original image size is smaller than given size, it will be skipped.


## How to use

1. Run `docker build -t resizer .`
2. Run `docker run -p 8080:8080 -e S3_BUCKET=bucket-name -e S3_REGION=eu-west-1 -e S3_BUCKET_PREFIX=images -e AWS_ACCESS_KEY_ID=access-key-id -e AWS_SECRET_ACCESS_KEY=secret-access-key resizer`
3. Send a POST request to `http://localhost:8080/resize` with the following body:

```json
{
  "url": "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png",
  "sizes": [
    {
      "width": 100,
      "height": 100
    },
    {
      "width": 200,
      "height": 200
    }
  ],
  "prefix": "example",
  "key": "e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f",
  "save_original": true
}
```

This will resize given image to 100x100 and 200x200 and save it to S3 bucket with the following paths:
* `images/example/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f.webp`
* `images/example/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f_100x100.webp`
* `images/example/e5f9f1b0-7b1f-4b2f-8f1c-1c1f1f1f1f1f_200x200.webp`