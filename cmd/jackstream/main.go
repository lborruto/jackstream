package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/lborruto/jackstream/internal/bt"
	"github.com/lborruto/jackstream/internal/server"
)

func envInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return def
}

func main() {
	port := envInt("PORT", 7000)
	httpsPort := envInt("HTTPS_PORT", 7001)

	btClient, err := bt.NewClient()
	if err != nil {
		log.Fatalf("bt.NewClient: %v", err)
	}
	defer btClient.Shutdown()

	srv := server.New(btClient)
	handler := srv.BuildHandler()

	stopCleanup := make(chan struct{})
	go func() {
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stopCleanup:
				return
			case <-t.C:
				btClient.Cleanup(srv.MaxConcurrent())
			}
		}
	}()

	httpSrv := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: handler,
	}
	go func() {
		log.Printf("jackstream listening on :%d", port)
		if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("[http] serve: %v", err)
		}
	}()

	var httpsSrv *http.Server
	if os.Getenv("HTTPS_DISABLED") != "1" && os.Getenv("HTTPS_DISABLED") != "true" {
		cert, kpErr := loadCert()
		if kpErr != nil {
			log.Printf("[https] disabled (%v)", kpErr)
		} else {
			httpsSrv = &http.Server{
				Addr:      ":" + strconv.Itoa(httpsPort),
				Handler:   handler,
				TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
			}
			go func() {
				log.Printf("jackstream https listening on :%d", httpsPort)
				if err := httpsSrv.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
					log.Printf("[https] serve: %v", err)
				}
			}()
		}
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Printf("received signal, shutting down")

	close(stopCleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	if httpsSrv != nil {
		_ = httpsSrv.Shutdown(ctx)
	}
	btClient.Shutdown()
}

func loadCert() (tls.Certificate, error) {
	certPath := os.Getenv("HTTPS_CERT_PATH")
	keyPath := os.Getenv("HTTPS_KEY_PATH")
	if certPath != "" && keyPath != "" {
		return tls.LoadX509KeyPair(certPath, keyPath)
	}
	return tls.X509KeyPair(server.CertPEM, server.KeyPEM)
}
