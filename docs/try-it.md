# Try it

Build the example first; it writes `bin/kolo-meet`:

```sh
GOWORK=off wails3 build
```

The network is chosen with flags only: Settings shows the network, the entry nodes and the connection relays the window uses, and a command to copy that starts it the same way. Every command below gives each window its own data folder with `-data`. The example creates the folder with mode 0700 and refuses one that is open to other users, so let the example create it: a folder made with `mkdir` is usually open to others. Folders under `/tmp` go away with the next restart, and the example then gets a new identity.

Words used here:

- **Peer**: one running copy of the example.
- **Entry node**: a [kolo-dht](https://github.com/ciranda-dev/kolo-dht) server every peer contacts first to find the others, such as a [kolo-node](https://github.com/ciranda-dev/kolo-node).
- **Public network**: the entry nodes Kolo ships, run by Ciranda. Free, with no uptime guarantee while Kolo is a beta. The example uses them when you give no network flag.
- **Your own network**: entry nodes you run and give with `-bootstrap`. When it is set, only those are used, never the public ones.
- **This computer only**: `-local`. The windows of one computer find each other through an entry node on 127.0.0.1.
- **Connection relay**: a [kolo-relay](https://github.com/ciranda-dev/kolo-relay) that carries a connection when two computers cannot connect directly, such as one run with `kolo-node relay`. It copies encrypted bytes it cannot read.

## Anywhere, on the public network

Start the example with no network flag, on each computer, each with its own data folder.

```sh
./bin/kolo-meet -data /tmp/kolo-meet-demo/me
```

The two computers can be on different networks. Kolo connects them directly when it can. If the public network cannot be reached when the example starts (no internet, or nothing answered), the window says so and names the three options: check the connection and Try again, `-bootstrap` for a network of your own, or `-local` for this computer only. Nothing switches network by itself. A connection relay is needed only when two computers behind strict NATs cannot connect directly, and you run your own (see below).

In the first window choose a name, click Start a call and copy the invite. In the other choose a name, click Join with an invite, paste it, then click Join call in the lobby. The `+` beside Calls in the sidebar opens the same two choices.

## Same computer

Start every window with `-local` (or `KOLO_MEET_LOCAL=1`). The first window runs an entry node on `127.0.0.1:7451`, and the others join it. Nothing leaves the computer, and windows on other computers cannot reach it. If another program holds that port, the window says so and offers Try again.

Terminal 1: the host.

```sh
./bin/kolo-meet -data /tmp/kolo-meet-demo/host -local
```

Terminal 2: the guest.

```sh
./bin/kolo-meet -data /tmp/kolo-meet-demo/guest -local
```

`-local` cannot be used with `-bootstrap` or `-first`. To test your own network on loopback with addresses you choose, the host's example can run the first entry node with `-first` (loopback only), and every other window uses `-bootstrap` with that address:

```sh
./bin/kolo-meet -data /tmp/kolo-meet-demo/host -first 127.0.0.1:4000
./bin/kolo-meet -data /tmp/kolo-meet-demo/guest -bootstrap 127.0.0.1:4000
```

A second window started with `-first` on the same address says so and offers the `-bootstrap` command to copy.

In the host's window choose a name, click Start a call and copy the invite. In the guest's window choose a name, click Join with an invite, paste it, then click Join call in the lobby. Closing a window quits that copy of the example.

## Two computers on your own network

Computer A runs an entry node and a connection relay; both computers run the example and point at A with `-bootstrap`. Two placeholders:

- `<ip-of-computer-a>`: A's IPv4 address. On one network, its LAN address: `ipconfig getifaddr en0` on macOS, `hostname -I` on Linux, `ipconfig` on Windows. On different networks, the public address that reaches A (see the next section).
- `<relay-key>`: the relay's key, 64 hex characters, which `kolo-node relay` logs as `key` in its `identity` line when it starts.

Computer A: install kolo-node.

```sh
go install github.com/ciranda-dev/kolo-node/cmd/kolo-node@v0.1.1
```

Computer A, terminal 1: the entry node. `-identity` is the path of its key file: kolo-node creates the file with mode 0600 on the first run and keeps the same key on the next runs. `-listen 0.0.0.0:7447` makes it listen on UDP port 7447 on every interface.

```sh
kolo-node run -listen 0.0.0.0:7447 -identity /tmp/kolo-meet-demo/node/node.key -first
```

Computer A, terminal 2: the connection relay, on UDP port 7448 on every interface, with its own key file. Without `-allow-file` it is open to any key and logs `open relay`; with an allow list, add the key of each copy (each copy prints it in its terminal, in the `kolo-id` line). A call's own relay has a new key for each call, which no list can name, so for a call between two networks use a relay without an allow list.

```sh
kolo-node relay -listen 0.0.0.0:7448 -identity /tmp/kolo-meet-demo/relay/relay.key
```

Computer A: allow UDP ports 7447 and 7448 in the firewall.

Computer B: check that both answer. The first prints `ok <ip-of-computer-a>:7447`, the second `ok relay ...`.

```sh
kolo-node check <ip-of-computer-a>:7447
```

```sh
kolo-node check -relay <ip-of-computer-a>:7448@<relay-key>
```

Computers A and B: the example.

```sh
./bin/kolo-meet -data /tmp/kolo-meet-demo/me \
  -bootstrap <ip-of-computer-a>:7447 \
  -relay <ip-of-computer-a>:7448@<relay-key>
```

With an entry node that is not on loopback, the example listens on `0.0.0.0:0` so the other computer can connect. Instead of the flags:

- Entry nodes: `KOLO_BOOTSTRAP=<ip-of-computer-a>:7447`.
- Connection relays: `-relay-file FILE` (one `host:port@key` per line), or `KOLO_MEET_RELAYS` (read only when neither flag is given).

A flag wins over its variable. More in [configuration](configuration.md#connection-relays).

## Your own network between different networks

The commands are the same as on one network, with one change: both copies must reach A's entry node and relay, so `<ip-of-computer-a>` is a public address. A needs a public IPv4 address (for example a small server) or a UDP port forward to it from a router with a public address. On a server, keep the key files in a lasting folder instead of `/tmp`; the [kolo-node deploy guide](https://github.com/ciranda-dev/kolo-node/blob/main/docs/deploy.md) has a systemd unit and the firewall rules. What the example can use, the two kinds of relay and an allow list for your relay are in [your own servers](own-servers.md).

Kolo connects the two computers directly when it can, with the entry node's help. When their networks let no direct connection through, they meet through a connection relay that both list. During a call, Settings says how many relays are ready and why one holds no place, and a tile marked "via relay" is a person reached through one. With relays listed, a join through a relay takes about 12 seconds more than a direct one for each of its two connections (the host's app, then its call relay), and it may take up to 72 seconds for each before it gives up.

The host's relay keys travel in the answer to a guest's join. A guest whose list shares no relay with the host's sees a warning in the call: two people who cannot connect directly need a relay both list, or they stay "waiting for keys" for each other. Two guests are not compared with each other. A join that found the host but could not connect ends with a sentence saying so, with Try again; to add a relay, start the example again with `-relay`.

## Camera and microphone

- macOS asks the first time you turn the camera or the microphone on. For the microphone, Kolo Meet reads the permission first and asks only if you were never asked; the lobby has Allow the microphone, and Cancel while the system waits. To change it later: System Settings, Privacy & Security, Camera or Microphone.
- Windows asks nothing in the example: Settings, Privacy & security, Camera or Microphone must let desktop apps use them.
- Linux asks nothing: your user must be allowed to use the devices (often the video group for cameras), and a sandbox must allow them.
- A refused device is said on its button; you can join and only listen. A microphone that sends only silence for 4 seconds shows a hint, since a refusal on macOS can give silent audio.

## It does not connect

1. Every copy uses the same network: all with no network flag (public), all with the same `-bootstrap`, or all with `-local` on one computer (Settings, Network shows which). With your own network, `kolo-node check <ip-of-computer-a>:7447` prints `ok` from each computer. On the public network, check your internet connection and that UDP is not blocked; the public servers have no uptime guarantee while Kolo is a beta.
2. UDP is allowed: ports 7447 and 7448 in A's firewall (and the provider's), and Kolo Meet in each computer's firewall.
3. On different networks, both are started with the same connection relay (`-relay`), `kolo-node check -relay` prints `ok relay`, and during the call Settings counts it as ready.

## The window shows an error

If the window shows an error pane, press Copy and look in the terminal for the line starting with `frontend error`. A call keeps running behind the pane: use its Leave button (End call for the host), or Turn them off for a camera or microphone left on in the lobby, then quit Kolo Meet and open it again.
