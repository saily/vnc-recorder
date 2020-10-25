package main

/**
* XXX: Ugly workaround for https://github.com/amitbet/vnc2video/issues/10. I've copied the file and build a
* X264ImageCustomEncoder. Once this is merged, we can drop the encoder.go file again.
*/

import (
	"errors"
	"fmt"
	vnc "github.com/amitbet/vnc2video"
	"github.com/amitbet/vnc2video/encoders"
	log "github.com/sirupsen/logrus"
	"image"
	"image/color"
	"io"
	"os"
	"os/exec"
	"strconv"
)

func encodePPMforRGBA(w io.Writer, img *image.RGBA) error {
	maxvalue := 255
	size := img.Bounds()
	// write ppm header
	_, err := fmt.Fprintf(w, "P6\n%d %d\n%d\n", size.Dx(), size.Dy(), maxvalue)
	if err != nil {
		return err
	}

	if convImage == nil {
		convImage = make([]uint8, size.Dy()*size.Dx()*3)
	}

	rowCount := 0
	for i := 0; i < len(img.Pix); i++ {
		if (i % 4) != 3 {
			convImage[rowCount] = img.Pix[i]
			rowCount++
		}
	}

	if _, err := w.Write(convImage); err != nil {
		return err
	}

	return nil
}

func encodePPMGeneric(w io.Writer, img image.Image) error {
	maxvalue := 255
	size := img.Bounds()
	// write ppm header
	_, err := fmt.Fprintf(w, "P6\n%d %d\n%d\n", size.Dx(), size.Dy(), maxvalue)
	if err != nil {
		return err
	}

	// write the bitmap
	colModel := color.RGBAModel
	row := make([]uint8, size.Dx()*3)
	for y := size.Min.Y; y < size.Max.Y; y++ {
		i := 0
		for x := size.Min.X; x < size.Max.X; x++ {
			color := colModel.Convert(img.At(x, y)).(color.RGBA)
			row[i] = color.R
			row[i+1] = color.G
			row[i+2] = color.B
			i += 3
		}
		if _, err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

var convImage []uint8

func encodePPM(w io.Writer, img image.Image) error {
	if img == nil {
		return errors.New("nil image")
	}
	img1, isRGBImage := img.(*vnc.RGBImage)
	img2, isRGBA := img.(*image.RGBA)
	if isRGBImage {
		return encodePPMforRGBImage(w, img1)
	} else if isRGBA {
		return encodePPMforRGBA(w, img2)
	}
	return encodePPMGeneric(w, img)
}
func encodePPMforRGBImage(w io.Writer, img *vnc.RGBImage) error {
	maxvalue := 255
	size := img.Bounds()
	// write ppm header
	_, err := fmt.Fprintf(w, "P6\n%d %d\n%d\n", size.Dx(), size.Dy(), maxvalue)
	if err != nil {
		return err
	}

	if _, err := w.Write(img.Pix); err != nil {
		return err
	}
	return nil
}

type X264ImageCustomEncoder struct {
	encoders.X264ImageEncoder
	FFMpegBinPath string
	cmd           *exec.Cmd
	input         io.WriteCloser
	closed        bool
	Framerate     int
	ConstantRateFactor int
}

func (enc *X264ImageCustomEncoder) Init(videoFileName string) {
	if enc.Framerate == 0 {
		enc.Framerate = 12
	}
	cmd := exec.Command(enc.FFMpegBinPath,
		"-f", "image2pipe",
		"-vcodec", "ppm",
		"-r", strconv.Itoa(enc.Framerate),
		"-an", // no audio
		"-y",
		"-i", "-",
		"-vcodec", "libx264",
		"-preset", "veryfast",
		"-g", "250",
		"-crf", strconv.Itoa(enc.ConstantRateFactor),
		videoFileName,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	encInput, err := cmd.StdinPipe()
	enc.input = encInput
	if err != nil {
		log.Error("can't get ffmpeg input pipe")
	}
	enc.cmd = cmd
}
func (enc *X264ImageCustomEncoder) Run(videoFileName string) error {
	if _, err := os.Stat(enc.FFMpegBinPath); os.IsNotExist(err) {
		return err
	}

	enc.Init(videoFileName)
	log.Infof("launching binary: %v", enc.cmd)
	err := enc.cmd.Run()
	if err != nil {
		log.Errorf("error while launching ffmpeg: %v\n err: %v", enc.cmd.Args, err)
		return err
	}
	return nil
}
func (enc *X264ImageCustomEncoder) Encode(img image.Image) {
	if enc.input == nil || enc.closed {
		return
	}

	err := encodePPM(enc.input, img)
	if err != nil {
		log.Error("error while encoding image:", err)
	}
}

func (enc *X264ImageCustomEncoder) Close() {
	if enc.closed {
		return
	}
	enc.closed = true
	enc.input.Close()
}
