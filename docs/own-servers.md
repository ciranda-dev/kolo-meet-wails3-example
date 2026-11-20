# Your own servers

What Kolo Meet needs from the network, and how to run it yourself.

## What the example uses

| Server | Kolo Meet | Point the example at it with |
|---|---|---|
| Entry node (`kolo-node run`) | optional: the public ones are the default | `-bootstrap HOST:PORT[,HOST:PORT...]` or `KOLO_BOOTSTRAP` |
| Connection relay (`kolo-node relay`) | optional | `-relay HOST:PORT@KEY` (repeatable, or a comma list), `-relay-file FILE`, or `KOLO_MEET_RELAYS` when neither flag is given |
| Pin (`kolo-pin run`) | not used | nothing |

The public servers Ciranda runs are entry nodes only, and the example uses them when you give no network flag: free, with no uptime guarantee while Kolo is a beta. There is no public relay and no public pin, so a relay, when you need one, is yours. With `-bootstrap` only your entry nodes are used; `-local` keeps everything on one computer ([try it](try-it.md)).

## Two kinds of relay

- **The call's relay** carries the media. The host's example runs it (a kolo-moq relay with its own identity) for the length of the call: there is nothing to install, and every person's audio and video go through it, sealed so it cannot see them ([security](security.md)).
- **A connection relay** is a server you run with `kolo-node relay`. It carries a connection only when two copies cannot connect directly: a guest's connection to the host's call relay, and the key link between two participants. It copies encrypted bytes it cannot read. It does not replace the call's relay, it stores nothing, and it sees each copy's key, its address, when it is in a call, and who connects to whom, when and how much.

## When you need one

- **Your own entry node**: for a network that only your team uses, or when you do not want to depend on the public servers.
- **A connection relay**: two computers behind strict (symmetric) NATs, or behind a firewall that drops unknown UDP, cannot connect directly. Both must list the same relay: the host and each guest who cannot reach the host, and two participants who cannot reach each other.
- **A pin**: keeps encrypted data for people who are online at different times, such as a chat. A call is live and keeps nothing for later, so Kolo Meet uses none.

## Run an entry node

Run it on a host every computer reaches: a public IPv4 address, such as a VPS, or a router that forwards UDP port 7447 to it.

Install kolo-node.

```sh
go install github.com/ciranda-dev/kolo-node/cmd/kolo-node@v0.1.0
```

The first entry node. It creates its key file with mode 0600 on the first run and keeps it for the next runs.

```sh
kolo-node run -listen 0.0.0.0:7447 -identity "$HOME/kolo-node/node.key" -first
```

A second entry node on another host, so the network stays up when one stops. Replace `<node-1>` with the first node's address.

```sh
kolo-node run -listen 0.0.0.0:7447 -identity "$HOME/kolo-node/node.key" -bootstrap <node-1>:7447
```

Each computer: check that the node answers. It prints `ok <node-1>:7447`.

```sh
kolo-node check <node-1>:7447
```

## Run a connection relay

On a host every computer reaches, with its own key file and its own UDP port. It logs its key as `key` in its `identity` line when it starts: that is the `<relay-key>` below, 64 hex characters.

```sh
kolo-node relay -listen 0.0.0.0:7448 -identity "$HOME/kolo-node/relay.key"
```

Without `-allow-file` the relay is open to any key and logs `open relay`. An allow list says which keys may hold a place there, so that only your team's copies receive connections through it: list each copy's key (the example prints it in the terminal, in the `kolo-id` line), one per line. Any key can still connect through the relay to a copy that holds a place. The relay of a call has a key made for that call, which no allow list can name, so it cannot hold a place at a relay with an allow list: a guest who cannot reach the host's example directly needs a relay without one.

```sh
kolo-node relay -listen 0.0.0.0:7448 -identity "$HOME/kolo-node/relay.key" -allow-file "$HOME/kolo-node/allow.txt"
```

Each computer: check that the relay answers. It prints `ok relay ...`.

```sh
kolo-node check -relay <relay-host>:7448@<relay-key>
```

## Point the example at them

Each computer: start the example with the same entry nodes and relay. With `-bootstrap` the public entry nodes are not used.

```sh
./bin/kolo-meet -bootstrap <node-1>:7447,<node-2>:7447 -relay <relay-host>:7448@<relay-key>
```

`KOLO_BOOTSTRAP` and `KOLO_MEET_RELAYS` take the same lists; a flag wins over its variable. Settings shows the network, its entry nodes and the relays in use, and during a call how many relays are ready. Every relay option is in [configuration](configuration.md#connection-relays). The [kolo-node deploy guide](https://github.com/ciranda-dev/kolo-node/blob/main/docs/deploy.md) has a systemd unit and the firewall rules for a server.

## Ports

- Entry node: UDP 7447 in, in the host's firewall and in the provider's. With ufw: `sudo ufw allow 7447/udp`.
- Connection relay: UDP 7448 in, the same way. With ufw: `sudo ufw allow 7448/udp`.
- The example: UDP on ports the system picks; allow Kolo Meet when macOS or Windows asks.
- No TCP port is used.
