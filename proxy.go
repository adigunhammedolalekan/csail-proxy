package proxy

import (
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v7"
	"github.com/mholt/certmagic"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

type Store interface {
	Set(key, value string) error
	Get(key string) (string, error)
}

type redisStore struct {
	client *redis.Client
}

func (r *redisStore) Set(key, value string) error {
	return r.client.Set(strings.ToLower(key), value, 0).Err()
}

func (r *redisStore) Get(key string) (string, error) {
	return r.client.Get(strings.ToLower(key)).Result()
}

func newRedisStore() (Store, error) {
	log.Println("connecting to redis on host: ", os.Getenv("REDIS_HOST"))
	client := redis.NewClient(&redis.Options{
		Addr:               os.Getenv("REDIS_HOST"),
		Password:           os.Getenv("REDIS_PASSWORD"),
		DB:                 0,
	})
	if err := client.Ping().Err(); err != nil {
		return nil, err
	}
	return &redisStore{client: client}, nil
}

type proxyServer struct {
	store Store
}

type reverseProxy struct {
	handler *httputil.ReverseProxy
}

func newReverseProxy(targetUrl string) (*reverseProxy, error) {
	u, err := url.Parse(targetUrl)
	if err != nil {
		return nil, err
	}
	handler := httputil.NewSingleHostReverseProxy(u)
	return &reverseProxy{handler: handler}, nil
}

func newProxyServer(s Store) *proxyServer {
	return &proxyServer{store: s}
}

func (s *proxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestUri := r.RequestURI
	log.Printf("Handling %s", requestUri)
	if requestUri == "/set" {
		s.handleSet(w, r)
		return
	}
	s.proxyHandler(w, r)
}

func (s *proxyServer) handleSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var body struct{
		Name string `json:"name"`
		ServiceUrl string `json:"service_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := s.store.Set(strings.ToLower(body.Name), body.ServiceUrl); err != nil {
		log.Printf("failed to set redis value: Key=%s; Value=%s", body.Name, body.ServiceUrl)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *proxyServer) proxyHandler(w http.ResponseWriter, r *http.Request)  {
	serviceUrl, err := s.extractServiceUrl(r)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	p, err := newReverseProxy(serviceUrl)
	if err != nil {
		log.Printf("bad serviceURL: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	p.handler.ServeHTTP(w, r)
}

func (s *proxyServer) extractServiceUrl(r *http.Request) (string, error) {
	host := r.Host
	parts := strings.Split(host, ".")
	// subdomain.example.com
	if len(parts) > 2 && parts[0] != "www" {
		return parts[0], nil
	}
	// www.subdomain.example.com
	if len(parts) == 4 {
		return parts[1], nil
	}
	return "", errors.New("unknown URL.Host format")
}

func Run(addr string) error {
	serveHttps := os.Getenv("ENV") == "prod"
	s, err := newRedisStore()
	if err != nil {
		return err
	}
	handler := newProxyServer(s)
	if serveHttps {
		log.Println("started https server!")
		certmagic.Default.Agreed = true
		certmagic.Default.Email = "adigunhammed.lekan@gmail.com"
		certmagic.Default.CA = certmagic.LetsEncryptStagingCA
		return certmagic.HTTPS([]string{"hostgolang.com", "www.hostgolang.com"}, handler)
	}else {
		log.Printf("Proxy server running on %s", addr)
		srv := &http.Server{Addr: addr, Handler: handler}
		return srv.ListenAndServe()
	}
}