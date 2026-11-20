# Security

## End-to-end encryption

Calls use kolo-media's protected broadcasts with group keys: each participant seals every object of its broadcast, the catalog and the profile included, with a key of its own, and gives that key to each other participant over the connection between the two (direct, or through a connection relay both list), never through the call's relay. Display names travel on a small profile track whose object is padded to 512 bytes before it is sealed, so the relay does not see how long a name is.

The call is end-to-end encrypted only while every participant's keys have arrived, and only then does the window say so. The window says exactly this, from `Call.Security`:

> N in call, end-to-end encrypted, through relay run by NAME

(or "run by you" for the host). While K people still wait for keys, the line drops the "end-to-end" words:

> N in call, K waiting for keys, through relay run by NAME

When the line is opened, the window shows:

> The relay cannot see audio, video or names, not even how long a name is. Anyone in the call could, with the relay's help, make audio, video or names that seem to come from someone else. The relay sees who is in the call, who watches whom, when each person sends and how much, when microphones and cameras are off, and the room ID; it can drop or delay media. Anyone in the call can read everyone's media. There is no protection for the current key period if a device is compromised, and no recovery after a compromise.

followed, for a guest, by "NAME runs the relay, so NAME could make audio, video or names that seem to come from someone else." (for the host: "You run the relay, so you could ..."), while someone waits for keys, by "Nothing from a person who is waiting for keys is played.", and, when connection relays are in use, by "Each connection relay in your settings sees your key, your address and when you are in a call, and, for the connections it carries, who connects to whom, when and how much. It cannot read them: they are encrypted end to end between the two copies." Forging is possible because every participant holds every sender's key and objects are not signed per sender.

What this does not give: forward secrecy within a key period, recovery after a compromise, protection of who is in the call and when they speak from the relay, or protection from forging: anyone in the call, with the relay's help, and the person who runs the relay could make media seem to come from someone else. A participant whose key has not arrived is shown as "waiting for keys" and nothing of them plays: there is no unprotected playback, and no unprotected mode. A change of who is in the call changes the keys: someone who left or was removed reads nothing a participant seals after that participant's copy saw them go, and the others see "NAME no longer receives your media". Each copy changes its own key when it sees the change, so for a moment after a removal a participant whose copy has not seen it yet still seals under the old key. What the person received before stays with them, and a modified copy could keep it.

Go's AES-GCM is constant time only on amd64 and arm64 CPUs with AES instructions, ppc64x and s390x; on other CPUs the protection is the weakest link.

## Camera and microphone

The camera and the microphone are opened only when you turn them on, and closed when you turn them off, when the call ends and when the example quits. Before it opens the microphone, the example reads the system's permission status (kolo-audio's `AccessStatus`), and when the person has not been asked yet it asks with `RequestAccess`, which shows the system prompt on macOS. A refusal is a typed error that says so, and the example works with what is left. As a second signal, when the microphone is on but has sent only silence (all-zero audio) for 4 seconds, the window shows a hint to check the system's privacy settings and that the microphone is not muted. The tests never open a device: they use made-up cameras, microphones and speakers.

## Keys and identity

