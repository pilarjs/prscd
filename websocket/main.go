// Package websocket serve websocket connections
package websocket

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"

	"github.com/pilarjs/prscd/chirp"
	"github.com/pilarjs/prscd/util"
)

var log = util.Log

const (
	// DurationOfPing describes the interval of ping
	DurationOfPing = 10 * time.Second
)

// ListenAndServe create the websocket server
func ListenAndServe(addr string, config *tls.Config) {
	// create TCP listener
	lp, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer lp.Close()

	// wrap TCP listener with TLS
	ln := tls.NewListener(lp, config)
	defer ln.Close()

	log.Info("prscd start WebSocket Server", "addr", ln.Addr())

	// var node = chirp.Node

	for {
		// TCP has new connection
		conn, err := ln.Accept()
		if err != nil {
			log.Error("ln.accept error", "err", err)
			conn.Close()
			continue
		}

		var cuid, appID, credential string // Pilar.js client user id

		rejectionHeader := ws.RejectionHeader(ws.HandshakeHeaderString("X-Prscd-Version: v2\r\nX-Prscd-MeshID: " + os.Getenv("MESH_ID") + "\r\n"))

		// HTTP layer
		u := ws.Upgrader{
			OnRequest: func(req []byte) error {
				// the request url should be like: /v1?id=xxx&publickey=xxx
				url, err := url.ParseRequestURI(string(req))
				if err != nil {
					log.Error("url parse error: %s", err)
					return ws.RejectConnectionError(
						ws.RejectionStatus(500),
						rejectionHeader,
						ws.RejectionReason("url parse error"),
					)
				}
				log.Debug("ws upgrade", "path", url.Path, "query", url.Query())
				if url.Path != chirp.Endpoint {
					return ws.RejectConnectionError(
						ws.RejectionStatus(404),
						rejectionHeader,
						ws.RejectionReason("path not allowed"),
					)
				}
				cuid = url.Query().Get("id")
				if cuid == "" {
					return ws.RejectConnectionError(
						ws.RejectionStatus(401),
						rejectionHeader,
						ws.RejectionReason("id must not be empty"),
					)
				}
				// publickey can be used for identify user if developer want integrate with other systems
				authPublicKey := url.Query().Get("publickey")
				if authPublicKey == "" {
					return ws.RejectConnectionError(
						ws.RejectionStatus(401),
						rejectionHeader,
						ws.RejectionReason("publickey must not be empty"),
					)
				}
				var ok bool
				appID, credential, ok = chirp.AuthUserAndGetYoMoCredential(authPublicKey)
				if !ok {
					return ws.RejectConnectionError(
						ws.RejectionStatus(403),
						rejectionHeader,
						ws.RejectionReason("illegal public key"),
					)
				}
				log.Info("ws.upgrade", "queryId", cuid, "appID", appID)
				return nil
			},
			OnHeader: func(key, value []byte) error {
				// implement this method to check request headers if needed
				// log.Info("header: %s=%s", string(key), string(value))
				return nil
			},
			OnBeforeUpgrade: func() (ws.HandshakeHeader, error) {
				// before upgrade to websocket, logic can be implemented here
				return ws.HandshakeHeaderHTTP(http.Header{
					"X-Prscd-VER":    []string{"v2.1.1"},
					"X-Prscd-MESHID": []string{os.Getenv("MESH_ID")},
				}), nil
			},
		}

		// zero-copy reuse the TCP connection
		p, err := u.Upgrade(conn)
		if err != nil {
			if err == io.EOF {
				log.Inspect("connection closed by peer.", "remoteAddr", conn.RemoteAddr().String())
			} else {
				log.Info("[ws] new conn", "remoteAddr", conn.RemoteAddr().String())
				// if is rejected connection error, send close frame to client
				var rejectErr *ws.ConnectionRejectedError
				if errors.As(err, &rejectErr) {
					ws.WriteFrame(conn, ws.NewCloseFrame(ws.NewCloseFrameBody(ws.StatusCode(rejectErr.StatusCode()), rejectErr.Error())))
					log.Error("u.upgrade reject error, close connection", "remoteAddr", conn.RemoteAddr().String(), "err", err)
				} else {
					log.Error("u.upgrade unknown error, close connection", "remoteAddr", conn.RemoteAddr().String(), "err", err)
				}
			}

			conn.Write(ws.CompiledClose)
			conn.Close()
			continue
		}

		log.Info("upgrade success, start serving", "remoteAddr", conn.RemoteAddr().String(), "handshake", p)

		// now, the authorization is done, we can create realm instance by appID
		node := chirp.GetOrCreateRealm(appID, credential)

		// if can not connect to yomo zipper, close connection
		if node == nil {
			conn.Close()
			return
		}

		// create peer instance after Websocket handshake
		pconn := chirp.NewWebSocketConnection(conn)
		peer := node.AddPeer(pconn, cuid)
		log.Debug("Upgrade done!", "sid", peer.Sid, "cid", peer.Cid)

		keepaliveDone := make(chan bool)
		go func(c net.Conn) {
			// according to https://tools.ietf.org/html/rfc6455#section-5.5.2, Web Browsers will not send Ping frame,
			// backend server should send Ping frame to keep connection alive, and Web Browsers will auto reply Pong frame when receive Ping frame. But in Chrome DevTools, Ping/Pong frame is not shown.
			ticker := time.NewTicker(DurationOfPing)
			defer ticker.Stop()
			for {
				select {
				case <-keepaliveDone:
					log.Debug("ticker done", "sid", peer.Sid)
					return
				case <-ticker.C:
					// c.Write(generatePingFrame())
					pconn.RawWrite(generatePingFrame())
				}
			}
		}(conn)

		// handle WebSocket requests
		go func() {
			defer conn.Close()
			defer close(keepaliveDone)

			for {
				// read data
				header, r, err := wsutil.NextReader(conn, ws.StateServerSide)
				if err != nil {
					log.Error("read from ws error", "err", err)
					switch et := err.(type) {
					case wsutil.ClosedError:
						// Client close the connection:
						log.Info("[client disconnect] ClosedError", "code", et.Code, "reason", et.Reason)
					default:
						// detect connection has been closed
						log.Info("read error", "code", et, "err", err)
						// send Close frame to client
						conn.Write(ws.MustCompileFrame(ws.NewCloseFrame(ws.NewCloseFrameBody(ws.StatusNormalClosure, "bye-default"))))
					}
					// clear connection
					peer.Disconnect()
					return
				}

				// handle Websocket Control Frame: https://www.rfc-editor.org/rfc/rfc6455#section-5.5
				// there are three types of Control Frame: 0x08(Close), 0x09(Ping) and 0x0A(Pong)
				// be careful that Control frames can be interjected in the middle of a fragmented message.
				if header.OpCode.IsControl() {
					// Close Frame
					if header.OpCode == ws.OpClose {
						log.Debug(">GOT CLOSE", "sid", peer.Sid)
						peer.Disconnect()
						wsutil.ControlFrameHandler(conn, ws.StateServerSide)
						// conn.Write(ws.MustCompileFrame(ws.NewCloseFrame(ws.NewCloseFrameBody(ws.StatusNormalClosure, "bye"))))
						// conn.Close()
						closeConn(conn, "bye")
						return
					}

					// Pong Frame
					if header.OpCode == ws.OpPong {
						handlePongFrame(peer.Sid, r, header)
						continue
					}

					log.Debug(">GOT Unhandled Control Frame", "sid", peer.Sid, "OpCode", header.OpCode)
					wsutil.ControlFrameHandler(conn, ws.StateServerSide)

					continue
				}

				// handle Websocket Data Frames: https://www.rfc-editor.org/rfc/rfc6455#section-5.6
				// only accept Binary mode message, will break if receive Text mode message
				if header.OpCode == ws.OpText {
					log.Error("peer sent text which not allowed", "sid", peer.Sid)
					// https://datatracker.ietf.org/doc/html/rfc6455#section-7.4.1 1003
					// conn.Write(ws.MustCompileFrame(ws.NewCloseFrame(ws.NewCloseFrameBody(ws.StatusUnsupportedData, "no text allowed"))))
					conn.Close()
					closeConn(conn, "no text allowed")
					break
				}

				_ = peer.HandleSignal(r)
			}
		}()
	}
}

