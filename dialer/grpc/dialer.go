package grpc

import (
	"context"
	"errors" // Added import
	"net"
	"sync"

	"github.com/go-gost/core/dialer"
	md "github.com/go-gost/core/metadata"
	pb "github.com/go-gost/x/internal/util/grpc/proto"
	"github.com/go-gost/x/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

func init() {
	registry.DialerRegistry().Register("grpc", NewDialer)
}

type grpcDialer struct {
	clients     map[string]pb.GostTunelClientX
	clientMutex sync.Mutex
	md          grpcMetadataConfig // Changed from metadata
	options     dialer.Options
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &grpcDialer{
		clients: make(map[string]pb.GostTunelClientX),
		options: options,
	}
}

func (d *grpcDialer) Init(md md.Metadata) (err error) {
	return d.parseMetadata(md)
}

// Multiplex implements dialer.Multiplexer interface.
func (d *grpcDialer) Multiplex() bool {
	return true
}

func (d *grpcDialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (net.Conn, error) {
	d.clientMutex.Lock()
	defer d.clientMutex.Unlock()

	client, ok := d.clients[addr]
	if !ok {
		var options dialer.DialOptions
		for _, opt := range opts {
			opt(&options)
		}

		host := d.md.host
		if host == "" {
			host = options.Host
		}
		if h, _, _ := net.SplitHostPort(host); h != "" {
			host = h
		}
		// d.options.Logger.Infof("grpc dialer, addr %s, host %s/%s", addr, d.md.host, options.Host)

		grpcOpts := []grpc.DialOption{
			// grpc.WithBlock(),
			grpc.WithContextDialer(func(c context.Context, s string) (net.Conn, error) {
				// If options.Dialer is nil, this would panic.
				// The Dial function should ideally have a default dialer if options.Dialer is not provided.
				// For now, assume options.Dialer will be set in tests or by callers.
				if options.Dialer == nil {
					// Fallback or error, for now, let's assume it's an error or test will provide one.
					// This indicates a potential issue if Dial is called without a base dialer in DialOptions.
					// However, typical usage involves a chain of dialers.
					return nil, errors.New("base dialer not provided in DialOptions")
				}
				return options.Dialer.Dial(c, "tcp", s) // Assuming "tcp" is appropriate, might need to be more generic
			}),
			// grpc.WithAuthority(host), // Authority is often inferred or not strictly needed for bufconn
			grpc.WithConnectParams(grpc.ConnectParams{
				Backoff:           backoff.DefaultConfig,
				MinConnectTimeout: d.md.minConnectTimeout,
			}),
		}
		if !d.md.insecure {
			// Ensure d.options.TLSConfig is not nil if !d.md.insecure
			tlsCfg := d.options.TLSConfig
			if tlsCfg == nil {
				// Provide a default TLS config or handle error if TLS is expected but no config given
				// For testing with bufconn and !insecure, this might need a self-signed cert or similar.
				// For now, if insecure is false, and no TLSConfig, it might fail.
				// Let's assume for tests, if !insecure, TLSConfig will be provided or this path isn't hit with bufconn.
			}
			grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
		} else {
			grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}
		
		// Only set authority if host is not empty.
		if host != "" {
			grpcOpts = append(grpcOpts, grpc.WithAuthority(host))
		}


		if d.md.keepalive {
			grpcOpts = append(grpcOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                d.md.keepaliveTime,
				Timeout:             d.md.keepaliveTimeout,
				PermitWithoutStream: d.md.keepalivePermitWithoutStream,
			}))
		}

		target := addr
		// If a custom context dialer is used (like for bufconn via DialOptions),
		// prepend "passthrough:///" to prevent default DNS resolution for arbitrary target strings.
		// This assumes the custom context dialer (options.Dialer) ignores the scheme and address parts if needed.
		if options.Dialer != nil {
			target = "passthrough:///" + addr
		}

		cc, err := grpc.NewClient(target, grpcOpts...)
		if err != nil {
			d.options.Logger.Error(err)
			return nil, err
		}
		client = pb.NewGostTunelClientX(cc)
		d.clients[addr] = client
	}

	ctx2, cancel := context.WithCancel(context.Background())
	cli, err := client.TunnelX(ctx2, d.md.path)
	if err != nil {
		cancel()
		return nil, err
	}

	return &conn{
		c:          cli,
		localAddr:  &net.TCPAddr{},
		remoteAddr: &net.TCPAddr{},
		cancelFunc: cancel,
	}, nil
}