Keys live in memory only and are overwritten when replaced or closed, as far as Go allows (kolo-media's limits apply). When the call ends, the keys of the other participants are erased: they are overwritten, as far as Go allows, and an object sealed under them no longer opens. The identity is `identity.key` in the data directory, mode 0600, never printed.

## The invite

The invite is a bearer secret: whoever has it can ask to join until the call ends, the invite expires (24 hours) or the call is full. It is never written to the history, the logs, the tray or the notch page. Send it over a channel you trust.

The invite stays on the clipboard after Copy only until the call ends: the example then empties the clipboard if it still holds that invite, and leaves it alone if you copied something else.

What the invite binds today: a code with a fingerprint of the key the invite is served under (a key made for the call, not the relay's) and a secret; up to 100 uses and 24 hours; the host's example lets each asker in while the call has a place. A guest proves it holds the invite with a request sealed to a key derived from the secret, carrying an auth value from it, and signed with its own kolo-id key (kolo-invite), so the host knows which key asks, not who the person is. An invite answers a key it let in before with the same answer, without asking the host again.

Leaving and removal: when a guest leaves, loses the connection or is removed, the host's example takes away that key's place at the relay and notes the invites that exist at that moment. The relay then refuses that key, so the guest cannot come back with those invites, nor by connecting to the relay with what it learned in the call; it sees "this invite no longer lets you in". An invite the host makes later lets the key in again: that is how the host re-admits someone. The list lives in the host's memory for the call only and holds at most 4 keys for each place of the call (32 for 8), the oldest dropped first: a key dropped from it has no place at the relay any more, but an older invite it never used would let it in again.

What this does not stop: the list is of keys, not people. Someone who still has a valid invite and makes a new identity (a new kolo-id key, which any modified copy can do) asks as a stranger and is let in while there is a place. To keep a person out for good, do not send them a new invite and end the call when the old invites must stop working: they work until they expire, are used up or the call ends.

## The call quality report

The report is on by default, so that a test on a real network leaves data, and the example logs one line at each start that says where it goes and how to stop it. Each call adds lines to `kolo-meet-quality.jsonl` in the data directory (mode 0600 in a directory of mode 0700; at most 2 MiB, then the file becomes `kolo-meet-quality.jsonl.1`, replacing the one there). A line holds:

- when it was written (`at`, unix seconds), the seconds since the call began (`t`), the length of the interval, a random call ID, the role (`host` or `guest`) and how many people were in the call (`people`);
- counts, sizes and times of one other person (a number counted from 1 in the order they were first reported, never a name or a key): audio frames decoded, missed, made up and silent, pictures decoded and shown a second, freezes, bytes received, the jitter buffer's delay and jitter, decode and draw times;
- the same for what this example sent: the picture size and bit rate it chose, why it changed them, encode times, groups given up, sessions made again, key frames written;
- what the transport measured on the connections to the relay: round-trip times, packet loss, the congestion window, the bytes in flight, the name of the congestion controller and how many times the path changed. No address or key is among them.

It holds no name, key, invite, address, file name or media. It is still a durable record of when you were in calls and how many people were in them. `-quality-report off` (or `KOLO_MEET_QUALITY=off`) writes nothing; delete the file and its `.1` copy to remove what is there. `-quality-log` also prints the same lines to the log, and nothing else. [Configuration](configuration.md#the-call-quality-report) lists every field.

## The log

The example writes a log to the terminal it was started from. It holds no name, invite, message or media. It does hold your own example's public key (the `kolo-id` line, for a connection relay's operator), the address of a connection relay you listed, the first 8 characters of the keys of other people in a call (when media arrives from them, and when the host removes one), and the call ID. Look through it before pasting it in an issue.

## Memory

A peer on a connection makes the example hold memory: kolo-transport keeps about 96 heap bytes for each byte of an open stream, so one connection that opens 16 streams (kolo-transport's default) can hold up to about 856 MiB, and one that opens 1000 up to about 47 GiB. The limit of a connection is set when it is made and cannot change afterwards. The example therefore serves the two kinds of peer on two network nodes:

| Node | Who reaches it | Incoming streams of a connection | Worst case of one connection |
|---|---|---|---|
| The host's invite node | anyone who has the invite, or finds its topic on the network | 16 (a request to join needs one) | about 856 MiB |
| The host's relay node | only the keys the host let in, and the ones it barred (so they can be told); any other key is refused after the handshake | 1000 (each subscription takes a stream and each group another) | about 47 GiB |
| A guest's node, the host's own session | people in the call; the host's own copy on 127.0.0.1 | 1000 | about 47 GiB |

kolo-transport admits at most 4 connections from one address and the node at most 32 from others, so a stranger with several addresses can still make the host hold several GiB while a call runs. Someone who has been let in can do more: send the invite only to people you trust, and remove anyone who misbehaves.

The example sets a soft memory limit of 2 GiB at start (`debug.SetMemoryLimit`), unless `GOMEMLIMIT` is set, which then decides; the log says which at start. A soft limit makes the garbage collector work harder as the heap nears it. It does not free memory a peer keeps live and does not stop an allocation, so the numbers above remain the worst case; a system that kills a process out of memory still can.

## Links between participants

The links between two participants refuse every request except for the key track made for the asker, so no media is served over them.

## Entry nodes

The example starts from entry nodes: the public ones by default (run by Ciranda, with no uptime guarantee while Kolo is a beta), only yours with `-bootstrap`, or one on this computer with `-local`. They help the example find the others and carry no media. Like any server your example sends packets to, they see your address. The call's media, names and keys stay end-to-end between the participants, whichever network is used.

## Connection relays

A connection relay (a [kolo-relay](https://github.com/ciranda-dev/kolo-relay) given with `-relay`) carries a connection only when two copies cannot connect directly. What goes through it stays encrypted end to end between the two copies (kolo-secure), so its operator cannot read keys, media or names. The operator does learn your example's key, your address, when you are in a call (the example holds a place there only during a call), and, for the connections it carries, who connects to whom, when and how much. It can drop or delay what it carries. The list is empty by default; the example prints its key in the terminal (the `kolo-id` line) so the operator of a relay with an allow list can let it hold a place there. To check each relay's key, the example makes one handshake with it whenever its network starts (at start and at each Reconnect), with a throwaway key, and asks it for nothing: the relay sees that key and your address. A test stands where a relay operator stands, holding the relay's key and passing two guests' key link on, and checks that the bytes it reads hold their keys but never the call's room or the hex keys that name its tracks.

## Reporting a problem

Report a security issue as [SECURITY.md](../SECURITY.md) says.
