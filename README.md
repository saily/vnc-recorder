# VNC recorder

> this is wip, don't use in production!

Record [VNC] screens to mp4 video using [ffmpeg]. Thanks to
[amitbet for providing his vnc2video](https://github.com/amitbet/vnc2video)
library which made this wrapper possible.

## Use

    docker run -it docker run ghcr.io/aluvare/vnc-recorder/vnc-recorder:<release> --help

    NAME:
       vnc-recorder - Connect to a vnc server and record the screen to a video.

    USAGE:
       vnc-recorder [global options] command [command options] [arguments...]

    VERSION:
       0.3.0

    AUTHOR:
       Daniel Widerin <daniel@widerin.net>

    COMMANDS:
         help, h  Shows a list of commands or help for one command

    GLOBAL OPTIONS:
       --ffmpeg value              Which ffmpeg executable to use (default: "ffmpeg") [$VR_FFMPEG_BIN]
       --host value                VNC host (default: "localhost") [$VR_VNC_HOST]
       --port value                VNC port (default: 5900) [$VR_VNC_PORT]
       --password value            Password to connect to the VNC host (default: "secret") [$VR_VNC_PASSWORD]
       --framerate value           Framerate to record (default: 30) [$VR_FRAMERATE]
       --crf value                 Constant Rate Factor (CRF) to record with (default: 35) [$VR_CRF]
       --outfile value             Output file to record to. (default: "output") [$VR_OUTFILE]
       --splitfile value           Mins to split file. (default: 0) [$VR_SPLIT_OUTFILE]
       --s3_endpoint value         S3 endpoint. [$VR_S3_ENDPOINT]
       --s3_accessKeyID value      S3 access key id. [$VR_S3_ACCESSKEY]
       --s3_secretAccessKey value  S3 secret access key. [$VR_S3_SECRETACCESSKEY]
       --s3_bucketName value       S3 bucket name. [$VR_S3_BUCKETNAME]
       --s3_region value           S3 region. (default: "us-east-1") [$VR_S3_REGION]
       --s3_ssl                    S3 SSL. (default: false) [$VR_S3_SSL]
       --help, -h                  show help (default: false)
       --version, -v               print the version (default: false)

**Note:** If you run vnc-recorder from your command line and don't use [docker]
you might want to customize the `--ffmpeg` flag to point to an existing
[ffmpeg] installation.


## Build

    docker build -t yourbuild .
    docker run -it yourbuild --help

## TODO

- [ ] Add tests!
- [ ] Add more encoder options
- [ ] Get some patches merged for our dependencies

[ffmpeg]: https://ffmpeg.org
[docker]: https://www.docker.com
[vnc]: https://en.wikipedia.org/wiki/Virtual_Network_Computing
