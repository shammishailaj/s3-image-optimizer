# S3 Image Optimizer

A lambda function that will listen to the `ObjectCreated` event on an S3 bucket, if a valid image is uploaded, it will optimize the image and return it to a different, optimized folder.

### Configuration

Use these environment variables to customize the lambda function.

| Key          | Description                                                                                                     |
| ------------ | --------------------------------------------------------------------------------------------------------------- |
| BUCKET       | The path to the bucket the folders reside in                                                                    |
| FROM_BUCKET  | The folder within the bucket that will be listened to for `ObjectCreated` events                                |
| TO_BUCKET    | The folder that the optimised images will be sent to                                                            |
| JPEG_QUALITY | Integer between 0 - 100 to dictate the level of compression (https://golang.org/pkg/image/jpeg/#Options)        |
| PNG_QUALITY  | Integer between -3 - 0 to dictate the level of compression (https://golang.org/pkg/image/png/#CompressionLevel) |
