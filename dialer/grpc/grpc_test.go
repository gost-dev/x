package grpc

import (
	"context"
	"errors" // Added import
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/go-gost/core/dialer"
	mdata "github.com/go-gost/core/metadata"
	xlogger "github.com/go-gost/x/logger"
	xmd "github.com/go-gost/x/metadata"
	grpcmeta "google.golang.org/grpc/metadata" 
	pb "github.com/go-gost/x/internal/util/grpc/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	// "google.golang.org/grpc/codes" // Unused
	// "google.golang.org/grpc/credentials/insecure" // Unused
	// "google.golang.org/grpc/status" // Unused
	"google.golang.org/grpc/test/bufconn"
)

// mockCoreMetadata is a simple implementation of metadata.Metadata for testing.
type mockCoreMetadata struct {
	m map[string]any
}

func newMockCoreMetadata(m map[string]any) mdata.Metadata {
	if m == nil {
		return xmd.NewMetadata(nil) 
	}
	return xmd.NewMetadata(m)
}

func TestParseMetadata(t *testing.T) {
	defaultLogger := xlogger.NewLogger() 
	tests := []struct {
		name     string
		input    mdata.Metadata
		expected grpcMetadataConfig 
		wantErr  bool
	}{
		{
			name:  "empty metadata",
			input: newMockCoreMetadata(nil),
			expected: grpcMetadataConfig{ 
				insecure:                     false,
				host:                         "",
				path:                         "",
				keepalive:                    false,
				keepaliveTime:                0, 
				keepaliveTimeout:             0, 
				keepalivePermitWithoutStream: false,
				minConnectTimeout:            30 * time.Second,
			},
		},
		{
			name: "insecure true",
			input: newMockCoreMetadata(map[string]any{
				"grpc.insecure": true,
			}),
			expected: grpcMetadataConfig{
				insecure:          true,
				minConnectTimeout: 30 * time.Second,
				keepaliveTime:     0,
				keepaliveTimeout:  0,
			},
		},
		{
			name: "insecure via alias",
			input: newMockCoreMetadata(map[string]any{
				"insecure": "true", 
			}),
			expected: grpcMetadataConfig{
				insecure:          true,
				minConnectTimeout: 30 * time.Second,
				keepaliveTime:     0,
				keepaliveTimeout:  0,
			},
		},
		{
			name: "host set",
			input: newMockCoreMetadata(map[string]any{
				"host": "example.com",
			}),
			expected: grpcMetadataConfig{
				host:              "example.com",
				minConnectTimeout: 30 * time.Second,
				keepaliveTime:     0,
				keepaliveTimeout:  0,
			},
		},
		{
			name: "path set",
			input: newMockCoreMetadata(map[string]any{
				"grpc.path": "/MyService.Tunnel",
			}),
			expected: grpcMetadataConfig{
				path:              "/MyService.Tunnel",
				minConnectTimeout: 30 * time.Second,
				keepaliveTime:     0,
				keepaliveTimeout:  0,
			},
		},
		{
			name: "keepalive enabled with custom values",
			input: newMockCoreMetadata(map[string]any{
				"keepalive":                 true,
				"keepalive.time":            "60s",
				"keepalive.timeout":         "10s",
				"keepalive.permitWithoutStream": true, 
			}),
			expected: grpcMetadataConfig{
				keepalive:                    true,
				keepaliveTime:                60 * time.Second,
				keepaliveTimeout:             10 * time.Second,
				keepalivePermitWithoutStream: true,
				minConnectTimeout:            30 * time.Second,
			},
		},
		{
			name: "keepalive enabled with default time/timeout",
			input: newMockCoreMetadata(map[string]any{
				"keepalive": true,
			}),
			expected: grpcMetadataConfig{
				keepalive:                    true,
				keepaliveTime:                30 * time.Second, 
				keepaliveTimeout:             30 * time.Second, 
				keepalivePermitWithoutStream: false,
				minConnectTimeout:            30 * time.Second,
			},
		},
		{
			name: "keepalive enabled with invalid time string",
			input: newMockCoreMetadata(map[string]any{
				"keepalive":      true,
				"keepalive.time": "invalid", 
			}),
			expected: grpcMetadataConfig{
				keepalive:                    true,
				keepaliveTime:                30 * time.Second, 
				keepaliveTimeout:             30 * time.Second, 
				keepalivePermitWithoutStream: false,
				minConnectTimeout:            30 * time.Second,
			},
		},
		{
			name: "minConnectTimeout custom",
			input: newMockCoreMetadata(map[string]any{
				"minConnectTimeout": "5s",
			}),
			expected: grpcMetadataConfig{
				minConnectTimeout: 5 * time.Second,
				keepaliveTime:     0,
				keepaliveTimeout:  0,
			},
		},
		{
			name: "minConnectTimeout invalid",
			input: newMockCoreMetadata(map[string]any{
				"minConnectTimeout": "invalid",
			}),
			expected: grpcMetadataConfig{
				minConnectTimeout: 30 * time.Second, 
				keepaliveTime:     0,
				keepaliveTimeout:  0,
			},
		},
		{
			name: "all options set",
			input: newMockCoreMetadata(map[string]any{
				"grpcInsecure":                "true",
				"grpc.host":                   "server.name",
				"path":                        "/service/path",
				"grpc.keepalive":              true,
				"grpc.keepalive.time":         "15s",
				"grpc.keepalive.timeout":      "5s",
				"keepalive.permitWithoutStream": false, 
				"grpc.minConnectTimeout":      "3s",
			}),
			expected: grpcMetadataConfig{
				insecure:                     true,
				host:                         "server.name",
				path:                         "/service/path",
				keepalive:                    true,
				keepaliveTime:                15 * time.Second,
				keepaliveTimeout:             5 * time.Second,
				keepalivePermitWithoutStream: false,
				minConnectTimeout:            3*time.Second, 
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDialer(dialer.LoggerOption(defaultLogger)).(*grpcDialer)
			err := d.parseMetadata(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.expected, d.md, "parsed metadata mismatch")
		})
	}
}

