package psig

import (
	"fmt"
	"log/slog"

	"github.com/vmihailenco/msgpack/v5"
)

func marshal(tag uint32, sig *Signalling) (uint32, []byte, error) {
	// encode sig with msgpack v5
	buf, err := msgpack.Marshal(sig)
	if err != nil {
		slog.Error("marshal", "err", err)
		return 0, nil, err
	}

	return tag, buf, nil
}

// MarshalDataSig marshals the data signal with the given channel and payload.
func MarshalDataSig(channel string, payload any, cid string) (uint32, []byte, error) {
	if channel == "" {
		return 0, nil, fmt.Errorf("channel is required")
	}
	payloadBuf, err := msgpack.Marshal(payload)
	if err != nil {
		slog.Error("MarshalDataSig", "err", err)
		return 0, nil, err
	}

	sig := &Signalling{
		Type:    SigData,
		Channel: channel,
		Payload: payloadBuf,
		Cid:     cid,
	}

	return marshal(0x21, sig)
}
