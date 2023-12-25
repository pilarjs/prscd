package psig

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/yomorun/yomo/serverless"
)

func TestContext(t *testing.T) {
	t.Run("new context error", func(t *testing.T) {
		yctx := mockYomoCtx{read: []byte("bytes that cannot be msgpack unmarshalled")}
		_, err := NewContext(&yctx, "test")
		assert.EqualError(t, err, "msgpack: unexpected code=62 decoding map length")
	})

	t.Run("new context", func(t *testing.T) {
		event := &ChannelEvent{
			Event: "compute",
			Data:  "mock-data",
		}
		eventBytes, _ := msgpack.Marshal(&event)

		sig := &Signalling{
			Type:    "data",
			OpCode:  "1",
			Channel: "psig",
			Sid:     "123456",
			Payload: eventBytes,
			Cid:     "mock-cid",
			AppID:   "mock-app-id",
			MeshID:  "mock-mesh-id",
		}
		sigBytes, _ := msgpack.Marshal(&sig)
		yctx := mockYomoCtx{read: sigBytes}

		pctx, err := NewContext(&yctx, "test-sfn")
		assert.NoError(t, err)

		psig := pctx.ReadSignalling()
		assert.Equal(t, sig, psig)

		ev, err := pctx.ReadEvent()
		assert.NoError(t, err)
		assert.Equal(t, event, ev)

		errEvent := ChannelEvent{"compute_error", "code=-1"}
		err = pctx.WriteEvent(&errEvent)
		assert.NoError(t, err)
		assert.Equal(t, errEvent, yctx.written)
	})

}

type mockYomoCtx struct {
	read    []byte
	written ChannelEvent
}

func (c *mockYomoCtx) Data() []byte          { return c.read }
func (c *mockYomoCtx) HTTP() serverless.HTTP { return nil }
func (c *mockYomoCtx) Tag() uint32           { return 0 }

func (c *mockYomoCtx) Write(tag uint32, data []byte) error {
	fmt.Println(len(data))
	var w Signalling
	_ = msgpack.Unmarshal(data, &w)

	var ev ChannelEvent
	_ = msgpack.Unmarshal(w.Payload, &ev)
	c.written = ev
	return nil
}
