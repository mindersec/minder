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

var (
	blockedRequests metric.Int64Counter
	metricsInit     sync.Once
)

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
	transport = transport.Clone()
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
			return nil, fmt.Errorf("remote address is not a TCP address")
		}
		if NonPublicIP(remote.IP) {
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

// Documented in RFC 6598
// May be used by e.g. Tailscale or Alibaba Cloud as "private"
var cgNatPrefix = net.IPNet{
	IP:   net.IP([]byte{100, 64, 0, 0}),
	Mask: net.CIDRMask(10, 32),
}

// Documented in RFC 6052
// There are many other ways to NAT64, so this is not completely authoritative.
var nat64Prefix = net.IPNet{
	IP:   net.ParseIP("64:ff9b::"),
	Mask: net.CIDRMask(96, 128),
}

// NonPublicIP returns true if the IP is one of the assigned ranges for local /
// non-internet communication.  This is somewhat heuristic, but protects calling
// most services accessible to Minder but not to the public internet.
func NonPublicIP(ip net.IP) bool {
	return !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsPrivate() || cgNatPrefix.Contains(ip) || nat64Prefix.Contains(ip)
}
