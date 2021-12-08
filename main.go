package main

import (
	"context"
	"fmt"
	vnc "github.com/amitbet/vnc2video"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"net"

	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"
	"time"
	"strconv"

	"errors"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	app := &cli.App{
		Name:    path.Base(os.Args[0]),
		Usage:   "Connect to a vnc server and record the screen to a video.",
		Version: "0.4.2",
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "Daniel Widerin",
				Email: "daniel@widerin.net",
			},
		},
		Action: recorder,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "ffmpeg",
				Value:   "ffmpeg",
				Usage:   "Which ffmpeg executable to use",
				EnvVars: []string{"VR_FFMPEG_BIN"},
			},
			&cli.StringFlag{
				Name:    "host",
				Value:   "localhost",
				Usage:   "VNC host",
				EnvVars: []string{"VR_VNC_HOST"},
			},
			&cli.IntFlag{
				Name:    "port",
				Value:   5900,
				Usage:   "VNC port",
				EnvVars: []string{"VR_VNC_PORT"},
			},
			&cli.StringFlag{
				Name:    "password",
				Value:   "secret",
				Usage:   "Password to connect to the VNC host",
				EnvVars: []string{"VR_VNC_PASSWORD"},
			},
			&cli.IntFlag{
				Name:    "framerate",
				Value:   30,
				Usage:   "Framerate to record",
				EnvVars: []string{"VR_FRAMERATE"},
			},
			&cli.IntFlag{
				Name:    "crf",
				Value:   35,
				Usage:   "Constant Rate Factor (CRF) to record with",
				EnvVars: []string{"VR_CRF"},
			},
			&cli.StringFlag{
				Name:    "outfile",
				Value:   "output",
				Usage:   "Output file to record to.",
				EnvVars: []string{"VR_OUTFILE"},
			},
			&cli.IntFlag{
				Name:    "splitfile",
				Value:   0,
				Usage:   "Mins to split file.",
				EnvVars: []string{"VR_SPLIT_OUTFILE"},
			},
			&cli.StringFlag{
				Name:    "s3_endpoint",
				Value:   "",
				Usage:   "S3 endpoint.",
				EnvVars: []string{"VR_S3_ENDPOINT"},
			},
			&cli.StringFlag{
				Name:    "s3_accessKeyID",
				Value:   "",
				Usage:   "S3 access key id.",
				EnvVars: []string{"VR_S3_ACCESSKEY"},
			},
			&cli.StringFlag{
				Name:    "s3_secretAccessKey",
				Value:   "",
				Usage:   "S3 secret access key.",
				EnvVars: []string{"VR_S3_SECRETACCESSKEY"},
			},
			&cli.StringFlag{
				Name:    "s3_bucketName",
				Value:   "",
				Usage:   "S3 bucket name.",
				EnvVars: []string{"VR_S3_BUCKETNAME"},
			},
			&cli.StringFlag{
				Name:    "s3_region",
				Value:   "us-east-1",
				Usage:   "S3 region.",
				EnvVars: []string{"VR_S3_REGION"},
			},
			&cli.BoolFlag{
				Name:    "s3_ssl",
				Value:   false,
				Usage:   "S3 SSL.",
				EnvVars: []string{"VR_S3_SSL"},
			},
			&cli.BoolFlag{
				Name:    "debug",
				Value:   false,
				Usage:   "Debug.",
				EnvVars: []string{"VR_DEBUG"},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		logrus.WithError(err).Fatal("recording failed.")
	}
}