// --- Mock gRPC Server and Services ---
const bufSize = 1024 * 1024

// mockTunnelServer is a simple implementation of GostTunelServer
type mockTunnelServer struct {
	pb.UnimplementedGostTunelServer
	mu         sync.Mutex
	chunks     chan *pb.Chunk 
	lastdata   []byte         
	recvErr    error          
	sendErr    error          
}

func newMockTunnelServer() *mockTunnelServer {
	return &mockTunnelServer{
		chunks: make(chan *pb.Chunk, 10), 
	}
}

func (s *mockTunnelServer) Tunnel(stream pb.GostTunel_TunnelServer) error {
	go func() {
		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				s.mu.Lock()
				if s.recvErr == nil { s.recvErr = err }
				s.mu.Unlock()
				return
			}
			s.mu.Lock()
			s.lastdata = append(s.lastdata, chunk.Data...)
			s.mu.Unlock()
		}
	}()

	for chunk := range s.chunks { 
		s.mu.Lock()
		sendErr := s.sendErr 
		s.mu.Unlock()
		if sendErr != nil {
			return sendErr
		}
		if err := stream.Send(chunk); err != nil {
			return err
		}
	}
	return nil 
}


func startMockServer() (*grpc.Server, *bufconn.Listener, *mockTunnelServer) {
	listener := bufconn.Listen(bufSize) 
	s := grpc.NewServer()
	tunnelServer := newMockTunnelServer()
	pb.RegisterGostTunelServer(s, tunnelServer)
	go func() {
		if err := s.Serve(listener); err != nil {
			// log.Printf("Mock server exited with error: %v", err)
		}
	}()
	return s, listener, tunnelServer
}

// bufconnDialer is a simple dialer.Dialer for bufconn.
type bufconnDialer struct {
	lis *bufconn.Listener
}

func (d *bufconnDialer) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	return d.lis.Dial()
}


