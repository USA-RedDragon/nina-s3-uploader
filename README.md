# N.I.N.A. S3 Uploader

This project is a simple Go program that will watch for files in a directory and upload them to an S3 bucket. It is designed to be used with the [N.I.N.A.](https://nighttime-imaging.eu/) astrophotography software, but can be used with any software that can save files to a directory.

It is designed to only remove the previous file after the new file has been uploaded. This is to prevent the file from being deleted from the disk before N.I.N.A. or the user is done with it.

On Windows, I use this with WinFsp MemFs (particularly <https://github.com/Ceiridge/WinFsp-MemFs-Extended>) to create a virtual drive that N.I.N.A. can save files to. This program watches that directory and uploads the files to S3.

## Configuration

The service is configured via environment variables, a configuration YAML file, or command line flags. The [`config.example.yaml`](config.example.yaml) file shows the available configuration options. The command line flags match the schema of the YAML file, i.e. `--s3.endpoint='s3.amazonaws.com'` would equate to `s3.endpoint: "s3.amazonaws.com"`. Environment variables are in the same format, however they are uppercase and replace hyphens with underscores and dots with double underscores, i.e. `S3__ENDPOINT="s3.amazonaws.com"`.