//func vcodecRun(c *cli.Context, vcodec *X264ImageCustomEncoder, ccflags *vnc.ClientConfig, screenImage *vnc.VncCanvas, vncConnection *vnc.ClientConn, errorCh chan error, cchClient chan vnc.ClientMessage, cchServer chan vnc.ServerMessage, outfile string) {
func vcodecRun(vcodec *X264ImageCustomEncoder, c *cli.Context, outfileName string) error {
	address := fmt.Sprintf("%s:%d", c.String("host"), c.Int("port"))
	dialer, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		logrus.WithError(err).Error("connection to VNC host failed.")
		return err
	}
	defer dialer.Close()

	logrus.WithField("address", address).Info("connection established.")

	// Negotiate connection with the server.
	cchServer := make(chan vnc.ServerMessage)
	cchClient := make(chan vnc.ClientMessage)
	errorCh := make(chan error)

	var secHandlers []vnc.SecurityHandler

	var password string

	if c.String("password") != "secret" {
		password = c.String("password")
	}

	if password == "" {
		secHandlers = []vnc.SecurityHandler{
			&vnc.ClientAuthNone{},
		}
	} else {
		secHandlers = []vnc.SecurityHandler{
			&vnc.ClientAuthVNC{Password: []byte(password)},
		}
	}

	ccflags := &vnc.ClientConfig{
		SecurityHandlers: secHandlers,
		DrawCursor:       true,
		PixelFormat:      vnc.PixelFormat32bit,
		ClientMessageCh:  cchClient,
		ServerMessageCh:  cchServer,
		Messages:         vnc.DefaultServerMessages,
		Encodings: []vnc.Encoding{
			&vnc.RawEncoding{},
			&vnc.TightEncoding{},
			&vnc.HextileEncoding{},
			&vnc.ZRLEEncoding{},
			&vnc.CopyRectEncoding{},
			&vnc.CursorPseudoEncoding{},
			&vnc.CursorPosPseudoEncoding{},
			&vnc.ZLibEncoding{},
			&vnc.RREEncoding{},
		},
		ErrorCh: errorCh,
	}

	vncConnection, err := vnc.Connect(context.Background(), dialer, ccflags)
	defer vncConnection.Close()
	if err != nil {
		logrus.WithError(err).Error("connection negotiation to VNC host failed.")
		return err
	}
	screenImage := vncConnection.Canvas

	//goland:noinspection GoUnhandledErrorResult
	go vcodec.Run(outfileName + ".mp4")

	for _, enc := range ccflags.Encodings {
		myRenderer, ok := enc.(vnc.Renderer)

		if ok {
			myRenderer.SetTargetImage(screenImage)
		}
	}

	vncConnection.SetEncodings([]vnc.EncodingType{
		vnc.EncCursorPseudo,
		vnc.EncPointerPosPseudo,
		vnc.EncCopyRect,
		vnc.EncTight,
		vnc.EncZRLE,
		vnc.EncHextile,
		vnc.EncZlib,
		vnc.EncRRE,
	})

	go func() {
		for {
			timeStart := time.Now()

			vcodec.Encode(screenImage.Image)

			timeTarget := timeStart.Add((1000 / time.Duration(vcodec.Framerate)) * time.Millisecond)
			timeLeft := timeTarget.Sub(time.Now())
			if timeLeft > 0 {
				time.Sleep(timeLeft)
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGKILL,
	)

	frameBufferReq := 0
	timeStart := time.Now()

	for {
		select {
		case err := <-errorCh:
			panic(err)
		case msg := <-cchClient:
			logrus.WithFields(logrus.Fields{
				"messageType": msg.Type(),
				"message":     msg,
			}).Debug("client message received.")

		case msg := <-cchServer:
			if msg.Type() == vnc.FramebufferUpdateMsgType {
				secsPassed := time.Now().Sub(timeStart).Seconds()
				frameBufferReq++
				reqPerSec := float64(frameBufferReq) / secsPassed
				logrus.WithFields(logrus.Fields{
					"reqs":           frameBufferReq,
					"seconds":        secsPassed,
					"Req Per second": reqPerSec,
				}).Debug("framebuffer update")

				reqMsg := vnc.FramebufferUpdateRequest{Inc: 1, X: 0, Y: 0, Width: vncConnection.Width(), Height: vncConnection.Height()}
				reqMsg.Write(vncConnection)
			}
		case signal := <-sigCh:
			if signal != nil {
				logrus.WithField("signal", signal).Info("signal received.")
				vcodec.Close()
				// give some time to write the file
				time.Sleep(time.Second * 1)
				os.Exit(0)
			}
		}
	}
}

func recorder(c *cli.Context) error {
	if c.Bool("debug") {
		logrus.SetReportCaller(true)
	}

	var minioClient *minio.Client

	var outfileName string
	outfile := c.String("outfile")

	if c.Int("splitfile") > 0 {
		t := time.Now()
		outfileName = outfile + "-" + strconv.Itoa(t.Year()) + "-" + strconv.Itoa(int(t.Month())) + "-" + strconv.Itoa(t.Day()) + "-" + strconv.Itoa(t.Hour()) + "-" + strconv.Itoa(t.Minute())
	} else {
		outfileName = outfile
	}

	ffmpegPath, err := exec.LookPath(c.String("ffmpeg"))
	if err != nil {
		logrus.WithError(err).Error("ffmpeg binary not found.")
		return err
	}
	logrus.WithField("ffmpeg", ffmpegPath).Info("ffmpeg binary for recording found")

	vcodec := &X264ImageCustomEncoder{
		FFMpegBinPath:      ffmpegPath,
		Framerate:          c.Int("framerate"),
		ConstantRateFactor: c.Int("crf"),
	}

	if c.String("s3_endpoint") != "" {
		if c.Int("splitfile") == 0 {
			return errors.New("If you want to upload videos to S3, you need to split files.")
		}
		minioClient, err = minio.New(c.String("s3_endpoint"), &minio.Options{
			Creds:  credentials.NewStaticV4(c.String("s3_accessKeyID"), c.String("s3_secretAccessKey"), ""),
			Secure: c.Bool("s3_ssl"),
		})
		if err != nil {
			return err
		}
	}
	if c.Int("splitfile") > 0 {
		ticker := time.NewTicker(time.Duration(c.Int("splitfile")) * time.Minute)

		go vcodecRun(vcodec, c, outfileName)

		for {
			select {
			case _ = <-ticker.C:
				vcodec.Close()
				go func(outfileName string) {
					time.Sleep(10 * time.Second)
					if c.String("s3_endpoint") != "" {
						found, err := minioClient.BucketExists(context.Background(), c.String("s3_bucketName"))
						if err != nil {
							logrus.Error("minioClient.BucketExists", err)
							return
						}
						if ! found {
							err = minioClient.MakeBucket(context.Background(), c.String("s3_bucketName"), minio.MakeBucketOptions{Region: c.String("s3_region")})
							if err != nil {
								logrus.Error("minioClient.MakeBucket", err)
								return
							}
						}
						file, err := os.Open(outfileName + ".mp4")
						if err != nil {
							logrus.Error("os.Open", err)
							return
						}

						fileStat, err := file.Stat()
						if err != nil {
							logrus.Error("fileStat", err)
							file.Close()
							return
						}

						uploadInfo, err := minioClient.PutObject(context.Background(), c.String("s3_bucketName"), outfileName + ".mp4", file, fileStat.Size(), minio.PutObjectOptions{ContentType:"application/octet-stream"})
						if err != nil {
							logrus.Error("minioClient.PutObject", err)
							file.Close()
							return
						} else {
							file.Close()
							os.Remove(outfileName + ".mp4")
						}
						logrus.Debug("Successfully uploaded bytes: ", uploadInfo)
					}
				}(outfileName)
				t := time.Now()
				outfileName = outfile + "-" + strconv.Itoa(t.Year()) + "-" + strconv.Itoa(int(t.Month())) + "-" + strconv.Itoa(t.Day()) + "-" + strconv.Itoa(t.Hour()) + "-" + strconv.Itoa(t.Minute())
				vcodec = &X264ImageCustomEncoder{
					FFMpegBinPath:      ffmpegPath,
					Framerate:          c.Int("framerate"),
					ConstantRateFactor: c.Int("crf"),
				}
				go vcodecRun(vcodec, c, outfileName)
			}
		}

	} else {
		vcodecRun(vcodec, c, outfileName)
	}

	return nil
}
