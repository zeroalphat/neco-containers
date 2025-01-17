package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var config struct {
	timeout    time.Duration
	listenAddr string
	readyURL   string
	httpURL    string
	httpsAddr  string
}

type monitor struct {
	client         *http.Client
	timeout        time.Duration
	readyURL       string
	httpURL        string
	httpsAddr      string
	httpActivated  atomicBool
	httpsActivated atomicBool
}

type atomicBool struct {
	flag int32
}

func (b *atomicBool) set(flag bool) {
	var val int32
	if flag {
		val = 1
	}
	atomic.StoreInt32(&b.flag, val)
}

func (b *atomicBool) get() bool {
	return atomic.LoadInt32(&b.flag) != 0
}

var rootCmd = &cobra.Command{
	Use:   "livenessprobe",
	Short: "liveness probe for Envoy",
	Long:  `Liveness probe for Envoy.`,

	Run: func(cmd *cobra.Command, args []string) {
		err := well.LogConfig{}.Apply()
		if err != nil {
			log.ErrorExit(err)
		}

		mux := http.NewServeMux()

		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.DisableKeepAlives = true

		m := &monitor{
			client: &http.Client{
				Transport: transport,
				Timeout:   config.timeout,
			},
			readyURL:  config.readyURL,
			httpURL:   config.httpURL,
			httpsAddr: config.httpsAddr,
			timeout:   config.timeout,
		}
		mux.Handle("/", m)

		serv := &http.Server{
			Addr:    config.listenAddr,
			Handler: mux,
		}
		well.Go(func(ctx context.Context) error {
			<-ctx.Done()
			return serv.Shutdown(ctx)
		})
		err = serv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.ErrorExit(err)
		}
	},
}

// Execute executes the command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (m *monitor) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	env := well.NewEnvironment(req.Context())
	env.Go(m.monitorReady)
	env.Go(m.monitorHTTP)
	env.Go(m.monitorHTTPS)

	env.Stop()
	err := env.Wait()
	if err != nil {
		rw.WriteHeader(http.StatusBadGateway)
		return
	}

	log.Debug("returning success result", nil)
}

func (m *monitor) monitorReady(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", m.readyURL, nil)
	if err != nil {
		log.Error("failed to build HTTP request for readyURL", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		log.Error("failed to access readiness probe", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("readiness probe returned non-OK", map[string]interface{}{
			"status": resp.StatusCode,
		})
		return fmt.Errorf("readiness probe returned %d", resp.StatusCode)
	}

	return nil
}

func (m *monitor) monitorHTTP(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", m.httpURL, nil)
	if err != nil {
		log.Error("failed to build HTTP request for httpURL", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		log.Error("failed to access HTTP endpoint", map[string]interface{}{
			log.FnError: err,
		})
		if !m.httpActivated.get() {
			return nil
		}
		return err
	}
	defer resp.Body.Close()

	// Status code is not checked.
	// The current implementation of Envoy returns 404, but this can be changed.
	m.httpActivated.set(true)
	return nil
}

func (m *monitor) monitorHTTPS(ctx context.Context) error {
	conn, err := net.DialTimeout("tcp", m.httpsAddr, m.timeout)
	if err != nil {
		log.Error("failed to connect to HTTPS endpoint", map[string]interface{}{
			log.FnError: err,
		})
		if !m.httpsActivated.get() {
			return nil
		}
		return err
	}

	if conn != nil {
		conn.Close()
	}
	m.httpsActivated.set(true)
	return nil
}

func init() {
	fs := rootCmd.Flags()
	fs.StringVar(&config.listenAddr, "listen-addr", ":8502", "Listen address for probe")
	fs.DurationVar(&config.timeout, "timeout", time.Second*5, "Timeout")
	fs.StringVar(&config.readyURL, "ready-url", "http://localhost:8002/ready", "URL of Envoy readiness probe")
	fs.StringVar(&config.httpURL, "http-url", "http://localhost:8080/", "URL for checking HTTP behavior")
	fs.StringVar(&config.httpsAddr, "https-addr", "localhost:8443", "Address for checking HTTPS behavior")
}
