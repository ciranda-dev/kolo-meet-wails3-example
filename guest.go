package main

import (
	"context"
	"errors"
	"time"

	invite "github.com/ciranda-dev/kolo-invite"
	kolomoq "github.com/ciranda-dev/kolo-moq"
	koloswarm "github.com/ciranda-dev/kolo-swarm"
)

// joinAnswerTimeout bounds the wait for the host's answer: the invite library's own timeout for a request.
const joinAnswerTimeout = 30 * time.Second

// runGuest joins a call: it finds the host on the invite's topic, asks to join, takes the grant, and connects to the
// relay of the call.
func (c *call) runGuest() {
	if err := c.joinHost(); err != nil {
		c.end(explainJoin(err))
	}
}

func explainJoin(err error) *MeetError {
	if me, ok := err.(*MeetError); ok && me.Reason == "rejoin-refused" {
		return me
	}
	if me := relayRefusal(err); me.Reason == "call-full" {
		return me
	}
	if c := explain(err); c.Reason != "internal" {
		return c
	}
	return meetErr("relay-unreachable", err, "")
}

// relayRefusal says why a guest could not use the relay: the relay (the host's app) refused the request or closed the
// session, which it does when the call has no place for the guest; anything else is a relay that cannot be reached.
func relayRefusal(err error) *MeetError {
	var se *kolomoq.SessionError
	var re *kolomoq.RequestError
	if errors.As(err, &se) && se.Remote || errors.As(err, &re) {
		return meetErr("call-full", err, "")
	}
	return meetErr("relay-unreachable", err, "")
}

// relayHook is a test seam: when set, a guest reaches the call's relay through the connection it returns for the
// swarm's one, so a test can put an impaired link on the path of the media.
var relayHook func(c *call, conn kolomoq.Conn) kolomoq.Conn

func (c *call) joinHost() error {
	s := c.svc
	sw := s.swarm()
	if sw == nil {
		return meetErr("network-not-ready", ErrNotReady, "")
	}
	if err := sw.Join(c.ctx, c.code.Topic(), &koloswarm.JoinOptions{Lookup: true}); err != nil {
		return meetErr("lookup-failed", err, "")
	}
	var peer swarmPeer
	select {
	case peer = <-c.joinPeer:
	case <-time.After(s.set.FindTimeout):
		sw.Leave(c.code.Topic())
		c.mu.Lock()
		unreachable := c.noInviter
		c.mu.Unlock()
		if unreachable {
			return hostUnreachable(len(s.set.relays) > 0)
		}
		return ErrNotFound
	case <-c.ctx.Done():
		sw.Leave(c.code.Topic())
		return c.ctx.Err()
	}
	c.setPhase(phaseWaitingHost)
	actx, cancel := context.WithTimeout(c.ctx, joinAnswerTimeout)
	defer cancel()
	payload, err := invite.JoinConn(actx, peer.Conn(), s.id, c.code, nil)
	sw.Leave(c.code.Topic()) // the invite-only connection closes; the call gets its own
	if err != nil {
		return err
	}
	r, err := parseGrant(payload)
	if err != nil {
		return err
	}
	mine := s.settingsCopy().relays
	c.mu.Lock()
	c.room = r
	c.relayShare = shareRelay(mine, r.relayKeys)
	c.warning = relayMismatch(mine, r.relayKeys)
	c.mu.Unlock()
	if !c.move(callConnecting, phaseRelay) {
		return context.Canceled
	}
	if err := sw.Join(c.ctx, r.topic, nil); err != nil {
		return meetErr("lookup-failed", err, "")
	}
	select {
	case rp := <-c.relayPeer:
		kind, via := s.linkOf(rp.Conn())
		c.mu.Lock()
		c.relayLink, c.relayVia = kind, via
		c.mu.Unlock()
		conn := rp.Conn()
		if h := relayHook; h != nil {
			conn = h(c, conn)
		}
		return c.enter(conn)
	case <-time.After(s.set.FindTimeout):
		c.mu.Lock()
		defer c.mu.Unlock()
		return relayWaitReason(c.noRelay, c.relayShare)
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}
