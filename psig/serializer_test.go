package psig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

func TestMarshalDataSig(t *testing.T) {
	channel := "testChannel"
	payload := "testPayload"
	cid := "testCid"

	expectedTag, expectedBuf, err := MarshalDataSig(channel, payload, cid)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	payloadBuf, err := msgpack.Marshal(payload)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	sig := &Signalling{
		Type:    SigData,
		Channel: channel,
		Payload: payloadBuf,
		Cid:     cid,
	}

	tag, buf, err := marshal(0x21, sig)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	assert.Equal(t, tag, expectedTag)
	assert.Equal(t, buf, expectedBuf)
}

func TestMarshalDataSigWithoutChannel(t *testing.T) {
	channel := ""
	payload := "testPayload"
	cid := "testCid"

	expectedTag, expectedBuf, err := MarshalDataSig(channel, payload, cid)
	assert.EqualValues(t, 0, expectedTag)
	assert.EqualValues(t, nil, expectedBuf)
	assert.EqualError(t, err, "channel is required")
}
