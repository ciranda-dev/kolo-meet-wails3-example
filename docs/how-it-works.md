# How a call works

## What the window shows

One person starts a call and gets an invite. The others paste it, see a lobby with their camera, a microphone meter and device pickers, and join. The call is a grid of participants with names, a speaking indicator, mute, camera on and off, Leave and a timer, for 2 to 8 people.

The window has a sidebar on the left and the page beside it. The sidebar has the Kolo mark and a button that folds it to a narrow rail (the window remembers the choice), the heading Calls with a `+` that opens Start a call and Join with an invite, the call that runs (a window holds one call, so the list has at most one row; it is empty with no call), and a card with your name, History, Settings and one quiet line about the network: public, your own or this computer only, then how it stands, never an address. A bar on top of the page names the screen and the state, and in a call it holds People and, for the host, Invite. The microphone, the camera, the time and Leave are in the bar under the grid. Joining with an invite, the invite of your call, who is in it, what its protection line means, the history and the help of a device open in a dialog; Settings is a page beside the sidebar. At 800 px the sidebar shrinks.

There is no text chat, no file sharing and no screen sharing: the example only opens the camera and the microphone and makes the call.

## The steps

1. **The host starts the call.** The person who starts the call is the host. The host's example runs the relay of the call, a [kolo-moq](https://github.com/ciranda-dev/kolo-moq) relay with its own identity on its own network node, and makes an invite with [kolo-invite](https://github.com/ciranda-dev/kolo-invite), served by a second network node under another key made for the call. The first node accepts only people the host let in; the second accepts anyone, with the small stream budget of a stranger (see [security.md](security.md#memory)).
2. **The invite.** An invite is a code that starts with `koloinv0`. It carries a fingerprint of the key the invite is served under and a secret, never an address.
3. **A guest joins.** The guest finds the host's invite node on a topic derived from the secret. The host's example lets it in while the call has room (8 people by default, at most 20), then answers with the room, the relay and the host, sealed for that guest alone. The guest then finds the relay on the call's own topic.
4. **Everyone sends and watches.** Everyone publishes a broadcast with a catalog, an Opus audio track, a VP8 video track and a small profile track with the display name, and watches every other participant through the relay.
5. **Keys go direct.** Between each pair of participants there is also a connection of their own, direct or through a connection relay both list, used for keys only and never through the call's relay (see [security.md](security.md)).
6. **Leaving and removing.** A guest who leaves drops from everyone's list, and each person left changes their key. The host can also remove one guest (People, then Remove, then Remove again to confirm): the host's example closes that guest's connection to the relay, so the same happens, and the removed guest's window says "The host removed you from this call." and closes its camera and microphone. The host cannot be removed.
7. **The call ends** when the host leaves: the call lasts as long as the host stays, and when the host leaves it ends for everyone and says why.

## Good to know

