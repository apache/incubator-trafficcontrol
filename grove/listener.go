package grove

import (
	"crypto/tls"
	"net"
	"time"
)

type InterceptListener struct {
	realListener net.Listener
	connMap      *ConnMap
}

// InterceptListen creates and returns a net.Listener via net.Listen, which is wrapped with an intercepter, which counts Conn read and write bytes. If you want a `grove.NewCacheHandler` to be able to count in and out bytes per remap rule in the stats interface, it must be served with a listener created via InterceptListen or InterceptListenTLS.
func InterceptListen(network, laddr string) (net.Listener, *ConnMap, error) {
	l, err := net.Listen(network, laddr)
	if err != nil {
		return l, nil, err
	}
	connMap := NewConnMap()
	return &InterceptListener{realListener: l, connMap: connMap}, connMap, nil
}

// InterceptListenTLS is like InterceptListen but for serving HTTPS.
func InterceptListenTLS(net, laddr, certFile, keyFile string) (net.Listener, *ConnMap, error) {
	config := &tls.Config{}
	config.Certificates = make([]tls.Certificate, 1)
	err := error(nil)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, nil, err
	}
	l, err := tls.Listen(net, laddr, config)
	if err != nil {
		return l, nil, err
	}
	connMap := NewConnMap()
	return &InterceptListener{realListener: l, connMap: connMap}, connMap, nil
}

func (l *InterceptListener) Accept() (net.Conn, error) {
	c, err := l.realListener.Accept()
	if err != nil {
		return c, err
	}
	interceptConn := &InterceptConn{realConn: c}
	l.connMap.Push(interceptConn)
	return interceptConn, nil
}

func (l *InterceptListener) Close() error {
	return l.realListener.Close()
}

func (l *InterceptListener) Addr() net.Addr {
	return l.realListener.Addr()
}

type InterceptConn struct {
	realConn     net.Conn
	bytesRead    int
	bytesWritten int
}

func (c *InterceptConn) BytesRead() int {
	return c.bytesRead
}

func (c *InterceptConn) BytesWritten() int {
	return c.bytesWritten
}

func (c *InterceptConn) Read(b []byte) (n int, err error) {
	n, err = c.realConn.Read(b)
	c.bytesRead += n
	return
}
func (c *InterceptConn) Write(b []byte) (n int, err error) {
	n, err = c.realConn.Write(b)
	c.bytesWritten += n
	return
}
func (c *InterceptConn) Close() error {
	return c.realConn.Close()
}
func (c *InterceptConn) LocalAddr() net.Addr {
	return c.realConn.LocalAddr()
}
func (c *InterceptConn) RemoteAddr() net.Addr {
	return c.realConn.RemoteAddr()
}
func (c *InterceptConn) SetDeadline(t time.Time) error {
	return c.realConn.SetDeadline(t)
}
func (c *InterceptConn) SetReadDeadline(t time.Time) error {
	return c.realConn.SetReadDeadline(t)
}
func (c *InterceptConn) SetWriteDeadline(t time.Time) error {
	return c.realConn.SetWriteDeadline(t)
}
