// plain http tunnel

package pht

import (
	"net"

	"github.com/go-gost/core/listener"
	"github.com/go-gost/core/logger"
	md "github.com/go-gost/core/metadata"
	admission "github.com/go-gost/x/admission/wrapper"
	limiter "github.com/go-gost/x/limiter/traffic/wrapper"
	metrics "github.com/go-gost/x/metrics/wrapper"
	"github.com/go-gost/x/registry"
	xnet "github.com/gost-dev/x/extra/net"
	pht_util "github.com/gost-dev/x/extra/util/pht"
)

func init() {
	registry.ListenerRegistry().Register("pht", NewListener)
	registry.ListenerRegistry().Register("phts", NewTLSListener)
}

type phtListener struct {
	addr       net.Addr
	tlsEnabled bool
	server     *pht_util.Server
	logger     logger.Logger
	md         metadata
	options    listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &phtListener{
		logger:  options.Logger,
		options: options,
	}
}

func NewTLSListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &phtListener{
		tlsEnabled: true,
		logger:     options.Logger,
		options:    options,
	}
}

func (l *phtListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	network := "tcp"
	if xnet.IsIPv4(l.options.Addr) {
		network = "tcp4"
	}
	l.addr, err = net.ResolveTCPAddr(network, l.options.Addr)
	if err != nil {
		return
	}

	l.server = pht_util.NewServer(
		l.options.Addr,
		pht_util.TLSConfigServerOption(l.options.TLSConfig),
		pht_util.EnableTLSServerOption(l.tlsEnabled),
		pht_util.BacklogServerOption(l.md.backlog),
		pht_util.PathServerOption(l.md.authorizePath, l.md.pushPath, l.md.pullPath),
		pht_util.LoggerServerOption(l.options.Logger),
	)

	go func() {
		if err := l.server.ListenAndServe(); err != nil {
			l.logger.Error(err)
		}
	}()

	return
}

func (l *phtListener) Accept() (conn net.Conn, err error) {
	conn, err = l.server.Accept()
	if err != nil {
		return
	}
	conn = metrics.WrapConn(l.options.Service, conn)
	conn = admission.WrapConn(l.options.Admission, conn)
	conn = limiter.WrapConn(l.options.TrafficLimiter, conn)
	return
}

func (l *phtListener) Addr() net.Addr {
	return l.addr
}

func (l *phtListener) Close() (err error) {
	return l.server.Close()
}
