package udp

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/go-coap/v2/net/blockwise"

	"github.com/go-ocf/go-coap/v2/net/keepalive"

	"github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/udp/client"
	"github.com/go-ocf/go-coap/v2/udp/message/pool"

	coapNet "github.com/go-ocf/go-coap/v2/net"
)

// A ServerOption sets options such as credentials, codec and keepalive parameters, etc.
type ServerOption interface {
	apply(*serverOptions)
}

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as COAP handlers.
type HandlerFunc = func(*client.ResponseWriter, *pool.Message)

type ErrorFunc = func(error)

type GoPoolFunc = func(func() error) error

type BlockwiseFactoryFunc = func(getSendedRequest func(token message.Token) (blockwise.Message, bool)) *blockwise.BlockWise

type OnNewClientConnFunc = func(cc *client.ClientConn)

var defaultServerOptions = serverOptions{
	ctx:            context.Background(),
	maxMessageSize: 64 * 1024,
	handler: func(w *client.ResponseWriter, r *pool.Message) {
		w.SetResponse(codes.NotFound, message.TextPlain, nil)
	},
	errors: func(err error) {
		fmt.Println(err)
	},
	goPool: func(f func() error) error {
		go func() {
			err := f()
			if err != nil {
				fmt.Println(err)
			}
		}()
		return nil
	},
	keepalive:                      keepalive.New(),
	blockwiseEnable:                true,
	blockwiseSZX:                   blockwise.SZX1024,
	blockwiseTransferTimeout:       time.Second * 3,
	onNewClientConn:                func(cc *client.ClientConn) {},
	transmissionNStart:             time.Second,
	transmissionAcknowledgeTimeout: time.Second * 2,
	transmissionMaxRetransmit:      4,
}

type serverOptions struct {
	ctx                            context.Context
	maxMessageSize                 int
	handler                        HandlerFunc
	errors                         ErrorFunc
	goPool                         GoPoolFunc
	keepalive                      *keepalive.KeepAlive
	net                            string
	blockwiseSZX                   blockwise.SZX
	blockwiseEnable                bool
	blockwiseTransferTimeout       time.Duration
	onNewClientConn                OnNewClientConnFunc
	transmissionNStart             time.Duration
	transmissionAcknowledgeTimeout time.Duration
	transmissionMaxRetransmit      int
}

type Server struct {
	maxMessageSize                 int
	handler                        HandlerFunc
	errors                         ErrorFunc
	goPool                         GoPoolFunc
	keepalive                      *keepalive.KeepAlive
	blockwiseSZX                   blockwise.SZX
	blockwiseEnable                bool
	blockwiseTransferTimeout       time.Duration
	onNewClientConn                OnNewClientConnFunc
	transmissionNStart             time.Duration
	transmissionAcknowledgeTimeout time.Duration
	transmissionMaxRetransmit      int

	conns             map[string]*client.ClientConn
	connsMutex        sync.Mutex
	ctx               context.Context
	cancel            context.CancelFunc
	serverStartedChan chan struct{}

	multicastRequests *sync.Map
	multicastHandler  *client.HandlerContainer
	msgID             uint32

	listen      *coapNet.UDPConn
	listenMutex sync.Mutex
}

func NewServer(opt ...ServerOption) *Server {
	opts := defaultServerOptions
	for _, o := range opt {
		o.apply(&opts)
	}

	ctx, cancel := context.WithCancel(opts.ctx)
	b := make([]byte, 4)
	rand.Read(b)
	msgID := binary.BigEndian.Uint32(b)
	serverStartedChan := make(chan struct{})

	return &Server{
		ctx:                            ctx,
		cancel:                         cancel,
		handler:                        opts.handler,
		maxMessageSize:                 opts.maxMessageSize,
		errors:                         opts.errors,
		goPool:                         opts.goPool,
		keepalive:                      opts.keepalive,
		blockwiseSZX:                   opts.blockwiseSZX,
		blockwiseEnable:                opts.blockwiseEnable,
		blockwiseTransferTimeout:       opts.blockwiseTransferTimeout,
		multicastHandler:               client.NewHandlerContainer(),
		multicastRequests:              &sync.Map{},
		msgID:                          msgID,
		serverStartedChan:              serverStartedChan,
		onNewClientConn:                opts.onNewClientConn,
		transmissionNStart:             opts.transmissionNStart,
		transmissionAcknowledgeTimeout: opts.transmissionAcknowledgeTimeout,
		transmissionMaxRetransmit:      opts.transmissionMaxRetransmit,

		conns: make(map[string]*client.ClientConn),
	}
}

