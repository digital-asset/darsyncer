// Copyright 2026 Copyright (c) 2026 Digital Asset (Switzerland) GmbH and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package darsyncer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type TransportSecurityPolicy int

const (
	TLSWithPlaintextFallback TransportSecurityPolicy = iota
	TLSOnly
)

func (p TransportSecurityPolicy) String() string {
	switch p {
	case TLSWithPlaintextFallback:
		return "https-with-plaintext-fallback"
	case TLSOnly:
		return "https-only"
	default:
		return fmt.Sprintf("unknown")
	}
}

func getTlsCredentials(ctx context.Context, endpoint string, policy TransportSecurityPolicy) (credentials.TransportCredentials, error) {
	tlsConfig, err := tlsConfigForEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	if policy == TLSOnly {
		return credentials.NewTLS(tlsConfig), nil
	}

	var d net.Dialer
	rawConn, err := d.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rawConn.Close() }()

	tlsConn := tls.Client(rawConn, tlsConfig)
	defer func() { _ = tlsConn.Close() }()

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		if isCertError(err) {
			return nil, fmt.Errorf("server supports TLS but certificate verification failed: %w", err)
		}

		slog.Info("falling back to plaintext as we encountered an error while dialing with tls", "err", err)
		return insecure.NewCredentials(), nil
	}

	slog.Info("using tls for the grpc connection")
	return credentials.NewTLS(tlsConfig), nil
}

func isCertError(err error) bool {
	_, ok := errors.AsType[*tls.CertificateVerificationError](err)
	return ok
}

func tlsConfigForEndpoint(endpoint string) (*tls.Config, error) {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return nil, fmt.Errorf("endpoint must be host:port, got %q: %w", endpoint, err)
	}

	return &tls.Config{
		ServerName: host,
	}, nil
}
