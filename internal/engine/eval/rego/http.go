// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package rego provides the rego rule evaluator
package rego

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var blockedRequests metric.Int64Counter
var metricsInit sync.Once

type dialContextFunc = func(ctx context.Context, network, addr string) (net.Conn, error)

// LimitedDialer is an HTTP Dialer (Rego topdowmn.CustomizeRoundTripper) which
// allows us to limit the destination of dialed requests to block specific
// network ranges (such as RFC1918 space).  It operates by attempting to dial
// the requested URL (going through DNS resolution, etc), and then examining
// the remote IP address via conn.RemoteAddr().
func LimitedDialer(transport *http.Transport) http.RoundTripper {
	metricsInit.Do(func() {
		meter := otel.Meter("minder")
		var err error
		blockedRequests, err = meter.Int64Counter(
			"rego.http.blocked_requests",
			metric.WithDescription("Number of Rego requests to private addresses blocked during evaluation"),
		)
		if err != nil {
			zerolog.Ctx(context.Background()).Warn().Err(err).Msg("Creating counter for blocked requests failed")
		}
	})
	if transport == nil {
		var ok bool
		transport, ok = http.DefaultTransport.(*http.Transport)
		if !ok {
			transport = &http.Transport{}
		}
	}
	transport.DialContext = publicOnlyDialer(transport.DialContext)
	return transport
}

func publicOnlyDialer(baseDialer dialContextFunc) dialContextFunc {
	if baseDialer == nil {
		baseDialer = (&net.Dialer{}).DialContext
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := baseDialer(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		remote, ok := conn.RemoteAddr().(*net.TCPAddr)
		if !ok {
			return nil, fmt.Errorf("Remote address is not a TCP address")
		}
		if !remote.IP.IsGlobalUnicast() || remote.IP.IsLoopback() || remote.IP.IsPrivate() {
			// We do not need to lock because blockedRequests is initialized in a sync.Once
			// which is called before this method
			if blockedRequests != nil {
				blockedRequests.Add(ctx, 1)
			}
			// Intentionally do not leak address resolution information
			return nil, fmt.Errorf("remote address is not public")
		}
		return conn, err
	}
}
