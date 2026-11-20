package main

import (
	"context"
	"errors"
	"io/fs"

	koloaudio "github.com/ciranda-dev/kolo-audio"
	kolovideo "github.com/ciranda-dev/kolo-video"
)

// Device is a camera, a microphone or a speaker. ID is opaque and stays on this computer.
type Device struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

// Devices lists what can be chosen. A kind without devices is an empty list, not an error.
type Devices struct {
	Cameras     []Device `json:"cameras"`
	Microphones []Device `json:"microphones"`
	Speakers    []Device `json:"speakers"`
}

// Names of the three kinds of device in reasons ("camera-denied") and in Lobby.
const (
	devCamera     = "camera"
	devMicrophone = "microphone"
	devSpeaker    = "speaker"
)

type cameraStream interface {
	Read(ctx context.Context) (*kolovideo.Frame, error)
	Close() error
}

type micStream interface {
	Read(ctx context.Context, p []int16) (int, error)
	Close() error
}

type speakerStream interface {
	Write(ctx context.Context, p []int16) (int, error)
	Close() error
}

// hardware is the part of the machine the app uses. The app's comes from kolo-video and kolo-audio, and tests have
// fakes: a test never opens a real device.
type hardware interface {
	Cameras() ([]Device, error)
	Microphones() ([]Device, error)
	Speakers() ([]Device, error)
	// MicrophoneAccess is the system's microphone permission, one of the mic* values below. It never asks and never
	// opens a device. An error means the platform cannot say (nothing to ask).
	MicrophoneAccess() (string, error)
	// RequestMicrophone asks for microphone access when the person has not been asked (macOS: the system prompt) and
	// returns when it is decided or ctx ends. It never opens a device and has no time limit of its own.
	RequestMicrophone(ctx context.Context) error
	// RequestCamera asks for camera access where the system requires it (macOS). It may show a system prompt.
	RequestCamera(ctx context.Context) error
	OpenCamera(id string, w, h int, fps float64) (cameraStream, error)
	OpenMicrophone(id string) (micStream, error)
	OpenSpeaker(id string) (speakerStream, error)
}

// The microphone permissions MicrophoneAccess reports, and the status the window gets.
const (
	micNotDetermined = "not-determined"
	micRestricted    = "restricted"
	micDenied        = "denied"
	micAuthorized    = "authorized"
	micUnknown       = "unknown" // the platform cannot say: opening the microphone tells
)

// Audio runs at 48 kHz mono in 20 ms frames, the Opus defaults of kolo-audio (opus.Config) and what kolo-media's
// group-call tests send (secure_scale_test.go).
const (
	sampleRate    = koloaudio.DefaultSampleRate
	audioChannels = koloaudio.DefaultChannels
	frameSamples  = sampleRate / 50
)

// realHardware is kolo-video and kolo-audio on the machine.
type realHardware struct{}

func list[T any](ds []T, err error, conv func(T) Device) ([]Device, error) {
	out := []Device{}
	for _, d := range ds {
		out = append(out, conv(d))
	}
	if errors.Is(err, kolovideo.ErrNoDevice) || errors.Is(err, kolovideo.ErrUnsupported) ||
		errors.Is(err, koloaudio.ErrNoDevice) || errors.Is(err, koloaudio.ErrUnsupported) {
		return out, nil
	}
	return out, err
}

func (realHardware) Cameras() ([]Device, error) {
	ds, err := kolovideo.Devices()
	return list(ds, err, func(d kolovideo.Device) Device { return Device{ID: d.ID, Name: d.Name, Default: d.Default} })
}

func audioDevices(kind koloaudio.Kind) ([]Device, error) {
	ds, err := koloaudio.Devices(kind)
	return list(ds, err, func(d koloaudio.Device) Device { return Device{ID: d.ID, Name: d.Name, Default: d.Default} })
}

func (realHardware) Microphones() ([]Device, error) { return audioDevices(koloaudio.Capture) }
func (realHardware) Speakers() ([]Device, error)    { return audioDevices(koloaudio.Playback) }

func (realHardware) MicrophoneAccess() (string, error) {
	a, err := koloaudio.AccessStatus()
	if err != nil {
		return "", err
	}
	switch a {
	case koloaudio.AccessNotDetermined:
		return micNotDetermined, nil
	case koloaudio.AccessRestricted:
		return micRestricted, nil
	case koloaudio.AccessDenied:
		return micDenied, nil
	case koloaudio.AccessAuthorized:
		return micAuthorized, nil
	}
	return "", koloaudio.ErrBackend
}

func (realHardware) RequestMicrophone(ctx context.Context) error { return koloaudio.RequestAccess(ctx) }

func (realHardware) RequestCamera(ctx context.Context) error { return kolovideo.RequestAccess(ctx) }

func (realHardware) OpenCamera(id string, w, h int, fps float64) (cameraStream, error) {
	c, err := kolovideo.OpenCapture(kolovideo.CaptureConfig{DeviceID: id, Width: w, Height: h, FrameRate: fps})
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (realHardware) OpenMicrophone(id string) (micStream, error) {
	in, err := koloaudio.OpenInput(koloaudio.Config{DeviceID: id, Format: koloaudio.Format{SampleRate: sampleRate, Channels: audioChannels}})
	if err != nil {
		return nil, err
	}
	return in, nil
}

func (realHardware) OpenSpeaker(id string) (speakerStream, error) {
	out, err := koloaudio.OpenOutput(koloaudio.Config{DeviceID: id, Format: koloaudio.Format{SampleRate: sampleRate, Channels: audioChannels}})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// deviceError turns an error of kolo-video or kolo-audio into the MeetError of a device kind ("camera",
// "microphone" or "speaker").
func deviceError(kind string, err error) *MeetError {
	var me *MeetError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &me):
		return me
	case errors.Is(err, kolovideo.ErrPermission), errors.Is(err, koloaudio.ErrPermission), errors.Is(err, fs.ErrPermission):
		return meetErr(kind+"-denied", err, "")
	case errors.Is(err, kolovideo.ErrNoDevice), errors.Is(err, koloaudio.ErrNoDevice),
		errors.Is(err, kolovideo.ErrUnsupported), errors.Is(err, koloaudio.ErrUnsupported):
		return meetErr(kind+"-none", err, "")
	case errors.Is(err, kolovideo.ErrDeviceLost), errors.Is(err, koloaudio.ErrDeviceLost):
		return meetErr(kind+"-lost", err, "")
	}
	return meetErr(kind+"-failed", err, "")
}