// --- Dialer Tests ---
func TestGrpcDialer_Dial_Successful(t *testing.T) {
	// serverPath := "/test/tunnel" // Unused
	grpcServer, listener, mockSrv := startMockServer() 
	defer grpcServer.Stop()

	d := NewDialer(dialer.LoggerOption(xlogger.NewLogger())).(*grpcDialer)
	
	md := newMockCoreMetadata(map[string]any{
		"path":     "/GostTunel/Tunnel", // Try fully qualified-like path
		"insecure": true,
	})
	err := d.Init(md)
	assert.NoError(t, err)

	dialerOpt := func(o *dialer.DialOptions) {
		o.Dialer = &bufconnDialer{lis: listener}
	}

	conn, err := d.Dial(context.Background(), "bufnet", dialerOpt) 
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	if conn == nil { 
		t.FailNow()
	}
	defer conn.Close()

	testMsg := []byte("hello world")
	
	go func() {
		mockSrv.chunks <- &pb.Chunk{Data: testMsg} 
		close(mockSrv.chunks) 
	}()
	
	nWrite, err := conn.Write(testMsg)
	assert.NoError(t, err)
	assert.Equal(t, len(testMsg), nWrite)

	readBuf := make([]byte, len(testMsg)) 
	nRead, readErr := conn.Read(readBuf) 
	
	if readErr != nil && readErr != io.EOF {
		t.Fatalf("Read failed unexpectedly: %v", readErr)
	}
	assert.Equal(t, len(testMsg), nRead)
	assert.Equal(t, testMsg, readBuf[:nRead])

	mockSrv.mu.Lock()
	assert.Equal(t, testMsg, mockSrv.lastdata, "Server did not receive correct data")
	mockSrv.mu.Unlock()

	_, readErrAfterClose := conn.Read(make([]byte, 1))
	assert.Equal(t, io.EOF, readErrAfterClose)
}


func TestGrpcDialer_Init(t *testing.T) {
	d := NewDialer().(*grpcDialer)
	m := make(map[string]any)
	m["host"] = "testhost.com"
	m["insecure"] = true
	md := newMockCoreMetadata(m)

	err := d.Init(md)
	assert.NoError(t, err)
	assert.Equal(t, "testhost.com", d.md.host)
	assert.True(t, d.md.insecure)
}

func TestGrpcDialer_Multiplex(t *testing.T) {
	d := NewDialer().(*grpcDialer)
	assert.True(t, d.Multiplex(), "Multiplex should return true for grpcDialer")
}


// --- Conn Tests ---

type mockGostTunel_TunnelClient struct {
	pb.GostTunel_TunnelClient 
	ctx                       context.Context
	cancelFunc                context.CancelFunc
	recvChan                  chan *pb.Chunk 
	recvErr                   error          
	sendErr                   error          
	closeSendErr              error          
	sentData                  [][]byte       
	mu                        sync.Mutex
}

func newMockTunnelClient(ctx context.Context) *mockGostTunel_TunnelClient {
	clientCtx, cancel := context.WithCancel(ctx)
	return &mockGostTunel_TunnelClient{
		ctx:        clientCtx,
		cancelFunc: cancel,
		recvChan:   make(chan *pb.Chunk, 5), 
		sentData:   make([][]byte, 0),
	}
}

func (m *mockGostTunel_TunnelClient) Recv() (*pb.Chunk, error) {
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	select {
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	case chunk, ok := <-m.recvChan:
		if !ok { 
			return nil, io.EOF
		}
		return chunk, nil
	}
}

func (m *mockGostTunel_TunnelClient) Send(chunk *pb.Chunk) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sentData = append(m.sentData, chunk.Data)
	return nil
}

func (m *mockGostTunel_TunnelClient) CloseSend() error {
	return m.closeSendErr
}

func (m *mockGostTunel_TunnelClient) Context() context.Context {
	return m.ctx
}

func (m *mockGostTunel_TunnelClient) Header() (grpcmeta.MD, error) { return nil, nil }
func (m *mockGostTunel_TunnelClient) Trailer() grpcmeta.MD      { return nil }


