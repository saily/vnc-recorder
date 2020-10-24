package main

import (
	"context"
	"fmt"
	vnc "github.com/amitbet/vnc2video"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
	"net"

	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"
	"time"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	app := &cli.App{
		Name:    path.Base(os.Args[0]),
		Usage:   "Connect to a vnc server and record the screen to a video.",
		Version: "1.0",
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
			&cli.StringFlag{
				Name:    "outfile",
				Value:   "output.mp4",
				Usage:   "Output file to record to.",
				EnvVars: []string{"VR_OUTFILE"},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func recorder(c *cli.Context) error {
	address := fmt.Sprintf("%s:%d", c.String("host"), c.Int("port"))
	nc, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		log.Fatalf("Error connecting to VNC host. %v", err)
		return err
	}
	defer nc.Close()

	log.Infof("Connected to %s", address)

	// Negotiate connection with the server.
	cchServer := make(chan vnc.ServerMessage)
	cchClient := make(chan vnc.ClientMessage)
	errorCh := make(chan error)

	var secHandlers []vnc.SecurityHandler
	if c.String("password") == "" {
		secHandlers = []vnc.SecurityHandler{
			&vnc.ClientAuthNone{},
		}
	} else {
		secHandlers = []vnc.SecurityHandler{
			&vnc.ClientAuthVNC{Password: []byte(c.String("password"))},
		}
	}

	ccfg := &vnc.ClientConfig{
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

	cc, err := vnc.Connect(context.Background(), nc, ccfg)
	defer cc.Close()

	screenImage := cc.Canvas
	if err != nil {
		log.Fatalf("Error negotiating connection to VNC host. %v", err)
		return err
	}

	ffmpeg_path, err := exec.LookPath(c.String("ffmpeg"))
	if err != nil {
		panic(err)
	}
	log.Infof("Using %s for encoding", ffmpeg_path)
	vcodec := &X264ImageCustomEncoder{
		FFMpegBinPath: ffmpeg_path,
		Framerate:     c.Int("framerate"),
	}

	//goland:noinspection GoUnhandledErrorResult
	go vcodec.Run(c.String("outfile"))

	for _, enc := range ccfg.Encodings {
		myRenderer, ok := enc.(vnc.Renderer)

		if ok {
			myRenderer.SetTargetImage(screenImage)
		}
	}

	cc.SetEncodings([]vnc.EncodingType{
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
			log.Debugf("Received client message type:%v msg:%v\n", msg.Type(), msg)
		case msg := <-cchServer:
			if msg.Type() == vnc.FramebufferUpdateMsgType {
				secsPassed := time.Now().Sub(timeStart).Seconds()
				frameBufferReq++
				reqPerSec := float64(frameBufferReq) / secsPassed
				log.Debugf("reqs=%d, seconds=%f, Req Per second= %f", frameBufferReq, secsPassed, reqPerSec)

				reqMsg := vnc.FramebufferUpdateRequest{Inc: 1, X: 0, Y: 0, Width: cc.Width(), Height: cc.Height()}
				reqMsg.Write(cc)
			}
		case signal := <-sigCh:
			if signal != nil {
				log.Info(signal, " received, exit.")
				vcodec.Close()
				// give some time to write the file
				time.Sleep(time.Second * 1)
				os.Exit(0)
			}
		}
	}
	return nil
}
