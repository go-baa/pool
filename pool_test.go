package pool

import (
	"log"
	"net"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// serverAddr  test tcp server address
var serverAddr = "127.0.0.1:8003"

func TestPoolTCP(t *testing.T) {
	var pool *Pool
	var err error
	var n int
	go tcpServer()
	Convey("create connection pool", t, func() {
		pool, err = New(2, 10, func() interface{} {
			addr, _ := net.ResolveTCPAddr("tcp4", serverAddr)
			cli, err := net.DialTCP("tcp4", nil, addr)
			if err != nil {
				log.Fatalf("create client connection error: %v", err)
			}
			return cli
		})
		So(err, ShouldBeNil)
		pool.Close = func(v interface{}) {
			v.(*net.TCPConn).Close()
		}
		So(pool.Len(), ShouldEqual, 2)
	})
	Convey("get connection then put", t, func() {
		v, err := pool.Get()
		So(err, ShouldBeNil)
		cli := v.(*net.TCPConn)
		n, err = cli.Write([]byte("PING"))
		So(err, ShouldBeNil)
		So(n, ShouldEqual, 4)
		re := make([]byte, 4)
		n, err = cli.Read(re)
		So(err, ShouldBeNil)
		So(n, ShouldEqual, 4)
		So(string(re), ShouldEqual, "PONG")
		So(pool.Len(), ShouldEqual, 1)
		pool.Put(cli)
		So(pool.Len(), ShouldEqual, 2)
	})
	Convey("get connection reuse then put", t, func() {
		v, err := pool.Get()
		So(err, ShouldBeNil)
		cli := v.(*net.TCPConn)
		for i := 0; i < 10; i++ {
			n, err = cli.Write([]byte("PING"))
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 4)
			re := make([]byte, 4)
			n, err = cli.Read(re)
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 4)
			So(string(re), ShouldEqual, "PONG")
		}
		So(pool.Len(), ShouldEqual, 1)
		pool.Put(cli)
		So(pool.Len(), ShouldEqual, 2)
	})
	Convey("get many connections", t, func() {
		for i := 0; i < 10; i++ {
			v, err := pool.Get()
			So(err, ShouldBeNil)
			cli := v.(*net.TCPConn)
			n, err = cli.Write([]byte("PING"))
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 4)
			re := make([]byte, 4)
			n, err = cli.Read(re)
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 4)
			So(string(re), ShouldEqual, "PONG")
			pool.Put(cli)
		}
		So(pool.Len(), ShouldEqual, 2)
	})
	Convey("get overlay connections", t, func() {
		conns := make([]interface{}, 20)
		for i := 0; i < 20; i++ {
			v, err := pool.Get()
			So(err, ShouldBeNil)
			cli := v.(*net.TCPConn)
			n, err = cli.Write([]byte("PING"))
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 4)
			re := make([]byte, 4)
			n, err = cli.Read(re)
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 4)
			So(string(re), ShouldEqual, "PONG")
			conns[i] = cli
		}
		for _, cli := range conns {
			pool.Put(cli)
		}
		So(pool.Len(), ShouldEqual, 10)
	})
	Convey("get connection and no back", t, func() {
		v, err := pool.Get()
		So(err, ShouldBeNil)
		cli := v.(*net.TCPConn)
		n, err = cli.Write([]byte("PING"))
		So(err, ShouldBeNil)
		So(n, ShouldEqual, 4)
		re := make([]byte, 4)
		n, err = cli.Read(re)
		So(err, ShouldBeNil)
		So(n, ShouldEqual, 4)
		So(string(re), ShouldEqual, "PONG")
		So(pool.Len(), ShouldEqual, 9)
	})
	Convey("destroy connection pool", t, func() {
		pool.Destroy()
		So(pool.Len(), ShouldEqual, 0)
	})
	Convey("get connection after destroy", t, func() {
		v, err := pool.Get()
		So(err, ShouldBeNil)
		cli := v.(*net.TCPConn)
		n, err = cli.Write([]byte("PING"))
		So(err, ShouldBeNil)
		So(n, ShouldEqual, 4)
		re := make([]byte, 4)
		n, err = cli.Read(re)
		So(err, ShouldBeNil)
		So(n, ShouldEqual, 4)
		So(string(re), ShouldEqual, "PONG")
		So(pool.Len(), ShouldEqual, 0)
		pool.Put(cli)
		So(pool.Len(), ShouldEqual, 0)
	})
}

func tcpServer() error {
	ln, err := net.Listen("tcp4", serverAddr)
	if err != nil {
		log.Fatalf("test server start error: %v", err)
	}
	var connNum int
	for {
		conn, err := ln.Accept()
		connNum++
		log.Printf("\n->accept new connection %v, now has %d connections\n", conn.RemoteAddr(), connNum)
		if err != nil {
			log.Printf("test server accept error: %v", err)
			continue
		}
		go func(conn net.Conn) {
			for {
				re := make([]byte, 4)
				n, err := conn.Read(re)
				if err == nil && n == 4 {
					conn.Write([]byte("PONG"))
				}
			}
		}(conn)
	}
}
