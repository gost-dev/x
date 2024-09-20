package forward

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"io"
	"net/http"
	"strings"

	dissector "github.com/go-gost/tls-dissector"
	xio "github.com/go-gost/x/internal/io"
)

func Sniffing(ctx context.Context, rdw io.ReadWriter) (rw io.ReadWriter, host string, protocol string, err error) {
	rw = rdw

	// try to sniff TLS traffic
	var hdr [dissector.RecordHeaderLen]byte
	n, err := io.ReadFull(rw, hdr[:])
	rw = xio.NewReadWriter(io.MultiReader(bytes.NewReader(hdr[:n]), rw), rw)
	tlsVersion := binary.BigEndian.Uint16(hdr[1:3])
	if err == nil &&
		hdr[0] == dissector.Handshake &&
		(tlsVersion >= tls.VersionTLS10 && tlsVersion <= tls.VersionTLS13) {
		rw, host, err = sniffSNI(rw)
		protocol = ProtoTLS
		return
	}

	// try to sniff HTTP traffic
	if isHTTP(string(hdr[:])) {
		buf := new(bytes.Buffer)
		var r *http.Request
		r, err = http.ReadRequest(bufio.NewReader(io.TeeReader(rw, buf)))
		rw = xio.NewReadWriter(io.MultiReader(buf, rw), rw)
		if err == nil {
			host = r.Host
			protocol = ProtoHTTP
			return
		}
	}

	protocol = sniffProtocol(hdr[:])

	return
}

func sniffSNI(rw io.ReadWriter) (io.ReadWriter, string, error) {
	buf := new(bytes.Buffer)
	host, err := getServerName(io.TeeReader(rw, buf))
	rw = xio.NewReadWriter(io.MultiReader(buf, rw), rw)
	return rw, host, err
}

func getServerName(r io.Reader) (host string, err error) {
	clientHello, err := dissector.ParseClientHello(r)
	if err != nil {
		return
	}
	host = clientHello.ServerName
	return
}

func isHTTP(s string) bool {
	return strings.HasPrefix(http.MethodGet, s[:3]) ||
		strings.HasPrefix(http.MethodPost, s[:4]) ||
		strings.HasPrefix(http.MethodPut, s[:3]) ||
		strings.HasPrefix(http.MethodDelete, s) ||
		strings.HasPrefix(http.MethodOptions, s) ||
		strings.HasPrefix(http.MethodPatch, s) ||
		strings.HasPrefix(http.MethodHead, s[:4]) ||
		strings.HasPrefix(http.MethodConnect, s) ||
		strings.HasPrefix(http.MethodTrace, s)
}

const (
	ProtoHTTP  = "http"
	ProtoTLS   = "tls"
	ProtoSSHv2 = "SSH-2"
)

func sniffProtocol(hdr []byte) string {
	if string(hdr) == ProtoSSHv2 {
		return "ssh"
	}
	return ""
}
