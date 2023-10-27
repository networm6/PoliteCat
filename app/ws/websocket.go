package ws

import (
	"context"
	"github.com/networm6/PoliteCat/common/tools"
	"github.com/networm6/PoliteCat/tunnel"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/networm6/PoliteCat/common/cache"
)

const ConnTag = "conn"

/*
Client发出的所有网络请求包都会走tun网卡

在StartClient中，Client被抽象为一个双向流，outputStream是用户的请求，inputStream是请求的结果。
在mapStreamsToWebSocket中，用这两股流与ws交互。
在tunToWs中，不断的从outputStream中读取数据，并检测是否存在连接，如果存在则发送到ws。
在wsToTun中，不断的从ws中读取数据，并发送到inputStream。

*/

// UserApp --> Kernel --> UserApp(TUN) --> ReadFromTun --> tunToWs --> ws
func mapStreamsToWebSocket(config *Config, outputStream <-chan []byte, inputStream chan<- []byte, tunCtx context.Context) {
	go tunToWs(outputStream, tunCtx)
	for tools.ContextOpened(tunCtx) {
		// 为每个ws链接建立新的ctx
		connCtx, connCancel := context.WithCancel(tunCtx)
		conn := connectServer(config)
		if conn == nil {
			connCancel()
			time.Sleep(3 * time.Second)
			continue
		}
		// 设置一个链接的有效时长为24小时
		cache.GetCache().Set(ConnTag, conn, 24*time.Hour)
		go wsToTun(conn, inputStream, connCancel, connCtx)
		// 建立连接后，每3秒发送一次ping，检测是否断开。
		ping(conn, connCtx, connCancel)
		cache.GetCache().Delete(ConnTag)
		_ = conn.Close()
	}
}

// StartClient 启动Client端。
func StartClient(conf *Config, tun *tunnel.Tunnel) {
	mapStreamsToWebSocket(conf, tun.OutputStream, tun.InputStream, *tun.LifeCtx)
}

func ping(conn net.Conn, _ctx context.Context, _cancel context.CancelFunc) {
	for tools.ContextOpened(_ctx) {
		err := wsutil.WriteClientMessage(conn, ws.OpText, []byte("ping"))
		if err != nil {
			break
		}
		time.Sleep(3 * time.Second)
	}
	_cancel()
}

// ConnectServer connects to the server with the given address.
func connectServer(config *Config) net.Conn {
	scheme := "ws"

	u := url.URL{Scheme: scheme, Host: config.ServerAddr, Path: config.WSPath}
	header := make(http.Header)
	header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36")
	if config.Key != "" {
		header.Set("key", config.Key)
	}
	dialer := ws.Dialer{
		Header:  ws.HandshakeHeaderHTTP(header),
		Timeout: time.Duration(config.Timeout) * time.Second,
		NetDial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial(network, config.ServerAddr)
		},
	}
	c, _, _, err := dialer.Dial(context.Background(), u.String())
	if err != nil {
		log.Printf("[client] failed to dial websocket %s %v", u.String(), err)
		return nil
	}
	return c
}
