# Configuration

Every setting is a flag or an environment variable; a flag wins. A bad setting makes the window show the reason.

| Flag | Environment | Default | Meaning |
|---|---|---|---|
| `-data` | `KOLO_MEET_DATA` | `<user config dir>/kolo-meet` | the data directory (see below) |
| `-bootstrap` | `KOLO_BOOTSTRAP` | none: the public network | entry nodes of your own network, a comma list of `host:port` as for kolo-swarm's `ParseNodes`: IPv4 or a host name, at most 16. When set, only these are used |
| `-local` | `KOLO_MEET_LOCAL` | off | use this computer only: the first window runs an entry node on `127.0.0.1:7451` and the others join it. `1` turns it on from the variable. Not with `-bootstrap` or `-first` |
| `-relay` | `KOLO_MEET_RELAYS` | none | connection relays, `host:port@key` with the relay's key as 64 hex characters, as for kolo-swarm's `ParseRelays`; repeat the flag or give a comma list; at most 16 in all, duplicates dropped. `KOLO_MEET_RELAYS` is read only when neither `-relay` nor `-relay-file` is given |
| `-relay-file` | none | none | a file of connection relays, one `host:port@key` per line, `#` starts a comment; repeatable, added to `-relay` |
| `-first` | `KOLO_MEET_FIRST` | none | run the first entry node of your own test network on this loopback IP:PORT, such as `127.0.0.1:4000`; the other windows use `-bootstrap` with it. For local testing only. Not with `-local` |
| `-listen` | `KOLO_MEET_LISTEN` | `127.0.0.1:0` with `-local` or when every entry node is on loopback, else `0.0.0.0:0` | local IPv4 IP:PORT |
| `-max-participants` | `KOLO_MEET_MAX_PARTICIPANTS` | `8` | people in a call you host, yourself included, from 2 to 20. kolo-media measured protected calls of 4, 8 and 20 people |
| `-find-timeout` | `KOLO_MEET_FIND_TIMEOUT` | `1m`, `195s` with connection relays | how long a join looks for the host and for its relay. With connection relays the default covers kolo-dht's hole punch through up to three of its nodes, 45 s each, and one relay dial of 60 s, all of which can come before the relay is used |
| `-lookup-interval` | `KOLO_MEET_LOOKUP_INTERVAL` | `5s` | time between lookups of a topic |
| `-video-size` | `KOLO_MEET_VIDEO_SIZE` | `1920x1080` | the ceiling of the picture, `WxH`, from `1x1` to `1920x1080` (the largest picture the example takes from another participant), and the size the camera is asked for; the camera gives its nearest mode. It is a ceiling, not the size sent: the example sends 360p, 720p or this size as the connection, the processor and the call allow, and never more than the camera gives (see below) |
| `-video-bitrate` | `KOLO_MEET_VIDEO_BITRATE` | kolo-video's rule: 0.07 bits per pixel per frame at 30 fps, at most 3,800,000 bit/s: 483,840 at 640x360, 1,935,360 at 1280x720, 3,800,000 at 1920x1080 | VP8 bits a second at `-video-size`; each smaller picture sent keeps the same bits per pixel. At most 4,294,967,295,999, libvpx's limit |
| `-display-fps` | `KOLO_MEET_DISPLAY_FPS` | `auto` | the pictures a second each tile of the window draws: `auto`, `30` or `15` (see [display rate](#display-rate)) |
| `-congestion` | `KOLO_MEET_CONGESTION` | `reno` | how the connections that carry a call pace what they send: `reno`, `cubic` or `bbr` (see [congestion control](#congestion-control)) |
| `-quality-report` | `KOLO_MEET_QUALITY` | `on` | `on` writes the call quality report, `kolo-meet-quality.jsonl`, to the data directory; `off` does not (see [the call quality report](#the-call-quality-report)) |
| `-quality-log` | `KOLO_MEET_QUALITY_LOG` | off | also write each line of the report to the log, starting `quality:`; `1` turns it on from the variable |

`GOMEMLIMIT` (not a Kolo setting) sets the soft memory limit of the process; without it the example sets 2 GiB at start (see [security](security.md#memory)).

Kolo is IPv4 only: an IPv6 entry node, connection relay or listen address is refused. A `-congestion` other than `reno`, `cubic` or `bbr`, a `-display-fps` other than `auto`, `30` or `15` and a `-quality-report` other than `on` or `off` are refused too.

## The network

There are three networks, and exactly one is used:

- **Public**, when none of `-bootstrap`, `-first` and `-local` is given. The example starts from the public entry nodes Kolo ships, run by Ciranda: free, with no uptime guarantee while Kolo is a beta. Two people on different networks find each other with no setup.
- **Your own**, with `-bootstrap` (or `KOLO_BOOTSTRAP`, or `-first` for a loopback test). Only your entry nodes are used; the public ones are never added.
- **This computer only**, with `-local` (or `KOLO_MEET_LOCAL=1`). One entry node on `127.0.0.1:7451`, run by the first window of the computer; the others join it. If another program holds the port, the window says so.

`-local` with `-bootstrap` or `-first` is refused with a message that names both. If the public network cannot be reached when the example starts, the window shows an error that names the three options, and Try again starts it again; the example never falls back to another network by itself. The public servers are entry nodes only: there is no public connection relay and no public pin, and the example uses a relay only if you give one with `-relay`.

## Chosen in the window

The display name, the camera, the microphone, the speaker and the notch setting are chosen in Settings and saved in `settings.json`; the language and the theme are kept by the window. The network and the connection relays are set with the flags or the variables above, never in the window: Settings shows the network in use (public, your own or this computer only), the entry nodes of your own network (never the public ones), where each came from, and a command to copy that starts the example the same way.

## Audio and video

- Video is captured at `-video-size` (1920x1080 by default, the ceiling) and 30 frames a second, kolo-video's default rate, and sent as VP8 with a key frame every 2 seconds. The largest picture sent is the camera's, scaled down until three bounds hold: no larger than `-video-size`, no larger than kolo-video's engine encodes in real time (`vpx.Capabilities`: 1920x1080 with the built-in library on macOS and Windows, 640x480 with the portable engine, the Linux default), and, with the streams of all the others, no more pixels a second than one real-time decoder (1920x1080 at 30 fps) at each receiver. With the default setting that is 1920x1080 for 2 people, 960x540 for 3 to 4, 640x360 above (for 1280x720, 1280x720 for 2 and 3 people and 640x360 for 4 to 10). A camera that gives less than the ceiling is the ceiling: nothing is ever sent larger than the camera's picture.
- The picture is sent at one of a few sizes, the rungs of a quality ladder: 640x360, 1280x720 and the largest picture above, each at kolo-video's 0.07 bits per pixel per frame (483,840, 1,935,360 and 3,800,000 bit/s; at 1080p the rule would give 4.35 Mbit/s, and a larger picture needs fewer bits per pixel, so it stops at 3.8 Mbit/s; `-video-bitrate` sets the ceiling's rate and the rungs keep its bits per pixel). A rung that is within a quarter of the next one is left out, so a 800p camera has 360p and 800p. Below the lowest rung is the audio-only floor: when the queue is older than a second the camera is not encoded until it empties. A call starts at the lowest rung and climbs; see [how it works](how-it-works.md#good-to-know). A change of rung adds the video track again with the new size, at a key frame; the receivers follow without anything else changing.
- The window shows each picture at most at the size it is on screen, within the same setting: 1920x1080 with one or two pictures, 960x540 with 3 to 8, 480x270 above. How many pictures a second each tile draws is `-display-fps` (below): 30 up to 4 other people with video and 15 with more by default. All pictures together stay within 3,579,138 bytes a second at 30 and 1,789,569 at 15, so a tile whose pictures are large gets fewer of them a second. The sizes above do not depend on the rate or on `-display-fps`.
- Audio is mono Opus at 48 kHz in 20 ms frames at a constant 32 kbit/s, so packet sizes do not reveal speech.
- A bad connection is handled with fixed timings, not options: a call tries for 60 seconds to get its session to the relay back before it ends, with waits from 0.5 s to 4 s; a person who sends media and delivers none for 6 seconds makes the path count as dead (doubling after each reset that did not help, up to 48 seconds); a guest that carries nothing for 5 seconds is not counted in the host's camera rate; the ladder steps up after 4 seconds of a clear path, with the wait multiplied by 4 after each step that failed, up to 120 seconds, and steps down at once; at most one key frame a second is made because a viewer asked; audio is made up for at most 3 frames in a row. See [how it works](how-it-works.md#good-to-know).

## Display rate

`-display-fps` (or `KOLO_MEET_DISPLAY_FPS`) sets the pictures a second each tile of the window draws, yours included. A flag wins over its variable; any other value is refused.

- `auto`, the default, draws 30 pictures a second while at most 4 other people show video (a call of up to 5 people) and 15 with more. A guard measures the time it takes to draw and compress each picture (a running mean for each tile): when drawing every tile at 30 would take more than 50% of one processor core, the tiles draw at 15 until that work is under 40%.
- `30` draws 30 for every tile whatever the number of people, and the guard is off.
- `15` draws 15 for every tile.

The numbers are constants in `tiles.go`: `tileRateHigh`, `tileRateLow`, `tileFewRemote`, `tileCPUMax`, `tileCPUResume`. The call quality report names the setting in use in its `local` line, `display_mode`.

What 30 costs. Drawing a tile means scaling the decoded picture and compressing it to a JPEG on the processor, for each picture. On an Apple M5 a 1280x720 tile takes 6 to 8 ms a picture when the picture is smooth and 11 to 17 ms when it is noisy, which is about 20% and 30% or more of one core for each tile at 30 pictures a second; the 640x360 pictures that tiles get in a group take 4 to 9 ms, so a tile there costs less. The cost adds up with the tiles: 5 smooth tiles of 640x360 at 30 took 80% to 105% of a core, and with your own tile 105% to 139% ([the measurement](how-it-works.md#what-drawing-the-tiles-costs)). So `30` in a call of many people can take more than one core on a weak laptop, which leaves less for the decoders and the encoder; `auto` falls back to 15 there and `30` does not. Use `30` where the computer has the room, and `15` or `auto` where it does not.

What `30` does not change:

- The size of each picture. It still shrinks with the number of tiles as above (1280x720 for 2 people, 640x360 for 4 and 6 with a 1280x720 camera), because the pixels a second all tiles may draw stay bounded.
- The byte budget. All pictures together stay within 3,579,138 bytes a second, shared by the tiles, so with many tiles each gets a smaller share. A tile whose pictures are larger than its share sends fewer than 30 a second (and a picture larger than a whole second of its share is not sent). Smooth pictures, such as a face on a plain background, are small enough for 30 with 5 other people; the noisy picture of the tests is not (see the measurement). The rate the window gets is `video_shown_fps` in the quality report.
- What Wails does with the events. A picture over 8 KiB is parked in a store of at most 64 MiB until the webview fetches it, for 30 seconds at most. At 3,579,138 bytes a second a webview that fetches nothing fills the store in about 18 seconds (at 15, 1,789,569 bytes a second never fills it). When it is full, Wails sends the next events inline instead of dropping them, and on macOS that keeps their memory in the example for as long as it goes on. A webview that is drawing fetches them as they come, and the store stays small.

## Congestion control

Every connection paces what it sends with a congestion controller. `-congestion` picks it for the connections that carry a call: the ones each person's copy makes to the call's relay and the ones the relay's own swarm makes to the guests. Each end paces its own sending, so the setting works on the side that sends; set it on every computer you compare. The default is `reno`, kolo-transport's own, and nothing changes with it.

`bbr` estimates the link's rate and round trip and does not slow down for isolated losses, so on a lossy path it carries much more than `reno` (kolo-transport measured 18.9 Mbit/s against 1.8 on 20 Mbit/s with 1% random loss each way and 60 ms of delay each way). A call is not like a file: it sends no more than its camera and its microphone make, so it is limited by the application and not by the path, and `bbr` there keeps a standing queue of 25 to 150 ms (kolo-transport's guide, "Congestion control") that adds to the delay of every frame and takes bandwidth from other flows on a shallow queue. That is why `reno` is the default and `bbr` is there to compare on a lossy path, such as a mobile one: run the same call once with each and read the [quality report](how-it-works.md#reading-the-quality-report) (the rung, `sent_kbps`, `queue_ms` and `playout_delay_ms`). `cubic` grows its window faster than `reno` after a loss and is no better under random loss. The host's own session to its relay is on the same computer and is not paced by this setting.

## The call quality report

While a call runs the example writes numbers about it, so a test on a real network gives data and not a feeling: one JSON line every 5 seconds for each other person in the call and one for what this example sends. The lines go to `kolo-meet-quality.jsonl` in the data directory, mode 0600 in a directory of mode 0700, unless `-quality-report off` (or `KOLO_MEET_QUALITY=off`). With `-quality-log` (or `KOLO_MEET_QUALITY_LOG=1`) the same lines also go to the log, each starting `quality:`, which is the terminal the example was started from.

- The report holds counts, sizes and times only, and when each call was and how many people were in it. It never holds a name, a key, an invite, an address or any media: a person is a number, 1 for the first person seen in the call, and a call is its random ID, which the log shows too. It is on by default while Kolo is a beta; the log says at start where the file is and how to turn it off, and [security](security.md#the-call-quality-report) says what it holds.
- The file takes at most 2 MiB. When the next line would pass that, the file is renamed to `kolo-meet-quality.jsonl.1`, replacing the one there, and a new file starts, so the report takes at most 4 MiB and keeps the last 2 to 4 MiB of lines (a line is about 400 bytes, so a call with one other person keeps its last 3 hours at the least). Calls are appended one after the other.
- The window shows nothing of it, and what is counted does not change what the call does.
- The numbers of the transport (round trip, loss, congestion window) are not in it: kolo-moq's `SessionStats` of v0.1.4 and kolo-transport v0.1.7 do not give them. `queue_ms`, `groups_reset` and `groups_expired` show a congested path instead.

Every field is in [how it works](how-it-works.md#reading-the-quality-report).

## Where data lives

The data directory holds `identity.key` (mode 0600), `settings.json`, `history.json`, `kolo-meet-quality.jsonl` (mode 0600, see above) and the video codec's library. It is created with mode 0700. An existing data directory open to other users is refused.

## Your own network

To keep a private network, every copy lists the same entry nodes: kolo-dht servers all of them can reach, such as [kolo-node](https://github.com/ciranda-dev/kolo-node) servers of your own. Give the list with `-bootstrap` or `KOLO_BOOTSTRAP`:

```sh
KOLO_BOOTSTRAP=node1.example.org:7447,node2.example.org:7448 ./bin/kolo-meet
```

With entry nodes that are not on loopback the example listens on `0.0.0.0:0`, so the other computers can connect. UDP must not be blocked between them.

## Connection relays

Two computers that cannot connect directly, for example both behind strict NATs, can still reach each other through a connection relay: a [kolo-relay](https://github.com/ciranda-dev/kolo-relay) server, such as one run with `kolo-node relay` (see [kolo-node](https://github.com/ciranda-dev/kolo-node)). The list is empty by default: there is no public connection relay, and the example never uses one you did not give. Give them with the flags or the variable:

```sh
./bin/kolo-meet -relay relay1.example.org:7000@<key in hex> -relay-file relays.txt
```

```sh
KOLO_MEET_RELAYS=relay1.example.org:7000@<key in hex> ./bin/kolo-meet
```

What the list does:

- Both of the example's network nodes use it: the one every participant runs, and the one that serves the host's call relay. A connection is tried directly first; a relay is used only when the direct connection fails.
- Two people meet through a relay only when both list it. The host's relay keys travel in the answer to a guest's join, so a guest whose list shares no relay with the host's sees a warning in the call.
- The example holds a place at its relays only while it is in a call. Settings and the call say how many are ready, and why one holds no place; the terminal logs each reason once. When a place ends with the relay's lifetime the example takes a new one, and that is not shown.
- When its network starts, at start and at each Reconnect, the example proves each relay's key with one handshake and asks the relay for nothing, at most 30 s for each relay and stopped with the network. A relay listed with a wrong key then shows in Settings, and while a guest joins, at once and not when the join gives up.
- Settings says where the list came from: `-relay`, `-relay-file`, both, or `KOLO_MEET_RELAYS`.
- In a call, each tile says when that person is reached through a relay, and the protection panel says how a guest reaches the call's relay.

What it does not do:

- It does not replace the call's relay: the host still runs it and every person's media goes through it.
- A guest is warned only about the host's list. Two guests who each share a relay with the host but none with each other are not warned; if they cannot connect directly they stay "waiting for keys" for each other.
- A relay operator can read nothing of what it carries (the connections are encrypted end to end between the two copies), but learns your key, your address, when you are in a call, and who connects to whom, when and how much ([security](security.md#connection-relays)).

A relay with an allow list lets only the keys on it hold a place, so it must list your example's key: the example prints it in the terminal, in the `kolo-id` line, once the network is up. The relay of a call has a key made for that call, which no allow list can name, so it cannot hold a place at a relay with an allow list: a guest who cannot reach the host's example directly needs a relay without one. Many people behind one address can fill a relay's places: size it with [kolo-relay's limits](https://github.com/ciranda-dev/kolo-relay/blob/main/docs/limits.md).
