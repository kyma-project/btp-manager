package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/fsnotify/fsnotify"
)

func main() {
	certFile := getEnv("TLS_CERT_FILE", "/etc/tls/tls.crt")
	keyFile := getEnv("TLS_KEY_FILE", "/etc/tls/tls.key")
	addr := getEnv("LISTEN_ADDR", ":8443")

	srv := &http.Server{Addr: addr}
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			log.Printf("failed to encode response: %v", err)
		}
	})

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	_ = watcher.Add(certFile)

	getCert := func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
		c, err := tls.LoadX509KeyPair(certFile, keyFile)
		return &c, err
	}
	srv.TLSConfig = &tls.Config{GetCertificate: getCert}

	go func() {
		for range watcher.Events {
			log.Println("cert changed, will reload on next connection")
		}
	}()

	log.Printf("fake-server listening on %s", addr)
	log.Fatal(srv.ListenAndServeTLS("", ""))
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
