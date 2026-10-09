# N.I.N.A. S3 Uploader

This project is a simple Go program that will watch for files in a directory and upload them to an S3 bucket. It is designed to be used with the [N.I.N.A.](https://nighttime-imaging.eu/) astrophotography software, but can be used with any software that can save files to a directory.

On Windows, I use this with WinFsp MemFs (particularly <https://github.com/Ceiridge/WinFsp-MemFs-Extended>) to create a virtual drive that N.I.N.A. can save files to. This program watches that directory and uploads the files to S3. This allows me to use a much smaller disk and prevents excessive writes as my system is deployed at a remote observatory.

## Configuration

The uploader reads `config.yaml` from the working directory if it exists, or the file passed with `--config`/`-c`. [`config.example.yaml`](config.example.yaml) lists every option. Environment variables override the file and flags override both. List values such as `uploader.extensions` are comma-separated in an environment variable (`UPLOADER__EXTENSIONS=.fits,.xisf`) and repeated or comma-separated on the command line. AWS credentials come from the usual AWS environment variables or shared config files.

<!-- configulator:begin -->

| Key                        | Type           | Default     | Environment                  | Flag                         | Description                                                                                                                                |
|----------------------------|----------------|-------------|------------------------------|------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------|
| `log-level`                | string         | `info`      | `LOG_LEVEL`                  | `--log-level`                | Log level, one of debug, info, warn or error                                                                                               |
| `s3.region`                | string         | `us-east-1` | `S3__REGION`                 | `--s3.region`                | The region to use                                                                                                                          |
| `s3.bucket`                | string         |             | `S3__BUCKET`                 | `--s3.bucket`                | The bucket to upload to (required)                                                                                                         |
| `s3.prefix`                | string         | `/`         | `S3__PREFIX`                 | `--s3.prefix`                | The prefix to use for the uploaded files                                                                                                   |
| `s3.endpoint`              | string         |             | `S3__ENDPOINT`               | `--s3.endpoint`              | A custom S3-compatible endpoint URL, such as https://minio.example.com. Empty uses AWS                                                     |
| `uploader.directory`       | string         |             | `UPLOADER__DIRECTORY`        | `--uploader.directory`       | The directory to watch for new files (required)                                                                                            |
| `uploader.extensions`      | list of string |             | `UPLOADER__EXTENSIONS`       | `--uploader.extensions`      | The file extensions to watch for, such as .fits. Comma-separated in an environment variable (required)                                     |
| `uploader.local.directory` | string         |             | `UPLOADER__LOCAL__DIRECTORY` | `--uploader.local.directory` | Files are only stored here if they fail to upload to S3. Once a file uploads at a later time, it is deleted from this directory (required) |
| `uploader.delay`           | string         |             | `UPLOADER__DELAY`            | `--uploader.delay`           | How long to wait after a file is uploaded or moved to the local directory before removing it from the watched directory, such as 30s       |

<!-- configulator:end -->