func TestGrpcConn_Read(t *testing.T) {
	ctx := context.Background()
	
	t.Run("successful read", func(t *testing.T) {
		mockClient := newMockTunnelClient(ctx)
		grpcConn := &conn{
			c:          mockClient,
			localAddr:  &net.TCPAddr{},
			remoteAddr: &net.TCPAddr{},
			cancelFunc: mockClient.cancelFunc,
		}
		defer grpcConn.Close()

		testData := []byte("hello")
		mockClient.recvChan <- &pb.Chunk{Data: testData}
		close(mockClient.recvChan) 

		buf := make([]byte, 10)
		n, err := grpcConn.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, len(testData), n)
		assert.Equal(t, testData, buf[:n])

		_, err = grpcConn.Read(buf)
		assert.Equal(t, io.EOF, err)
	})

	t.Run("read with internal buffer", func(t *testing.T) {
		mockClient := newMockTunnelClient(ctx)
		grpcConn := &conn{
			c:          mockClient,
			rb:         []byte("buffered"), 
			cancelFunc: mockClient.cancelFunc,
		}
		defer grpcConn.Close()
		
		buf := make([]byte, 5)
		n, err := grpcConn.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("buffe"), buf[:n])
		assert.Equal(t, []byte("red"), grpcConn.rb, "Internal buffer not consumed correctly")

		n, err = grpcConn.Read(buf) 
		assert.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, []byte("red"), buf[:n])
		assert.Empty(t, grpcConn.rb, "Internal buffer should be empty")
	})

	t.Run("read error from Recv", func(t *testing.T) {
		mockClient := newMockTunnelClient(ctx)
		mockClient.recvErr = errors.New("recv error")
		grpcConn := &conn{c: mockClient, cancelFunc: mockClient.cancelFunc}
		defer grpcConn.Close()

		buf := make([]byte, 10)
		_, err := grpcConn.Read(buf)
		assert.Error(t, err)
		assert.Equal(t, "recv error", err.Error())
	})

	t.Run("context done during read", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(ctx)
		mockClient := newMockTunnelClient(canceledCtx) 
		
		grpcConn := &conn{
			c:          mockClient,
			cancelFunc: func() { mockClient.cancelFunc() }, 
		}
		
		cancel() 

		buf := make([]byte, 10)
		_, err := grpcConn.Read(buf)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestGrpcConn_Write(t *testing.T) {
	ctx := context.Background()
	mockClient := newMockTunnelClient(ctx)
	grpcConn := &conn{
		c:          mockClient,
		localAddr:  &net.TCPAddr{},
		remoteAddr: &net.TCPAddr{},
		cancelFunc: mockClient.cancelFunc,
	}
	defer grpcConn.Close()

	testData := []byte("write this")
	n, err := grpcConn.Write(testData)
	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)

	mockClient.mu.Lock()
	assert.Len(t, mockClient.sentData, 1, "Data should have been sent once")
	assert.Equal(t, testData, mockClient.sentData[0], "Sent data mismatch")
	mockClient.mu.Unlock()

	mockClient.sendErr = errors.New("send error")
	_, err = grpcConn.Write([]byte("fail this"))
	assert.Error(t, err)
	assert.Equal(t, "send error", err.Error())
}

func TestGrpcConn_Close(t *testing.T) {
	ctx := context.Background()
	mockClient := newMockTunnelClient(ctx)
	
	var cancelCalled bool
	cancelFunc := func() {
		mockClient.cancelFunc() 
		cancelCalled = true
	}

	grpcConn := &conn{
		c:          mockClient,
		cancelFunc: cancelFunc,
	}

	err := grpcConn.Close()
	assert.NoError(t, err)
	assert.True(t, cancelCalled, "Context cancelFunc was not called")

	mockClient.closeSendErr = errors.New("close send error")
	err = grpcConn.Close() 
	assert.Error(t, err)
	assert.Equal(t, "close send error", err.Error())
}

func TestGrpcConn_AddrDeadline(t *testing.T) {
	grpcConn := &conn{
		localAddr:  &net.TCPAddr{},
		remoteAddr: &net.TCPAddr{},
	}
	assert.Equal(t, &net.TCPAddr{}, grpcConn.LocalAddr())
	assert.Equal(t, &net.TCPAddr{}, grpcConn.RemoteAddr())

	err := grpcConn.SetDeadline(time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline not supported")

	err = grpcConn.SetReadDeadline(time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline not supported")

	err = grpcConn.SetWriteDeadline(time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline not supported")
}
