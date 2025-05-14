package web

import (
    "net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type ProxyCache struct {
    mu     sync.Mutex
    cached map[string]*httputil.ReverseProxy
}

func (p *ProxyCache) Get(target string) (*httputil.ReverseProxy, error) {
    p.mu.Lock()
    defer p.mu.Unlock()

    if proxy, ok := p.cached[target]; ok {
        return proxy, nil
    }

    u, err := url.Parse(target)
    if err != nil {
        return nil, err
    }

    proxy := httputil.NewSingleHostReverseProxy(u)
    proxy.Director = func(req *http.Request) {
        // Копируем контекст от оригинального запроса к проксируемому
        originalContext := req.Context()
        req = req.WithContext(originalContext) // это важно, чтобы сохранить контекст

        // Копируем заголовки оригинального запроса
        req.Header = make(http.Header)
        for key, values := range req.Header {
            for _, value := range values {
                req.Header.Add(key, value)
            }
        }

        // Указываем новый URL для проксирования
        req.URL.Scheme = u.Scheme
        req.URL.Host = u.Host
    }
    p.cached[target] = proxy
    return proxy, nil
}

func NewProxyCache() *ProxyCache {
    return &ProxyCache{
        cached: make(map[string]*httputil.ReverseProxy),
    }
}