- The host can make more invites. An invite that turned someone away because the call was full keeps saying no to that person, so use a new one.
- Someone who left or was removed cannot come back with an invite that existed then: they see "this invite no longer lets you in". A new invite from the host (Invite, then Make a new invite) lets them in again. A guest who leaves, or is removed, needs a new invite; a guest whose connection is only lost does not (see the next points). The host keeps this list in memory for the call only, at most 4 people for each place (32 for a call of 8), the oldest dropped first; see [security.md](security.md#the-invite) for what it does not stop.
- On a slow or lossy connection, such as a mobile one, the call keeps its voice first. Each person's copy measures what its connection to the call's relay carries (how long the oldest unsent frame waits, and how much was sent) and lowers the camera's bitrate to about 75% of that, down to 5% of its full rate; it raises it again by 5% of the full rate a second once the connection has been clear for 3 seconds. If the queue is older than a second the camera's frames are not encoded until it empties, and when a group of frames was lost the next frame is a key frame. The microphone is never slowed down, and its frames go before the camera's. Every 5 seconds the example writes to its log, for each person, the audio frames and pictures it decoded a second and what the connection dropped, and a line starting `path:` each time it lowers the bitrate, which goes to the terminal the example was started from.
- The camera has a quality ladder with three rungs, 640x360 (about 0.5 Mbit/s), 1280x720 (about 1.9 Mbit/s) and the ceiling `-video-size`, 1920x1080 by default (3.8 Mbit/s), and below them the audio-only floor above. The ceiling is the most that is ever sent, and never more than the camera gives. A call starts at the lowest rung. The ladder steps down at once, at the next look at the connection (every 250 ms), when the queue is older than a quarter of a second and the throughput measured while it was full cannot carry the rung with 25% to spare (and 64 kbit/s for the audio), when about a tenth of the groups are lost (a fifth goes to the lowest rung), when the queue is older than a second, or when the encoder needs more than 60% of a frame interval for the rung; the processor is measured on the delta frames the encoder really encodes, as nanoseconds a pixel, and a rung is chosen only if its pixels, at that cost, fit in 60% of the 33 ms between two frames. It steps up one rung after the connection has been clear for 4 seconds, with the bitrate back at the rung's full rate and, if the throughput measured while the queue was full in the last 20 seconds is known, only to a rung it covers with 25% to spare. A step up that is followed by a step down within 10 seconds makes the next wait 4 times as long, up to 120 seconds; a step up that lasts resets the wait. A change of rung makes a new encoder, which starts with a key frame, and adds the video track again with the new size. With several guests the best of them decides, as for the bitrate. The 1080p and 720p rungs are cut down when more people join, as the next point of [configuration](configuration.md#audio-and-video) says.
- A person whose microphone delivers its frames slower than real time, because the example's thread for it was starved or the device overran, would have its audio timestamps (which count samples) fall behind the clock of its video without end, and every receiver, which measures the arrival of the audio and the video against one reference, would take that for audio that is always late and drop all of it while the video plays. The timeline of the audio therefore jumps forward to the clock when it is more than 100 ms behind; the receiver sees the frames lost as a gap and makes up the first 3 of them. The microphone's encoder also has a lock of its own, so it does not wait for a camera frame being encoded.
- A path can carry small packets and drop large ones, or the other end of it can change its port while yours does not follow: the connection stays up and nothing arrives, and the transport only gives up after 30 seconds. Each copy therefore also watches the media. When a person who sends audio or video, as its catalog says, delivers none for 6 seconds, or the example's own frames wait in the queue and almost no byte leaves for 6 seconds, the path counts as dead and the example makes a new session to the relay, as above. If that does not bring the media back, the next wait is doubled, up to 48 seconds. Someone who turned the microphone and the camera off sends nothing, and that is not a dead path. When only one person of several is silent, it is that person's path: their own copy resets it.
- A viewer whose picture stopped because a group was lost, or who just joined, asks the sender for a key frame at once (kolo-moq's request for a new group), instead of waiting up to 2 seconds for the next one. A sender makes at most one such key frame a second.
- In a call with several people the host sizes its camera for the guests that receive it: each guest's session has a controller of its own and the camera follows the best of them, so one guest on a thin path, or one whose connection died, does not lower the picture of the others. A guest that carries nothing for 5 seconds is not counted until it does again, and then gets a key frame. The guest on a thin path sees what its path carries, with its audio first.
- Missing audio is covered. When an audio frame does not arrive in time, up to 3 frames in a row are made up by the Opus decoder (the last of them from the forward error correction in the next packet when it has it) and a longer gap stays silent. When the path loses groups, the microphone turns on Opus's forward error correction with an expected loss of 5% to 20%, at the same constant bitrate and packet size. If the speaker's queue for a person gets longer than 100 ms, for example after a burst of late audio, the example drops 20 ms in every 60 ms by crossfading two frames into one, so it is back under 100 ms within 2 seconds without a click.
- A microphone or a camera that is off has no track and is closed, so the system's "in use" light goes out.
- Kolo connects each pair directly with the help of [kolo-dht](https://github.com/ciranda-dev/kolo-dht). When a direct connection fails, the two copies can meet through a connection relay, a [kolo-relay](https://github.com/ciranda-dev/kolo-relay) that both are started with (`-relay`) (empty by default; see [configuration](configuration.md#connection-relays)). This holds for the key link between two participants and for a guest's connection to the call's relay. Two participants who cannot connect directly and share no connection relay show each other as "waiting for keys" and get nothing of each other. A guest whose list shares no relay with the host's sees a warning in the call; two guests are not compared with each other.
- A join that found the host but could not connect to it ends with a sentence that says so: the host's example could not be reached directly or through a relay, or you and the host have no relay in common.
- Display names are cleaned: control and format characters dropped, blanks collapsed, at most 48 characters. A name from another participant is never trusted to be well formed.
- A person is in one call at a time; a second start gets a "busy" error.

## What drawing the tiles costs

The window's pictures are made by the example: each picture of each tile is scaled and compressed to a JPEG on the processor, and sent to the window as an event. `-display-fps` sets how many a second ([display rate](configuration.md#display-rate)). `auto` draws 30 up to 4 other people with video and 15 above, and falls back to 15 when the drawing takes over half a core; `30` always draws 30 and has no fallback.

`TestMeasureDisplayTiles` measures `-display-fps 30` with 720p cameras (`KOLO_MEET_MEASURE=1 go test -run TestMeasureDisplayTiles`, without `-race`, 5 seconds for each row): one tile for each other person, with the picture size the policy gives for that many people, with and without the example's own tile. The computer was an Apple M5 that other work was using (load average 5 to 9), so each CPU figure is the lowest and the highest of 3 runs. Smooth is a gradient, which gives small JPEGs; textured is the test camera's noise-like picture, which gives the largest ones and is worse than a face.

| Content | People | Picture | Tiles drawn | Pictures a second for each tile | Event | CPU, other people's tiles | CPU, with your own tile |
|---|---|---|---|---|---|---|---|
| smooth | 2 | 1280x720 | 1 | 30 | 30.5 KB | 19% to 29% | 38% to 56% |
| smooth | 4 | 640x360 | 3 | 30 | 10.1 KB | 39% to 73% | 58% to 85% |
| smooth | 6 | 640x360 | 5 | 30 | 10.1 KB | 80% to 105% | 105% to 139% |
| textured | 2 | 1280x720 | 1 | 15 | 118 KB | 16% to 26% | 33% to 43% |
| textured | 4 | 640x360 | 3 | 12.4 | 72 KB | 24% to 35% | 29% to 43% |
| textured | 6 | 640x360 | 5 | 8.1 | 75 KB | 24% to 40% | 29% to 42% |

- The CPU is of the whole process (percent of one core), and is what a tile costs at the rate it really drew. Smooth pictures reached 30 a second, so those figures are the cost of 30. The textured ones were held under 30 by the byte budget (3,579,138 bytes a second for all tiles, shared by them: 1766 to 3611 KB/s were sent), so their CPU is less than 30 would cost: the time to draw and compress one picture was 11 to 17 ms at 1280x720 and 6 to 9 ms at 640x360, which is about 30% to 50% of a core for each 1280x720 tile and 18% to 27% for each 640x360 tile at 30 pictures a second.
- A person's face is between the two kinds of picture. The shared budget is the limit the rate can meet with many tiles at 30: with 5 other people it holds 5 tiles of 10 KB pictures at 30 a second (1510 KB/s) but not 5 tiles of 75 KB ones (8 a second each).
- The webview's side of the work, fetching, decoding and drawing the pictures, is not in these figures.

## Reading the quality report

`kolo-meet-quality.jsonl` (in the data directory; see [configuration](configuration.md#the-call-quality-report) for the switches and the size cap) has one JSON object a line. Every line has `kind` (`remote` or `local`), `at` (unix seconds), `t` (seconds since the call began on this computer), `secs` (the length of the interval, about 5) and `call` (the call's random ID). Counts are of the interval, not of the call. Two computers' files can be set side by side by `at`.

A `remote` line is what one other person delivered to this example. `person` is 1 for the first person seen in the call, then 2, and so on; it is the same number in every line of the call.

| Field | Meaning |
|---|---|
| `audio_in` | audio frames decoded (50 a second when all arrive) |
| `audio_missed` | audio frames that did not arrive: gaps in the sequence |
| `audio_concealed` | frames made up for them (Opus loss concealment, and the forward error correction in the next packet), at most 3 in a row |
| `audio_silent` | missed and not made up: the part of a gap that stayed silent |
| `audio_dropped`, `video_dropped` | frames the jitter buffer threw away for arriving too late or over its limit |
| `video_decoded_fps` | video pictures decoded a second |
| `video_shown_fps` | pictures given to the window a second: at most the display rate, and fewer when pictures are over the byte budget |
| `video_w`, `video_h` | the size of the newest decoded picture |
| `freezes` | times the pictures stopped for more than 500 ms while the person sends video (its catalog lists a video track), counted at the window, so a stop of the display counts too. A stop that goes on over several lines is counted once and shows its length so far in each. Nothing counts before the first picture, or while the camera is off |
| `freeze_longest_ms` | the longest of them in the interval |
| `audio_bytes`, `video_bytes` | payload bytes of the frames decoded, without the transport's headers |
| `playout_delay_ms` | the jitter buffer's delay now (100 to 400 ms): audio and video of one person play after it |
| `playout_jitter_ms` | the 95th percentile of how late frames arrive, which sets the delay |
| `decode_ms` | the running mean time to decode a video frame |
| `tile_ms` | the running mean time to draw and compress one picture for the window |
| `display_fps` | the rate pictures are drawn at now: 30 or 15 (see [display rate](configuration.md#display-rate)) |

The `local` line is what this example sent, over all its sessions to the relay (a host has one for each guest).

| Field | Meaning |
|---|---|
| `role`, `people` | `host` or `guest`, and the people in the call, this one included |
| `rung_w`, `rung_h` | the size being sent: the rung of the quality ladder |
| `rung_changes`, `rung_why` | changes of rung in the interval, and the reason of the last one (the same text as the `video:` log line), empty without one |
| `rate_kbps` | the video encoder's target now, in kbit/s: the rung's full rate scaled by what the path carries |
| `sent_kbps` | what was handed to the transport in kbit/s, audio and video, all sessions |
| `encoded_fps`, `skipped_frames` | camera frames encoded a second, and camera frames not encoded because the queue was over a second old |
| `encode_ms` | the running mean time to encode a delta frame at the rung's size |
| `queue_ms` | how long the oldest unsent object waits, the worst session: over 250 ms is a full path |
| `sessions` | sessions to the relay that carry media |
| `groups_reset`, `groups_expired` | groups of frames given up in the interval, because the queue was over its limit or they were too old to be useful: the sign of a path that carries less than is sent |
| `fec_loss_pct` | the loss the microphone's encoder expects, 0 with its forward error correction off |
| `reconnects` | sessions to the relay made again after a loss |
| `watchdog_resets` | times the watchdog made a new session because no media arrived or left for 6 seconds or more |
| `display_mode` | the `-display-fps` setting in use: `auto`, `30` or `15` |
| `key_frames` | key frames written to the camera track in the interval; at most one a second, apart from the first one after a size change, which makes a new encoder |
| `net` | what the transport measured on the connection of the session with the highest round-trip time; left out when no session reports it (a session through a connection relay has no numbers) |

The `net` object has these fields. Only the loss covers every session; the rest is of that one session.

| Field | Meaning |
|---|---|
| `rtt_ms`, `min_rtt_ms`, `rtt_var_ms` | the smoothed round-trip time, the lowest one seen on the path (close to the propagation delay) and the mean deviation of the samples (jitter). A `rtt_ms` far above `min_rtt_ms` is a queue on the path |
| `loss_pct` | packets declared lost over packets sent in the interval, over all the sessions that report: an upper bound, since a packet that arrives late still counts |
| `cwnd_bytes`, `in_flight_bytes` | the bytes the congestion controller lets be in flight, and the bytes sent and not yet acknowledged. `in_flight_bytes` near `cwnd_bytes` with a growing `queue_ms` means the controller limits the call; far below it with a full queue means the application sends less than the path carries |
| `controller` | the congestion controller sending on the connection: `reno`, `cubic` or `bbr` |
| `bw_kbps` | the bottleneck bandwidth BBR measured, in kbit/s; left out for `reno` and `cubic`, which measure none, and before the first sample |
| `path_changes` | how many times the connection moved to a new path, as when a NAT gives the peer a new port |

What to look for in a test on a real network: audio frames a second near 50 and `audio_silent` under 2% of them; `freezes` at 0; `playout_delay_ms` near its floor of 100 ms; `rung_h` and `sent_kbps` following what the link carries; `queue_ms` under 250 and `groups_*` at 0 on a clear path.
