# Build and test

Only Wails needs cgo, on macOS and Linux, for the native window. The audio, video and network libraries are pure Go and need no codec, no sound library and no compiler. The repository has no `go.work` and no `replace`: every Kolo library comes from a published version in `go.mod` (among them kolo-media v0.1.4, kolo-moq v0.1.5, kolo-audio v0.1.6 and kolo-swarm v0.1.7), so a build with `GOWORK=off` works. If you work across several Kolo repositories at once, a `go.work` in their parent folder lets you build against your local copies.

## Checks

```sh
go vet ./...
go test ./...          # about eight minutes; 127.0.0.1 only, made-up devices
go test -race -timeout 40m ./...
go test -fuzz FuzzParseGrant -fuzztime 30s
cd frontend && npm test && npm run lint && npm run build
```

What the tests cover:

- The end-to-end tests run two to four services in one process, without the window: the first is the first entry node of a private loopback network and the others bootstrap from it.
- The three networks are tested by their settings, which send nothing (nothing given is the public network, `-bootstrap` is used alone, `-local` is one loopback entry node, `-local` with `-bootstrap` or `-first` is an error), and by running: two `-local` windows that meet and call, a port held by another program, and a network of your own whose entry node does not answer. A test peer that would start on the public network fails the test, so no test sends a packet to a public address.
- They check that everyone sees and hears everyone else (pictures of a known brightness, tones of a known pitch), the names, mute and camera off, leaving, the host leaving, a full call, one call at a time, a shutdown during a call, a participant whose key never arrives (nothing of them plays), denied permissions, a device lost during a call, that the invite stays in the main window, and that no goroutine and no device is left.
- The impaired-link tests (`impair_test.go`, `resilience_test.go`, `scenario_test.go`, `internal/netsim`) run a call over a forwarder on 127.0.0.1 that has a bandwidth cap, delay, jitter, loss, an MTU cap, a cut, a NAT that forgets a mapping and a changed port. They check that a call comes back, with both people, sound and pictures and none of them left, from an outage of 10 s and of 40 s, from a path that drops large datagrams and keeps small ones, and from a port that changes on the listener side (`go test -run TestCallRecoversFromFaults`); that a guest on a dead path does not lower the host's camera for another guest; and, with `KOLO_MEET_SLOW=1`, the reports of a late joiner, a lost group, a guest on a thin path and audio at 3% loss. `KOLO_MEET_FAULT=<seconds> go test -run TestFaultReport -v` prints the rates and the time to recover for every fault, and `KOLO_MEET_BEFORE=watchdog,reconnect,keyrequest,fair,conceal,adapt` turns the matching behaviors off to measure a call without them. The quality ladder has tests on a model of the path (`ladder_test.go`: the controllers are the real ones, the link and the encoder's output are modelled, on a virtual clock) and on the same forwarder (`ladder_net_test.go`: the host's camera over a clear path, a 1 Mbit/s cap and 20% loss, and with `KOLO_MEET_SLOW=1` a cap that comes and goes), which print the seconds spent at each rung and the changes with their reasons; a fake encode time (`fakeEncodeTime`) tests the processor guard.
- The connection relay tests start real kolo-relays on 127.0.0.1 and refuse the direct connections between some copies through two test hooks (`acceptHook` on the listener, `dialHook` on the dial; both nil in the example), so those copies can meet only through a relay: a call through a shared relay, through the second relay when the first is down, two copies with no relay in common, a guest that reaches the call's relay only through a relay, a relay listed with a wrong key, reported at start, a relay with the right key that then connects to someone else, which the check does not blame on the key, a Reconnect and a stop that end a check stalled on a relay that never answers, an operator's view of a relayed key link (the guests' keys, never the room or the track names), and a join that fails through the relays too, which says so.
- The window's tests (`frontend/test`, `node --test`) check the copy in three languages against the error table of `errors.go` and the protection sentences of `security.go`, that a stale event of an ended call never shows again, and the connection relay list. One test serves the window with the mock of `frontend/dev` in a headless Google Chrome (skipped when Chrome is not installed, loopback only): the sidebar collapses and remembers it, the `+` menu and the dialog of Join with an invite take the keyboard and give the focus back, the actions of a call (People, Invite, microphone, camera, Leave) stay reachable, the network line shows no address, and the tiles stay inside the grid at 800, 1440 and 2000 px. `node dev/shots.mjs <dir>` takes a screenshot of every screen in both themes and runs the same kind of checks on them. `npm run lint` checks the copy keys, the privacy rules and the contrast of every color pair.
- `smoke_test.go` (build tag `smoke`) runs a call against an entry node of a running example.
- `hardware_test.go` (build tag `hardware`) opens the real devices, only by hand with `KOLO_MEET_HARDWARE=1`.

## Builds

- `wails3 build` builds `bin/kolo-meet` for this computer.
- `wails3 task windows:build ARCH=amd64` cross-builds `bin/kolo-meet.exe` for Windows on x86-64.
- A Linux build needs cgo with GTK 4.10 or later and WebKitGTK 6.0, so build it on Linux, for example Ubuntu 24.04 with `libgtk-4-dev` and `libwebkitgtk-6.0-dev`.

## Version

The version in the window (Settings), in the status the example reports and in every package comes from the git tag: the Taskfile builds it in with `-ldflags "-X main.version=..."` (`KOLO_VERSION` if set, else `git describe --tags`), and a build without it, such as `wails3 dev`, says `dev`. The packaging files in `build/` hold the placeholder `0.0.0`; `scripts/stamp-version.sh VERSION [DIR]` writes the version into `build/config.yml`, both `Info.plist` files, the deb, the Windows version info, manifests and MSIX files, and the NSIS defaults of a copy of the example, and `scripts/release-local.sh` runs it on its copy of HEAD.

## Release files

`scripts/release-local.sh [output dir]` builds the release files of the committed HEAD on a Mac, unsigned and not notarized:

- macOS: a universal `.zip` and `.dmg`;
- Windows amd64 and arm64: `.exe`, `.zip` and installer;
- Linux amd64 and arm64: `.tar.gz` and `.deb`, built in Docker;
- `SHA256SUMS`.

It needs Go, Node.js, the `wails3` CLI and Docker. An unsigned app makes the system warn before it runs.
