package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/netip"
	"strconv"

	kolodht "github.com/ciranda-dev/kolo-dht"
	koloid "github.com/ciranda-dev/kolo-id"
	invite "github.com/ciranda-dev/kolo-invite"
	kolomoq "github.com/ciranda-dev/kolo-moq"
	kolosecure "github.com/ciranda-dev/kolo-secure"
	kolostore "github.com/ciranda-dev/kolo-store"
	koloswarm "github.com/ciranda-dev/kolo-swarm"
	kolotransport "github.com/ciranda-dev/kolo-transport"
)

// maxCallInvites bounds the invites a call keeps: each takes a topic of the invite swarm, which holds 16 by default
// (kolo-swarm Config.MaxTopics).
const maxCallInvites = 8

// Incoming streams of a connection, the most the peer may have open at once (kolo-transport MaxIncomingStreams, 1 to
// 1000). Each subscription takes a stream and each group in flight another, so a person in a call needs many: the
// connections of the call take the most, 1000. A stranger needs one, for a request to join, so the host's invite swarm
// keeps kolo-transport's default, 16. kolo-transport keeps about 96 heap bytes per byte of an open stream, 856 MiB for
// one connection at 16 streams and about 47 GiB at 1000 (kolo-transport docs/limits.md), and the limit of a connection
// cannot change after the handshake, so the two kinds of peer are served by two swarms.
const (
	maxStreams    = 1000
	inviteStreams = 16
)

// transportConfig is the setting of the connections of a call: loopback, the guests' swarm and the host's call relay
// swarm, which only people let in reach.
func transportConfig() *kolotransport.Config {
	return &kolotransport.Config{MaxIncomingStreams: maxStreams}
}

// inviteTransport is the setting of the connections a stranger can make: the host's invite swarm.
func inviteTransport() *kolotransport.Config {
	return &kolotransport.Config{MaxIncomingStreams: inviteStreams}
}

// Congestion controllers of -congestion, as kolo-transport names them.
const (
	congestionReno  = "reno"
	congestionCubic = "cubic"
	congestionBBR   = "bbr"
)

// validCongestion reports whether name is a controller kolo-transport takes.
func validCongestion(name string) bool {
	return name == congestionReno || name == congestionCubic || name == congestionBBR
}

// runHost starts a call: the relay (its own identity, on its own swarm node and on a loopback listener for the host's
// own session), the invite (its own identity and swarm node), and the host as the first participant.
func (c *call) runHost() {
	if err := c.startHost(); err != nil {
		c.end(explainStart(err))
	}
}

func explainStart(err error) *MeetError {
	var me *MeetError
	if errors.As(err, &me) {
		return me
	}
	return meetErr("call-unavailable", err, "")
}

func (c *call) startHost() error {
	s := c.svc
	sw := s.swarm()
	if sw == nil {
		return meetErr("network-not-ready", ErrNotReady, "")
	}
	relID, err := koloid.Generate()
	if err != nil {
		return err
	}
	roomID, seed := make([]byte, 16), make([]byte, 32)
	rand.Read(roomID)
	rand.Read(seed)
	c.mu.Lock()
	c.relID = relID.PublicKey()
	c.room = &room{id: hex.EncodeToString(roomID), topic: roomTopic(seed), relay: relID.PublicKey(), host: c.me,
		hostName: s.displayName(), max: s.maxParticipants()}
	c.mu.Unlock()
	set := s.settingsCopy()
	payload, err := encodeGrant(c.room.id, seed, c.room.relay, c.me, c.room.hostName, c.room.max, relayKeys(set.relays))
	if err != nil {
		return err
	}

	if c.relay, err = kolomoq.NewRelay(&kolomoq.RelayConfig{Session: kolomoq.Config{Authorize: c.relayAuthorize}, Resume: reconnectWindow}); err != nil {
		return err
	}
	if c.ln, err = kolosecure.Listen(c.ctx, "127.0.0.1:0", relID, &kolosecure.Config{Transport: transportConfig()}); err != nil {
		return err
	}
	c.spawn(func() {
		for {
			conn, err := c.ln.Accept(c.ctx)
			if err != nil {
				return
			}
			if bytes.Equal(conn.RemotePublicKey(), c.me) {
				c.spawn(func() { c.relay.NewServer(c.ctx, conn) })
				continue
			}
			c.spawn(func() { c.serveConn(conn, conn.RemotePublicKey()) })
		}
	})

	// The invite is served under a key of its own on a swarm any peer may reach; the call's relay only to the people
	// let in, so that a stranger never holds a connection with the stream budget of a person in the call.
	invID, err := koloid.Generate()
	if err != nil {
		return err
	}
	if c.rsw, err = c.openSwarm(set, swarmCallRelay, relID, c.relayAllow); err != nil {
		return err
	}
	if err := c.rsw.Join(c.ctx, c.room.topic, &koloswarm.JoinOptions{Announce: true}); err != nil {
		return meetErr("call-unavailable", err, "")
	}
	c.spawn(c.relayLoop)

	if c.isw, err = c.openSwarm(set, swarmCallInvite, invID, c.inviteAllow); err != nil {
		return err
	}
	db := kolostore.NewMemory()
	c.invDB = db
	if c.inv, err = invite.Open(db, invID, nil); err != nil {
		return err
	}
	c.spawn(func() {
		if err := c.inv.Announce(c.ctx, c.isw); err != nil && c.ctx.Err() == nil && !errors.Is(err, invite.ErrClosed) {
			c.logf("invite announce: %v", err)
		}
	})
	c.spawn(c.inviteSwarmLoop)
	c.grant = payload
	if _, err := c.newInvite(); err != nil {
		return err
	}
	c.spawn(c.inviteLoop)
	if err := sw.Join(c.ctx, c.room.topic, nil); err != nil {
		return meetErr("call-unavailable", err, "")
	}

	conn, err := kolosecure.Dial(c.ctx, c.ln.Addr().String(), s.id, relID.PublicKey(), &kolosecure.Config{Transport: transportConfig()})
	if err != nil {
		return meetErr("relay-unreachable", err, "")
	}
	return c.enter(conn)
}

