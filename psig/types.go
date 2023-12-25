// Package psig describes the communication protocol.
package psig

import "fmt"

const (
	// SigControl describes Control Signal
	SigControl = "control"
	// SigData describes Data Signal
	SigData = "data"
)

const (
	// OpChannelJoin describes peer join a channel. If it is client->server, means the peer is requesting to join the channel; if it's server->client, means the peer has joined the channel.
	OpChannelJoin = "channel_join"
	// OpPeerOffline describes peer leave a channel.
	OpPeerOffline = "peer_offline"
	// OpPeerOnline only used in server->client, notify others in the channel that the peer has joined the channel.
	OpPeerOnline = "peer_online"
	// OpState only used in client->client, notify others in the channel that the peer's state has been updated.
	OpState = "peer_state"
)

// Signalling describes the message format on this geo-distributed network.
type Signalling struct {
	Type    string `msgpack:"t"`              // Type describes the type of signalling, `Data Signal` or `Control Signal`
	OpCode  string `msgpack:"op,omitempty"`   // OpCode describes the operation type of signalling
	Channel string `msgpack:"c"`              // Channel describes the channel
	Sid     string `msgpack:"sid,omitempty"`  // Sid describes the peer id on this node in backend
	Payload []byte `msgpack:"pl,omitempty"`   // Payload describes the payload data of signalling
	Cid     string `msgpack:"p"`              // Cid describes the client id of peer, set by developer
	AppID   string `msgpack:"app,omitempty"`  // AppID describes the app_id
	MeshID  string `msgpack:"mesh,omitempty"` // MeshID describes the mesh_id of this node
}

// String returns the string representation of signalling.
func (sig *Signalling) String() string {
	return fmt.Sprintf("meshID:%s, appID:%s, type:%s, op:%s, ch:%s, sid:%s, cid:%s, payload:(%d)", sig.MeshID, sig.AppID, sig.Type, sig.OpCode, sig.Channel, sig.Sid, sig.Cid, len(sig.Payload))
}

// Clone a signalling.
func (sig *Signalling) Clone() Signalling {
	return Signalling{
		Type:    sig.Type,
		OpCode:  sig.OpCode,
		Channel: sig.Channel,
		Sid:     sig.Sid,
		Payload: sig.Payload,
		Cid:     sig.Cid,
		AppID:   sig.AppID,
		MeshID:  sig.MeshID,
	}
}

// ChannelEvent is Presencejs Channel event data structure used in channel.broadcast() and channel.subscribe()
type ChannelEvent struct {
	Event string `msgpack:"event"`
	Data  string `msgpack:"data"`
}
