package frp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// proxyInstance 代表一个本地反向代理
type proxyInstance struct {
	server   *http.Server
	listener net.Listener
	port     int
	target   *url.URL
}

// ProxyManager 管理 http/https 隧道的本地反向代理，用于兜底页面
type ProxyManager struct {
	mu      sync.Mutex
	proxies map[string]*proxyInstance
}

func NewProxyManager() *ProxyManager {
	return &ProxyManager{
		proxies: make(map[string]*proxyInstance),
	}
}

// EnsureProxy 为 HTTP/HTTPS 隧道启动本地代理，返回代理监听的 IP 和端口
func (pm *ProxyManager) EnsureProxy(t *Tunnel, fallbackHTML string) (string, int, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 如果已存在，先关闭旧的
	if old, ok := pm.proxies[t.Name]; ok {
		_ = old.server.Shutdown(context.Background())
		_ = old.listener.Close()
		delete(pm.proxies, t.Name)
	}

	targetURL, err := url.Parse(fmt.Sprintf("http://%s:%d", t.LocalIP, t.LocalPort))
	if err != nil {
		return "", 0, fmt.Errorf("解析目标地址失败: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", 0, fmt.Errorf("监听代理端口失败: %v", err)
	}
	proxyPort := ln.Addr().(*net.TCPAddr).Port

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 自定义错误处理，返回兜底页面
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, e error) {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.WriteHeader(http.StatusServiceUnavailable)
		_, _ = rw.Write([]byte(fallbackHTML))
	}

	srv := &http.Server{
		Handler: proxy,
	}

	inst := &proxyInstance{
		server:   srv,
		listener: ln,
		port:     proxyPort,
		target:   targetURL,
	}
	pm.proxies[t.Name] = inst

	// 异步启动
	go func() {
		_ = srv.Serve(ln)
	}()

	return "127.0.0.1", proxyPort, nil
}

// Remove 关闭并移除代理
func (pm *ProxyManager) Remove(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if p, ok := pm.proxies[name]; ok {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.server.Shutdown(ctx)
		_ = p.listener.Close()
		delete(pm.proxies, name)
	}
}