// relayAuthorize is the relay's rule: the room policy, and, on top of it, a guest announces its namespace only if the
// call has a place for it. RoomPolicy alone lets any peer announce its own namespace and checks Member only for
// reading, so a guest without a place would be let in and closed afterwards, and would see a transport error
// instead of a refusal.
func (c *call) relayAuthorize(peer ed25519.PublicKey, kind kolomoq.RequestKind, ns kolomoq.Namespace, track string) error {
	if kind == kolomoq.KindPublishNamespace && !c.isMember(peer) {
		return errNoPlace
	}
	return kolomoq.RoomPolicy{Room: kolomoq.Namespace{c.room.id}, Member: c.isMember}.Authorize(peer, kind, ns, track)
}

var errNoPlace = errors.New("kolo-meet: the call has no place for this guest")

// openSwarm starts and bootstraps one of the call's two swarms under id. It closes the swarm when it fails.
func (c *call) openSwarm(set settings, which string, id *koloid.Identity, allow func(ed25519.PublicKey, netip.AddrPort) bool) (*koloswarm.Swarm, error) {
	sw, err := koloswarm.New(c.ctx, set.Listen, id, &koloswarm.Config{
		BootstrapNodes: set.nodes,
		LookupInterval: set.LookupInterval,
		RetryMax:       dialRetryMax,
		DHT:            &kolodht.Config{Secure: &kolosecure.Config{Transport: c.svc.swarmTransport(which)}},
		Allow:          allow,
		Relays:         set.relays,
	})
	if err != nil {
		return nil, meetErr("network-open", err, "")
	}
	if err := sw.Bootstrap(c.ctx, nil); err != nil {
		sw.Close()
		return nil, bootstrapFailure(set, err, false)
	}
	return sw, nil
}

// relayAllow admits to the relay's swarm the people the host let in, and the ones it barred, who hold the connection
// to be told so; nobody else, and not the host's own app, which reaches the relay in this process.
func (c *call) relayAllow(key ed25519.PublicKey, _ netip.AddrPort) bool {
	return !bytes.Equal(key, c.me) && (c.approved(key) || c.isBarred(key))
}

// inviteAllow admits any peer to the invite swarm but the host's own app: it asks to join there.
func (c *call) inviteAllow(key ed25519.PublicKey, _ netip.AddrPort) bool {
	return !bytes.Equal(key, c.me)
}

// relayLoop gives the relay every guest that connects to it on the network.
func (c *call) relayLoop() {
	for {
		ev, err := c.rsw.Next(c.ctx)
		if err != nil {
			return
		}
		if ev.Kind == koloswarm.RelayChanged {
			c.svc.relayChanged(swarmCallRelay, ev)
		}
		if ev.Kind == koloswarm.Connected {
			peer := ev.Peer
			c.spawn(func() { c.serveRelay(peer) })
		}
	}
}

