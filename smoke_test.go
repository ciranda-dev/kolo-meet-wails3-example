//go:build smoke

package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestSmokeAgainstRunningApp makes a call between two services through the entry node of a running app, such as
// `./bin/kolo-meet -first 127.0.0.1:4000`: go test -tags smoke -run Smoke with KOLO_SMOKE_ENTRY=127.0.0.1:4000. The
// devices are fakes; nothing opens a real camera or microphone.
func TestSmokeAgainstRunningApp(t *testing.T) {
	entry := os.Getenv("KOLO_SMOKE_ENTRY")
	if entry == "" {
		t.Skip("set KOLO_SMOKE_ENTRY to the entry node of a running app")
	}
	a := startPeer(t, "Ana", Config{Bootstrap: entry}, newFakeHW(60, 440), nil)
	a.rec.waitNetwork(t, netReady)
	b := startPeer(t, "Bruno", Config{Bootstrap: entry}, newFakeHW(130, 660), nil)
	b.rec.waitNetwork(t, netReady)
	ps := []*peer{a, b}
	invite := startCall(t, ps)
	a.NotchCopyInvite()
	if a.clip() != invite {
		t.Fatal("the notch's Copy did not copy the invite")
	}
	idA, idB := idOf(t, a), idOf(t, b)
	waitUntil(t, "media both ways", func() bool {
		return a.rec.frameCount(idB) > 3 && b.rec.frameCount(idA) > 3 && tone(a.hw.heard(), 660) > 0.05 && tone(b.hw.heard(), 440) > 0.05
	})
	a.rec.mu.Lock()
	defer a.rec.mu.Unlock()
	for _, e := range a.rec.notchRaw {
		if strings.Contains(e, invite) {
			t.Fatalf("the notch got the invite: %s", e)
		}
	}
	t.Logf("call of %d people through %s: pictures and sound both ways, %d notch events, none with the invite", len(ps), entry, len(a.rec.notchRaw))
	time.Sleep(10 * time.Millisecond)
}
