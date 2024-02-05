package chirp

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pilarjs/prscd/psig"
	"github.com/pilarjs/prscd/util"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/yomorun/yomo"
	"github.com/yomorun/yomo/pkg/trace"
	"github.com/yomorun/yomo/serverless"
)

const (
	// Endpoint is the base path of service
	Endpoint string = "/v1"
)

var log = util.Log
var allRealms sync.Map

// AuthUserAndGetYoMoCredential is used to authenticate user by `publickey` and get credential used to connect to YoMo
var AuthUserAndGetYoMoCredential func(publicKey string) (appID, credential string, ok bool)

// GetOrCreateRealm get or create realm by appID, if realm is created, it will connect to yomo zipper with credential.
func GetOrCreateRealm(appID string, credential string) (realm *node) {
	log.Debug("get or create realm", "appID", appID)
	res, ok := allRealms.LoadOrStore(appID, &node{
		MeshID: os.Getenv("MESH_ID"),
		id:     appID,
	})

	if !ok {
		log.Debug("create realm", "appID", appID)
		// connect to yomo zipper when created
		err := res.(*node).ConnectToYoMo(credential)
		// if can not connect to yomo zipper, remove this realm
		if err != nil {
			allRealms.Delete(appID)
			// Consider return nil and close connection. But currently, I am trying to let client connected to this node, next time, it will try to connect to yomo zipper again, this will fix the network problem between prscd and yomo zipper.
			// log.Error("connect to yomo zipper error: %+v", err)
			// return nil
		}
	}

	return res.(*node)
}

type node struct {
	id     string              // id is the unique id of this node
	cdic   sync.Map            // all channels on this node
	pdic   sync.Map            // all peers on this node
	Env    string              // Env describes the environment of this node, e.g. "dev", "prod"
	MeshID string              // MeshID describes the id of this node
	sndr   yomo.Source         // the yomo source used to send data to the geo-distributed network which built by yomo
	rcvr   yomo.StreamFunction // the yomo stream function used to receive data from the geo-distributed network which built by yomo
}

// AddPeer add peer to channel named `cid` on this node.
func (n *node) AddPeer(conn Connection, cid string) *Peer {
	log.Debug("node.add_peer", "remoteAddr", conn.RemoteAddr(), "cid", cid)
	peer := &Peer{
		Sid:      conn.RemoteAddr(),
		Cid:      cid,
		Channels: make(map[string]*Channel),
		conn:     conn,
		realm:    n,
	}

	n.pdic.Store(peer.Sid, peer)

	return peer
}

// RemovePeer remove peer on this node.
func (n *node) RemovePeer(pid string) {
	log.Info("node.remove_peer", "pid", pid)
	n.pdic.Delete(pid)
}

// GetOrCreateChannel get or create channel on this node.
func (n *node) GetOrAddChannel(name string) *Channel {
	channel, ok := n.cdic.LoadOrStore(name, &Channel{
		UniqID: name,
		realm:  n,
	})

	if !ok {
		log.Info("create channel", "name", name)
	}

	return channel.(*Channel)
}

// FindChannel returns the channel on this node by name.
func (n *node) FindChannel(name string) *Channel {
	ch, ok := n.cdic.Load(name)
	if !ok {
		log.Debug("channel not found", "channel", name)
		return nil
	}
	return ch.(*Channel)
}

