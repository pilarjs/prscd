package chirp

import (
	"errors"
	"io"
	"sync"

	"github.com/pilarjs/prscd/psig"
	"github.com/vmihailenco/msgpack/v5"
)

// Peer describes user on this node.
type Peer struct {
	// Sid describes the unique id of this peer on this node, only used for backend.
	Sid string
	// Cid describes the unique id of this peer on who geo-distributed network, set by developer.
	Cid string
	// Channel describes the channel which this peer joined.
	Channels map[string]*Channel
	// conn is the connection of this peer.
	conn  Connection
	mu    sync.Mutex
	realm *node
}

// Join this peer to channel named `channelName`.
func (p *Peer) Join(channelName string) {
	// find channel on this node, if not exist, create it.
	c := p.realm.GetOrAddChannel(channelName)

	// add peer to this channel
	c.AddPeer(p)

	// and this channel to peer's channel list
	p.Channels[channelName] = c

	// ACK to peer has joined
	p.NotifyBack(NewSigChannelJoined(channelName))

	log.Info("peer.join_chanel ACK", "sid", p.Sid, "uniqID", c.UniqID, "cid", p.Cid)
}

// NotifyBack to peer with message.
func (p *Peer) NotifyBack(sig *psig.Signalling) {
	resp, err := msgpack.Marshal(sig)
	if err != nil {
		log.Error("msgpack marshal error", "err", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	err = p.conn.Write(resp)

	if err != nil {
		log.Error("NotifyBack error", "err", err)
	}
	log.Debug("SND>", "sid", p.Sid, "sig", sig)
}

// Leave a channel
func (p *Peer) Leave(channelName string) {
	// remove channel from peer's channel list
	p.mu.Lock()
	delete(p.Channels, channelName)
	p.mu.Unlock()

	// remove peer from channel's peer list
	c := p.realm.FindChannel(channelName)
	if c == nil {
		log.Error("peer.Leave(), channel is nil.", "pid", p.Sid, "channel", channelName)
		return
	}

	c.RemovePeer(p)

	// Notify others on this channel that this peer has left
	c.Broadcast(NewSigPeerOffline(channelName, p))
	log.Info("peer.leave", "sid", p.Sid, "uniqID", c.UniqID)
}

// Disconnect clears resources of this peer when leave.
func (p *Peer) Disconnect() {
	log.Info("peer.disconnect", "sid", p.Sid)
	// wipe this peer from all channels joined before
	for _, ch := range p.Channels {
		p.Leave(ch.UniqID)
	}
	// wipe this peer from current node
	p.realm.RemovePeer(p.Sid)
}

// BroadcastToChannel will broadcast message to channel.
func (p *Peer) BroadcastToChannel(sig *psig.Signalling) {
	sig.Cid = p.Cid
	c := p.Channels[sig.Channel]
	if c == nil {
		log.Error("peer.broadcastToChannel error, channel not exist", "channel", sig.Channel)
		return
	}

	c.Broadcast(sig)
}

// HandleSignal handle message sent from connection.
func (p *Peer) HandleSignal(r io.Reader) error {
	decoder := msgpack.NewDecoder(r)
	sig := &psig.Signalling{}
	if err := decoder.Decode(sig); err != nil {
		log.Error("msgpack.decode err, ignore", "err", err)
		return err
	}

	// p.Sid is the id of connection, set by backend.
	sig.Sid = p.Sid
	log.Debug("[>RCV]", "sid", p.Sid, "sig", sig)

	if sig.Type == psig.SigControl {
		// handle the Control Signalling
		switch sig.OpCode {
		case psig.OpChannelJoin: // `channel_join` signalling
			// join channel
			p.Join(sig.Channel)
		case psig.OpState: // `peer_state` signalling
			// Alice can notify Bob that her state has been updated, also,
			// Bob can use this signalling to initialize or update Alice's state
			if sig.Sid != "" && sig.Cid != "" {
				// if peer sid and client id are both set, then update the client id of this peer
				p.Cid = sig.Cid
				log.Info("peer state new ClientID", "sid", p.Sid, "cid", p.Cid)
			}
			p.BroadcastToChannel(sig)
		case psig.OpPeerOffline: // `peer_offline` signalling
			p.Leave(sig.Channel)
		case psig.OpPeerOnline: // `peer_online` signalling
			p.BroadcastToChannel(sig)
		default:
			log.Error("Unknown control opcode", "code", sig.OpCode)
		}
	} else if sig.Type == psig.SigData {
		// handle the Data Signalling
		p.BroadcastToChannel(sig)
	} else {
		log.Error("ILLEGAL sig.Type, should be `data` or `control`", "sig", sig)
		return errors.New("ILLEGAL sig.Type, should be `data` or `control`")
	}

	return nil
}