func (s *Server) checkAndSetListener(l *coapNet.UDPConn) error {
	s.listenMutex.Lock()
	defer s.listenMutex.Unlock()
	if s.listen != nil {
		return fmt.Errorf("server already serve: %v", s.listen.LocalAddr().String())
	}
	s.listen = l
	close(s.serverStartedChan)
	return nil
}

func (s *Server) Serve(l *coapNet.UDPConn) error {
	if s.blockwiseSZX > blockwise.SZX1024 {
		return fmt.Errorf("invalid blockwiseSZX")
	}

	err := s.checkAndSetListener(l)
	if err != nil {
		return err
	}

	defer func() {
		s.closeSessions()
		s.listenMutex.Lock()
		defer s.listenMutex.Unlock()
		s.listen = nil
		s.serverStartedChan = make(chan struct{}, 1)
	}()

	m := make([]byte, s.maxMessageSize)
	var wg sync.WaitGroup
	for {
		buf := m
		n, raddr, err := l.ReadWithContext(s.ctx, buf)
		if err != nil {
			wg.Wait()
			select {
			case <-s.ctx.Done():
				return nil
			default:
				return err
			}
		}
		buf = buf[:n]
		cc, closeFunc, created := s.getOrCreateClientConn(l, raddr)
		if created {
			if s.onNewClientConn != nil {
				s.onNewClientConn(cc)
			}
			if s.keepalive != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := s.keepalive.Run(cc)
					if err != nil {
						s.errors(err)
					}
				}()
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				closeFunc()
			}()
		}
		err = cc.Process(buf)
		if err != nil {
			cc.Close()
			s.errors(err)
		}
	}
}

// Stop stops server without wait of ends Serve function.
func (s *Server) Stop() {
	s.cancel()
	s.closeSessions()
}

func (s *Server) closeSessions() {
	s.connsMutex.Lock()
	tmp := s.conns
	s.conns = make(map[string]*client.ClientConn)
	s.connsMutex.Unlock()
	for _, v := range tmp {
		v.Close()
	}
}

func (s *Server) conn() *coapNet.UDPConn {
	s.listenMutex.Lock()
	serverStartedChan := s.serverStartedChan
	s.listenMutex.Unlock()
	select {
	case <-serverStartedChan:
	case <-s.ctx.Done():
	}
	s.listenMutex.Lock()
	defer s.listenMutex.Unlock()
	return s.listen
}

func (s *Server) getOrCreateClientConn(UDPConn *coapNet.UDPConn, raddr *net.UDPAddr) (cc *client.ClientConn, closeFunc func(), created bool) {
	s.connsMutex.Lock()
	defer s.connsMutex.Unlock()
	key := raddr.String()
	cc = s.conns[key]
	if cc == nil {
		created = true
		var blockWise *blockwise.BlockWise
		if s.blockwiseEnable {
			blockWise = blockwise.NewBlockWise(
				bwAcquireMessage,
				bwReleaseMessage,
				s.blockwiseTransferTimeout,
				s.errors,
				false,
				bwCreateHandlerFunc(s.multicastRequests),
			)
		}
		obsHandler := client.NewHandlerContainer()
		session := NewSession(
			s.ctx,
			UDPConn,
			raddr,
			s.maxMessageSize,
		)
		closeFunc = func() {
			<-session.Context().Done()
			session.close()
		}
		cc = client.NewClientConn(
			session,
			obsHandler,
			s.multicastRequests,
			s.transmissionNStart,
			s.transmissionAcknowledgeTimeout,
			s.transmissionMaxRetransmit,
			client.NewObservationHandler(obsHandler, func(w *client.ResponseWriter, r *pool.Message) {
				h, err := s.multicastHandler.Get(r.Token())
				if err == nil {
					h(w, r)
					return
				}
				s.handler(w, r)
			}),
			s.blockwiseSZX,
			blockWise,
			s.goPool,
		)
		cc.AddOnClose(func() {
			s.connsMutex.Lock()
			defer s.connsMutex.Unlock()
			delete(s.conns, key)
		})
		s.conns[key] = cc
	}
	return cc, closeFunc, created
}

// GetMID generates a message id for UDP-coap
func (s *Server) GetMID() uint16 {
	return uint16(atomic.AddUint32(&s.msgID, 1) % 0xffff)
}
