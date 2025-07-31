// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package smtp

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mhale/smtpd"
	"github.com/stretchr/testify/require"

	util "github.com/mindersec/minder/internal/util/rand"
	"github.com/mindersec/minder/pkg/config/server"
)

func TestNew(t *testing.T) {
	t.Parallel()

	// Create a temporary password file
	passwordFile := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(passwordFile, []byte("testpassword"), 0o600); err != nil {
		t.Errorf("Failed to write password file %s: %s", passwordFile, err)
	}

	tests := []struct {
		name        string
		config      server.SMTP
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid configuration",
			config: server.SMTP{
				Sender:       "sender@example.com",
				Host:         "smtp.example.com",
				Port:         587,
				Username:     "user@example.com",
				PasswordFile: passwordFile,
			},
			expectError: false,
		},
		{
			name: "Valid configuration without auth",
			config: server.SMTP{
				Sender: "sender@example.com",
				Host:   "smtp.example.com",
				Port:   25,
			},
			expectError: false,
		},
		{
			name: "Empty sender",
			config: server.SMTP{
				Sender:       "",
				Host:         "smtp.example.com",
				Port:         587,
				Username:     "user@example.com",
				PasswordFile: passwordFile,
			},
			expectError: true,
			errorMsg:    "sender email address cannot be empty",
		},
		{
			name: "Empty host",
			config: server.SMTP{
				Sender:       "sender@example.com",
				Host:         "",
				Port:         587,
				Username:     "user@example.com",
				PasswordFile: passwordFile,
			},
			expectError: true,
			errorMsg:    "SMTP host cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := New(tt.config)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
				require.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				require.Equal(t, tt.config, client.config)
			}
		})
	}
}

func TestSMTP_sendEmail(t *testing.T) {
	t.Parallel()

	testPassword := "testpassword"
	passwordFile := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(passwordFile, []byte(testPassword), 0o600); err != nil {
		t.Fatalf("Failed to write password file %s: %s", passwordFile, err)
	}

	tests := []struct {
		name          string
		config        server.SMTP
		to            string
		subject       string
		bodyHTML      string
		bodyText      string
		expectError   bool
		errorContains string
	}{
		{
			name: "Successful email send without auth",
			config: server.SMTP{
				Sender: "sender@example.com",
			},
			to:       "recipient@example.com",
			subject:  "Test Subject",
			bodyHTML: "<p>Test HTML Body</p>",
			bodyText: "Test Text Body",
		},
		{
			name: "Successful email send with auth credentials",
			config: server.SMTP{
				Sender:       "sender@example.com",
				Username:     "testuser",
				PasswordFile: passwordFile,
			},
			to:       "recipient@example.com",
			subject:  "Test Subject with Auth",
			bodyHTML: "<h1>Test HTML Body with Auth</h1>",
			bodyText: "Test Text Body with Auth",
		},
		{
			name: "Invalid recipient email",
			config: server.SMTP{
				Sender: "sender@example.com",
			},
			to:            "invalid-email",
			subject:       "Test Subject",
			bodyHTML:      "<p>Test HTML Body</p>",
			bodyText:      "Test Text Body",
			expectError:   true,
			errorContains: "failed to set recipient",
		},
		{
			name: "Missing password file",
			config: server.SMTP{
				Sender:       "sender@example.com",
				Username:     "testuser",
				PasswordFile: "/nonexistent/password/file",
			},
			to:            "recipient@example.com",
			subject:       "Test Subject",
			bodyHTML:      "<p>Test HTML Body</p>",
			bodyText:      "Test Text Body",
			expectError:   true,
			errorContains: "failed to read SMTP password file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := struct {
				From string
				To   []string
				Data []byte
			}{}

			port, err := util.GetRandomPort()
			require.NoError(t, err)
			cfg := tt.config
			cfg.Host = "localhost"
			cfg.Port = int(port)
			serverTLS, clientTLS := makeTLSConfig(t)

			server := smtpd.Server{
				Addr: fmt.Sprintf("127.0.0.1:%d", port),
				Handler: func(_ net.Addr, from string, to []string, data []byte) error {
					msg.From = from
					msg.To = to
					msg.Data = data
					return nil
				},
				AuthRequired: true,
				TLSConfig:    serverTLS,
			}
			if cfg.Username != "" {
				server.AuthHandler = func(_ net.Addr, _ string, username, password, _ []byte) (bool, error) {
					if string(username) == cfg.Username && string(password) == testPassword {
						return true, nil
					}
					return false, fmt.Errorf("invalid credentials")
				}
			}
			go server.ListenAndServe()
			t.Cleanup(func() { _ = server.Close() })
			time.Sleep(5 * time.Millisecond) // Give server time to start

			client, err := New(cfg)
			require.NoError(t, err)
			client.tlsConfig = clientTLS

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err = client.sendEmail(ctx, tt.to, tt.subject, tt.bodyHTML, tt.bodyText)
			require.NoError(t, server.Close())

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
				}
			} else {
				require.NoError(t, err)

				require.Equal(t, tt.config.Sender, msg.From)
				require.Equal(t, []string{tt.to}, msg.To)

				require.Contains(t, string(msg.Data), tt.subject)
				require.Contains(t, string(msg.Data), tt.bodyText)
				require.Contains(t, string(msg.Data), tt.bodyHTML)
			}
		})
	}
}

// returns server and client TLS config for testing
func makeTLSConfig(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()
	serverTLS := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "failed to generate ECDSA key")

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err, "failed to generate serial number")
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &private.PublicKey, private)
	require.NoError(t, err, "failed to create certificate")

	var certBuf, keyBuf bytes.Buffer

	require.NoError(t, pem.Encode(&certBuf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	}))
	keyBytes, err := x509.MarshalPKCS8PrivateKey(private)
	require.NoError(t, err, "failed to marshal private key")
	require.NoError(t, pem.Encode(&keyBuf, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	}))

	cert, err := tls.X509KeyPair(certBuf.Bytes(), keyBuf.Bytes())
	require.NoError(t, err, "failed to load TLS key pair")
	serverTLS.Certificates = []tls.Certificate{cert}

	certpool := x509.NewCertPool()
	certpool.AddCert(cert.Leaf)
	clientTLS := &tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    certpool,
		ServerName: "localhost",
	}

	return serverTLS, clientTLS
}

func TestSMTP_sendEmail_ConnectionFailure(t *testing.T) {
	t.Parallel()

	// Test connection failure to non-existent server
	config := server.SMTP{
		Sender: "sender@example.com",
		Host:   "nonexistent.example.com",
		Port:   587,
	}

	client, err := New(config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.sendEmail(ctx, "recipient@example.com", "Test Subject", "<p>Test HTML</p>", "Test Text")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send email")
}
