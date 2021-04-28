package smtp

import (
	"bytes"
	"net"
	"time"
)

type testConn struct {
	remote net.Addr
	local  net.Addr

	reader *bytes.Buffer
	writer *bytes.Buffer

	readCalls  int
	writeCalls int
	closeCalls int

	onRead  func(conn *testConn, bytes []byte) (int, error)
	onWrite func(conn *testConn, bytes []byte) (int, error)
	onClose func(conn *testConn) error
}

func (conn *testConn) Read(bytes []byte) (int, error) {
	conn.readCalls += 1

	if nil != conn.onRead {
		return conn.onRead(conn, bytes)
	}

	return conn.reader.Read(bytes)
}

func (conn *testConn) SetReadDeadline(deadline time.Time) error {
	return nil
}

func (conn *testConn) Write(bytes []byte) (int, error) {
	conn.writeCalls += 1

	if nil != conn.onWrite {
		return conn.onWrite(conn, bytes)
	}

	return conn.writer.Write(bytes)
}

func (conn *testConn) SetWriteDeadline(deadline time.Time) error {
	return nil
}

func (conn *testConn) SetDeadline(deadline time.Time) error {
	return nil
}

func (conn *testConn) Close() error {
	conn.closeCalls += 1

	if nil != conn.onClose {
		return conn.onClose(conn)
	}

	return nil
}

func (conn *testConn) RemoteAddr() net.Addr {
	return conn.remote
}

func (conn *testConn) LocalAddr() net.Addr {
	return conn.local
}

type testAddr struct {
	network string
	address string
}

func (addr *testAddr) Network() string {
	return addr.network
}

func (addr *testAddr) String() string {
	return addr.address
}
