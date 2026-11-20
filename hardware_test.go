//go:build hardware

package main

import (
	"os"
	"testing"
)

// These tests touch the real devices and run only by hand, with a person at the machine, because opening the camera
// or the microphone shows the system's permission prompt and lights the camera:
// KOLO_MEET_HARDWARE=1 go test -tags hardware -run Hardware.

func hardwareOrSkip(t *testing.T) {
	t.Helper()
	if os.Getenv("KOLO_MEET_HARDWARE") != "1" {
		t.Skip("set KOLO_MEET_HARDWARE=1 to use the real devices")
	}
}

// TestHardwareListsDevices lists the devices without opening one.
func TestHardwareListsDevices(t *testing.T) {
	hardwareOrSkip(t)
	var hw realHardware
	cams, err := hw.Cameras()
	t.Logf("cameras %+v %v", cams, err)
	mics, err := hw.Microphones()
	t.Logf("microphones %+v %v", mics, err)
	spks, err := hw.Speakers()
	t.Logf("speakers %+v %v", spks, err)
}

// TestHardwareOpensAndCloses opens the default camera and microphone, reads a frame of each and closes them.
func TestHardwareOpensAndCloses(t *testing.T) {
	hardwareOrSkip(t)
	p := startPeer(t, "Hardware", Config{First: freeUDP(t)}, nil, func(s *MeetService) { s.hw = realHardware{}; s.vid = defaultVideo })
	mic, err := p.SetMicrophone(true)
	t.Logf("microphone %+v %v", mic, err)
	cam, err := p.SetCamera(true)
	t.Logf("camera %+v %v", cam, err)
	p.SetMicrophone(false)
	p.SetCamera(false)
}