// generatePingFrame return a Ping Frame
func generatePingFrame() []byte {
	// according to RFC6455: https://www.rfc-editor.org/rfc/rfc6455#section-5.5.2,
	// Application Data can be carried by Ping frame, and the payload will be returned in Pong frame from Web Browser automatically, so we can calculate the RTT by this.
	ts := time.Now().UnixMilli()
	tsbuf := make([]byte, 8)
	binary.BigEndian.PutUint64(tsbuf, uint64(ts))
	pf := ws.MustCompileFrame(ws.NewPingFrame(tsbuf))
	log.Debug("PING Payload", "len", len(pf), "bytes", fmt.Sprintf("% X", pf))
	return pf
}

// handlePongFrame handle Pong Frame from Web Browser
func handlePongFrame(sid string, r io.Reader, header ws.Header) error {
	// read the Application Data from Pong frame
	buf := make([]byte, header.Length)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		log.Error("read PONG payload error", "err", err)
		return err
	}
	// calculate the RTT and prints to stdout
	appData := int64(binary.BigEndian.Uint64(buf))
	now := time.Now().UnixMilli()
	// log.Inspect("\tPONG Payload", "sid", sid, "len", len(buf), "val", appData, "𝚫", now-appData)
	log.Debug("[PONG]", "sid", sid, "len", len(buf), "buf", fmt.Sprintf("% X", buf), "val", appData, "𝚫", now-appData)
	return nil
}

// closeConn send Close Frame to client and close the connection
func closeConn(conn net.Conn, reason string) {
	ws.WriteFrame(conn, ws.NewCloseFrame(ws.NewCloseFrameBody(ws.StatusNormalClosure, reason)))
	conn.Close()
}