// inviteSwarmLoop serves every peer that connects to the invite swarm: a guest asking to join.
func (c *call) inviteSwarmLoop() {
	for {
		ev, err := c.isw.Next(c.ctx)
		if err != nil {
			return
		}
		if ev.Kind == koloswarm.RelayChanged {
			c.svc.relayChanged(swarmCallInvite, ev)
		}
		if ev.Kind != koloswarm.Connected {
			continue
		}
		peer := ev.Peer
		c.spawn(func() {
			if err := c.inv.ServePeer(c.ctx, peer); err != nil && c.ctx.Err() == nil && !errors.Is(err, invite.ErrClosed) {
				c.logf("invite: %v", err)
			}
		})
	}
}

// relayPeer is a guest's session at the relay and the connection under it.
type relayPeer struct {
	sess *kolomoq.Session
	conn koloswarm.Conn
}

// serveRelay runs the relay session of a guest. A barred guest gets the session set up first, so that it holds the
// connection when it is closed with closeShutOut and can say why; the relay's rule refuses it every request.
func (c *call) serveRelay(peer swarmPeer) { c.serveConn(peer.Conn(), peer.PublicKey()) }

// serveConn is serveRelay for a connection to the relay and the key of the guest at its other end.
func (c *call) serveConn(conn koloswarm.Conn, key ed25519.PublicKey) {
	sess, err := c.relay.NewServer(c.ctx, conn)
	if err != nil {
		return
	}
	c.mu.Lock()
	_, barred := c.barred[string(key)]
	if !barred {
		if c.relaySessions == nil {
			c.relaySessions = map[string]relayPeer{}
		}
		c.relaySessions[hex.EncodeToString(key)] = relayPeer{sess: sess, conn: conn}
	}
	c.mu.Unlock()
	if barred {
		conn.CloseWithError(closeShutOut, "")
	}
}

// remove takes the guest id out of the call: it is barred, and its connection to the relay is closed, so the relay
// drops its namespace and everyone left rotates their key.
func (c *call) remove(id string) error {
	c.mu.Lock()
	p := c.people[id]
	c.mu.Unlock()
	if p == nil {
		return meetErr("participant-unknown", nil, "")
	}
	c.shut(p.key, c.inviteIDs())
	c.mu.Lock()
	rp, ok := c.relaySessions[id]
	c.mu.Unlock()
	if ok {
		rp.conn.CloseWithError(closeShutOut, "")
	}
	c.logf("removed %s", id[:8])
	return nil
}

// inviteIDs lists the invites of the call that can still be used.
func (c *call) inviteIDs() []invite.ID {
	if c.inv == nil {
		return nil
	}
	infos, err := c.inv.List()
	if err != nil {
		return nil
	}
	var ids []invite.ID
	for _, i := range infos {
		if i.State == invite.Active {
			ids = append(ids, i.ID)
		}
	}
	return ids
}

// inviteLoop answers the requests to join: a guest is let in while there is room, and told so by the invite.
func (c *call) inviteLoop() {
	for {
		ev, err := c.inv.Next(c.ctx)
		if err != nil {
			return
		}
		if ev.Kind != invite.Requested {
			continue
		}
		c.admitRequest(ev)
	}
}

func (c *call) admitRequest(ev invite.Event) {
	if h := c.svc.requestHook; h != nil {
		h(ev)
	}
	if !c.letIn(ev.Joiner, ev.Invite) {
		c.inv.Deny(ev.Request)
		return
	}
	c.admit(ev.Joiner)
	if err := c.inv.Approve(ev.Request, c.grant); err != nil {
		c.forget(ev.Joiner)
		c.logf("invite approve: %v", err)
	}
}

// approved reports whether the host let key in; the relay's own rule, with the places, decides what it may read.
func (c *call) approved(key ed25519.PublicKey) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.admitted[string(key)]
	return ok
}

// newInvite makes an invite for the call and keeps its code as the one the window shows. The earlier invites keep
// working until they expire or the call ends.
func (c *call) newInvite() (string, error) {
	if c.inv == nil {
		return "", ErrNoCall
	}
	infos, err := c.inv.List()
	if err != nil {
		return "", err
	}
	active := 0
	for _, i := range infos {
		if i.State == invite.Active {
			active++
		}
	}
	if active >= maxCallInvites {
		return "", meetErr("too-many-invites", ErrBusy, strconv.Itoa(maxCallInvites))
	}
	code, _, err := c.inv.Create(&invite.CreateOptions{TTL: inviteTTL, MaxUses: maxInviteUses, Manual: true})
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.inviteStr = code
	c.mu.Unlock()
	c.changed()
	return code, nil
}
