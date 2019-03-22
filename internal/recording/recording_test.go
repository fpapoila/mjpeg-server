package recording

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"
)

func outputError(t *testing.T, output []byte, expected []byte) {
	tmpDir, _ := ioutil.TempDir("", "mpjpeg")
	outputPath := filepath.Join(tmpDir, "output.mpjpeg")
	expectedPath := filepath.Join(tmpDir, "expected.mpjpeg")
	ioutil.WriteFile(outputPath, output, 0600)
	ioutil.WriteFile(expectedPath, expected, 0600)
	t.Errorf(
		"Unexpected output: see %s. Expected: see %s",
		outputPath,
		expectedPath,
	)
}

func TestStart(t *testing.T) {
	exitStatusZero = errors.New("restart on exit zero")
	command := "go"
	mpjpegPath := "../../mpjpeg/main.go"
	filePath := "../../gopher.jpg"
	args := []string{"run", mpjpegPath, "-n", filePath}
	imageData, _ := ioutil.ReadFile(filePath)
	var buffer bytes.Buffer
	stop, wait := Start(command, args, &buffer)
	if stop == nil {
		t.Error("Unexpected: stop function is nil")
	}
	if wait == nil {
		t.Error("Unexpected: wait function is nil")
	}
	_, ok := interface{}(stop).(context.CancelFunc)
	if !ok {
		t.Error("Unexpected: stop function is not a context.CancelFunc")
	}
	_, ok = interface{}(wait).(WaitFunc)
	if !ok {
		t.Error("Unexpected: wait function is not a WaitFunc")
	}
	err := wait()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	expectedOutput := bytes.Join(
		[][]byte{
			[]byte("--ffmpeg"),
			[]byte("Content-Type: image/jpeg"),
			[]byte(""),
			imageData,
			[]byte("--ffmpeg--"),
			[]byte(""),
		},
		[]byte("\r\n"),
	)
	output, _ := ioutil.ReadAll(&buffer)
	if !bytes.Equal(output, expectedOutput) {
		outputError(t, output, expectedOutput)
	}
	args = []string{"run", mpjpegPath, "-s", "400ms", filePath}
	buffer.Reset()
	stop, wait = Start(command, args, &buffer)
	go func() {
		time.Sleep(800 * time.Millisecond)
		stop()
	}()
	err = wait()
	if err != context.Canceled {
		t.Errorf("Unexpected error: %s", err)
	}
	expectedOutput = bytes.Join(
		[][]byte{
			[]byte("--ffmpeg"),
			[]byte("Content-Type: image/jpeg"),
			[]byte(""),
			imageData,
			[]byte("--ffmpeg"),
			[]byte("Content-Type: image/jpeg"),
			[]byte(""),
			imageData,
		},
		[]byte("\r\n"),
	)
	output, _ = ioutil.ReadAll(&buffer)
	if !bytes.Equal(output, expectedOutput) {
		outputError(t, output, expectedOutput)
	}
	exitStatusZero = nil
}

func TestStartWithRestart(t *testing.T) {
	exitStatusZero = errors.New("restart on exit zero")
	command := "go"
	mpjpegPath := "../../mpjpeg/main.go"
	filePath := "../../gopher.jpg"
	args := []string{"run", mpjpegPath, "-n", "-s", "1000ms", filePath}
	imageData, _ := ioutil.ReadFile(filePath)
	var buffer bytes.Buffer
	stop, wait := Start(command, args, &buffer)
	go func() {
		time.Sleep(1800 * time.Millisecond)
		stop()
	}()
	err := wait()
	if err == nil {
		t.Error("Unexpected nil error")
	} else if err != context.Canceled {
		t.Errorf("Unexpected error: %s", err)
	}
	expectedOutput := bytes.Join(
		[][]byte{
			[]byte("--ffmpeg"),
			[]byte("Content-Type: image/jpeg"),
			[]byte(""),
			imageData,
			[]byte("--ffmpeg--"),
			[]byte("--ffmpeg"),
			[]byte("Content-Type: image/jpeg"),
			[]byte(""),
			imageData,
		},
		[]byte("\r\n"),
	)
	output, _ := ioutil.ReadAll(&buffer)
	if !bytes.Equal(output, expectedOutput) {
		outputError(t, output, expectedOutput)
	}
	exitStatusZero = nil
}

func TestStartWithInvalidCommand(t *testing.T) {
	command := "./invalid"
	args := []string{""}
	var buffer bytes.Buffer
	_, wait := Start(command, args, &buffer)
	err := wait()
	if err == nil {
		t.Error("Unexpected nil error")
	}
}