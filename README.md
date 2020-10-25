# VNC recorder

> this is wip, don't use in production!

Record [VNC] screens to mp4 video using [ffmpeg]. Thanks to
[amitbet for providing his vnc2video](https://github.com/amitbet/vnc2video)
library which made this wrapper possible.

## Use

    docker run -it widerin/vnc-recorder --help


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
       --ffmpeg value     Which ffmpeg executable to use (default: "ffmpeg") [$VR_FFMPEG_BIN]
       --host value       VNC host (default: "localhost") [$VR_VNC_HOST]
       --port value       VNC port (default: 5900) [$VR_VNC_PORT]
       --password value   Password to connect to the VNC host (default: "secret") [$VR_VNC_PASSWORD]
       --framerate value  Framerate to record (default: 30) [$VR_FRAMERATE]
       --crf value        Constant Rate Factor (CRF) to record with (default: 35) [$VR_CRF]
       --outfile value    Output file to record to. (default: "output.mp4") [$VR_OUTFILE]
       --help, -h         show help
       --version, -v      print the version

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
