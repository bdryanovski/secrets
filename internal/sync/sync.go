package sync

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"time"

	"github.com/bdryanovski/secrets/internal/models"
)

// SyncPayload is the data structure exchanged between machines during sync.
type SyncPayload struct {
	SourceFingerprint string              `json:"source_fingerprint"`
	TargetFingerprint string              `json:"target_fingerprint"`
	Credentials       []models.Credential `json:"credentials"`
	EnvSecrets        []models.EnvSecret  `json:"env_secrets"`
	Timestamp         time.Time           `json:"timestamp"`
}

// SyncServer listens for incoming sync connections.
type SyncServer struct {
	listener  net.Listener
	handler   func(payload *SyncPayload) error
	tlsConfig *tls.Config
}

// NewSyncServer creates a new sync server that listens on the given port.
func NewSyncServer(port int, handler func(payload *SyncPayload) error) (*SyncServer, error) {
	tlsConfig, err := generateSelfSignedTLS()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TLS config: %w", err)
	}

	listener, err := tls.Listen("tcp", fmt.Sprintf(":%d", port), tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to start listener: %w", err)
	}

	return &SyncServer{
		listener:  listener,
		handler:   handler,
		tlsConfig: tlsConfig,
	}, nil
}

// Addr returns the address the server is listening on.
func (s *SyncServer) Addr() string {
	return s.listener.Addr().String()
}

// Serve starts accepting connections. This blocks until the listener is closed.
func (s *SyncServer) Serve() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}

// Close stops the server.
func (s *SyncServer) Close() error {
	return s.listener.Close()
}

func (s *SyncServer) handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	data, err := io.ReadAll(io.LimitReader(conn, 100*1024*1024)) // 100MB limit
	if err != nil {
		fmt.Fprintf(conn, `{"error": "failed to read data: %s"}`, err)
		return
	}

	var payload SyncPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		fmt.Fprintf(conn, `{"error": "invalid payload: %s"}`, err)
		return
	}

	if err := s.handler(&payload); err != nil {
		fmt.Fprintf(conn, `{"error": "sync failed: %s"}`, err)
		return
	}

	fmt.Fprint(conn, `{"status": "ok"}`)
}

// SendSync sends a sync payload to the target machine.
func SendSync(address string, payload *SyncPayload) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Self-signed certs during sync.
	}

	conn, err := tls.Dial("tcp", address, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}

	// Read response.
	resp, err := io.ReadAll(io.LimitReader(conn, 4096))
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("invalid response: %s", string(resp))
	}

	if result.Error != "" {
		return fmt.Errorf("remote error: %s", result.Error)
	}

	return nil
}

// generateSelfSignedTLS creates a self-signed TLS certificate for sync.
func generateSelfSignedTLS() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "secrets-sync"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}