// ConnectToYoMo connect this node to the geo-distributed network which built by yomo.
func (n *node) ConnectToYoMo(credential string) error {
	// YOMO_ZIPPER env indicates the endpoint of YoMo Zipper to connect
	log.Debug("connect to YoMo Zipper", "realm", n.id, "endpoint", os.Getenv("YOMO_ZIPPER"))

	// add open tracing
	tp, shutdown, err := trace.NewTracerProvider("prscd")
	if err == nil {
		log.Info("🛰 tracing enabled")
	}
	defer shutdown(context.Background())

	// sndr is sender to send data to other prscd nodes by YoMo
	sndr := yomo.NewSource(
		os.Getenv("YOMO_SNDR_NAME")+"-"+n.id,
		os.Getenv("YOMO_ZIPPER"),
		yomo.WithCredential(credential),
		yomo.WithTracerProvider(tp),
		yomo.WithSourceReConnect(),
	)

	// rcvr is receiver to receive data from other prscd nodes by YoMo
	rcvr := yomo.NewStreamFunction(
		os.Getenv("YOMO_RCVR_NAME")+"-"+n.id,
		os.Getenv("YOMO_ZIPPER"),
		yomo.WithSfnCredential(credential),
		yomo.WithSfnTracerProvider(tp),
		yomo.WithSfnReConnect(),
	)

	sndr.SetErrorHandler(func(err error) {
		log.Error("sndr error: %+v", err)
	})

	rcvr.SetErrorHandler(func(err error) {
		log.Error("rcvr error: %+v", err)
	})

	// connect yomo source to zipper
	err = sndr.Connect()
	if err != nil {
		return err
	}

	sfnHandler := func(ctx serverless.Context) {
		var sig *psig.Signalling
		err := msgpack.Unmarshal(ctx.Data(), &sig)
		if err != nil {
			log.Error("Read from YoMo error", "err", err, "ctx.Data()", ctx.Data())
		}
		log.Debug("got sig", "sig", sig)

		// if sig.AppID != n.id {
		// 	log.Debug("ignore message from other app", "appID", sig.AppID)
		// 	return
		// }

		channel := n.FindChannel(sig.Channel)
		if channel != nil {
			channel.Dispatch(sig)
			log.Debug("[\u21CA] dispatched to", "cid", sig.Cid)
		} else {
			log.Debug("[\u21CA] dispatch to channel failed cause of not exist", "channel", sig.Channel)
		}
	}

	// set observe data tags from yomo network by yomo stream function
	// 0x20 comes from other prscd nodes
	// 0x21 comes from backend sfn
	rcvr.SetObserveDataTags(0x20, 0x21)

	// handle data from yomo network, and dispatch to the same channel on this node.
	rcvr.SetHandler(sfnHandler)

	err = rcvr.Connect()
	if err != nil {
		return err
	}

	n.sndr = sndr
	n.rcvr = rcvr
	return nil
}

// BroadcastToYoMo broadcast presence to yomo
func (n *node) BroadcastToYoMo(sig *psig.Signalling) {
	// sig.Sid is sender's sid when sending message
	log.Debug("[\u21C8\u21C8]", "appID", sig.AppID, "sig", sig)
	buf, err := msgpack.Marshal(sig)
	if err != nil {
		log.Error("msgpack marshal: %+v", err)
		return
	}

	if n.sndr == nil {
		log.Error("************** n.sndr is nil")
		return
	}

	err = n.sndr.Write(0x20, buf)
	if err != nil {
		log.Error("broadcast to yomo error: %+v", err)
	}
}

// DumpNodeState prints the user and room information to stdout.
func DumpNodeState() {
	log.Info("Dump start --------")
	allRealms.Range(func(appID, realm interface{}) bool {
		log.Info("Realm", "appID", appID)
		realm.(*node).cdic.Range(func(k1, v1 interface{}) bool {
			log.Info("\tChannel", "name", k1)
			ch := v1.(*Channel)
			log.Info("\t\tPeers", "count", ch.getLen())
			ch.pdic.Range(func(key, value interface{}) bool {
				log.Info("\t\tPeer", "sid", key, "cid", value)
				return true
			})
			return true
		})
		return true
	})
	log.Info("Dump done --------")
}

// DumpConnectionsState prints the user and room information to stdout.
func DumpConnectionsState() {
	log.Info("Dump start --------")
	counter := make(map[string]int)

	allRealms.Range(func(appIDStr, realm interface{}) bool {
		appID := appIDStr.(string)
		log.Info("Realm", "appID", appID)
		realm.(*node).cdic.Range(func(k1, v1 interface{}) bool {
			log.Info("\tChannel", "name", k1)
			chName := k1.(string)
			ch := v1.(*Channel)
			peersCount := ch.getLen()
			// chName is like "appID|channelName", so we need to split it to get appID
			// appID := strings.Split(chName, "|")[0]
			log.Info("\t\tPeers", "appID", appID, "channel", chName, "peersCount", peersCount)
			if _, ok := counter[appID]; !ok {
				counter[appID] = peersCount
			} else {
				counter[appID] += peersCount
			}
			return true
		})
		return true
	})

	// list all counter
	for appID, count := range counter {
		log.Info("->connections", "appID", appID, "peersCount", count)
	}
	// write counter to /tmp/conns.log
	f, err := os.OpenFile("/tmp/conns.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error("failed to open file", "err", err)
	}
	defer f.Close()
	timestamp := time.Now().Unix()
	for appID, count := range counter {
		if count > 0 {
			f.WriteString(fmt.Sprintf("{\"timestamp\": %d, \"conns\": %d, \"app_id\": \"%s\", \"mesh_id\": \"%s\"}\n\r", timestamp, count, appID, os.Getenv("MESH_ID")))
		}
	}
}
