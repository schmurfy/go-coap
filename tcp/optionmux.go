package tcp

import (
	"io"

	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/go-coap/v2/tcp/message/pool"
)

// WithMux set's multiplexer for handle requests.
func WithMux(m mux.Handler) HandlerFuncOpt {
	h := func(w *ResponseWriter, r *pool.Message) {
		muxw := &muxResponseWriter{
			w: w,
		}
		muxr := pool.ConvertTo(r)
		m.ServeCOAP(muxw, &mux.Message{
			Message:        muxr,
			SequenceNumber: r.Sequence(),
		})
	}
	return WithHandlerFunc(h)
}

type muxResponseWriter struct {
	w *ResponseWriter
}

func (w *muxResponseWriter) SetResponse(code codes.Code, contentFormat message.MediaType, d io.ReadSeeker, opts ...message.Option) error {
	return w.w.SetResponse(code, contentFormat, d, opts...)
}

func (w *muxResponseWriter) Client() mux.Client {
	return w.w.ClientConn().Client()
}
