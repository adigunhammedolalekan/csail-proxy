package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type memoryStore struct {
	store map[string]string
}

func (m *memoryStore) Set(k, v string) error {
	m.store[strings.ToLower(k)] = v
	return nil
}

func (m *memoryStore) Get(k string) (string, error) {
	if val, ok := m.store[strings.ToLower(k)]; ok {
		return val, nil
	}
	return "", errors.New("record not found")
}

func TestHandleSet(t *testing.T) {
	type body struct {
		Name string `json:"name"`
		ServiceUrl string `json:"service_url"`
	}
	b := &body{Name: "Test", ServiceUrl: "http://test.example.com"}
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(b); err != nil {
		t.Fatal(err)
	}
	m := &memoryStore{store: make(map[string]string)}
	h := newProxyServer(m)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/set", buf)

	h.handleSet(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected http code %d; got %d instead", http.StatusOK, w.Code)
	}
	val, err := m.Get(b.Name)
	if err != nil {
		t.Fatal(err)
	}
	if val != b.ServiceUrl {
		t.Fatal()
	}
}

func TestExtractServiceUrl(t *testing.T) {
	cases := map[string]string{
		"sub.example.com": "sub",
		"www.sub.example.com": "sub",
	}
	h := newProxyServer(nil)
	for k, v := range cases {
		req := &http.Request{
			Host:             k,
		}
		r, err := h.extractServiceUrl(req)
		if err != nil {
			t.Fatal(err)
		}
		if r != v {
			t.Fatalf("expected value to be %s; got %s", v, r)
		}
	}
}