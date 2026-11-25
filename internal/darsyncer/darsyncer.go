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
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	v30 "github.com/digital-asset/dazl-client/v8/go/api/com/digitalasset/canton/admin/participant/v30"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

type Service struct {
	Endpoint                string
	ProbeEndpoint           string
	TokenSource             *oauth2.TokenSource
	PackageDb               *PackageDatabase
	VetDars                 bool
	Ready                   atomic.Bool
	Check                   bool
	Wait                    bool
	RetryDelay              time.Duration
	TransportSecurityPolicy TransportSecurityPolicy
}

func (s *Service) Run(ctx context.Context) error {
	conn, err := s.connect(ctx)
	if err != nil {
		return err
	}
	defer closeConn(ctx, conn)

	if s.Check {
		slog.Info("running in check-only mode")
		return s.check(ctx, conn)
	}

	s.StartProbe(ctx)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// no need to log errors here, because handlers log more detailed errors
		if s.Wait {
			slog.Info("running in wait mode")
			if s.check(ctx, conn) == nil {
				return nil
			}
		} else {
			_ = s.runSync(ctx, conn)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.RetryDelay):
			// wait 15 seconds between retries as to not overly spam the ledger with connection attempts
		}
	}
}

func (s *Service) check(ctx context.Context, conn *grpc.ClientConn) error {
	response, err := v30.NewPackageServiceClient(conn).ListPackages(ctx, &v30.ListPackagesRequest{})
	if err != nil {
		slog.ErrorContext(ctx, "failed to list packages", slog.Any("err", err))
		return err
	}

	missingPackages := s.PackageDb.getMissingPackageIds(response)
	if len(missingPackages) == 0 {
		slog.Info("all dars present on target")
		return nil
	}

	errMsg := "detected missing packages"
	slog.Info(errMsg,
		slog.Any("preexistingPackageCount", len(response.GetPackageDescriptions())),
		slog.Any("missingPackages", missingPackages))
	return errors.New(errMsg)
}

func (s *Service) connect(ctx context.Context) (*grpc.ClientConn, error) {
	if s.TokenSource != nil {
		token, err := (*s.TokenSource).Token()
		if err != nil {
			slog.ErrorContext(ctx, "failed to fetch token", slog.Any("err", err))
			return nil, err
		}

		// add token to context
		md := metadata.Pairs("authorization", "Bearer "+token.AccessToken)
		ctx = metadata.NewOutgoingContext(ctx, md)

		slog.InfoContext(ctx, "token added to context")
	}

	transportCreds, err := getTlsCredentials(ctx, s.Endpoint, s.TransportSecurityPolicy)
	if err != nil {
		slog.ErrorContext(ctx, "failed to connect", slog.Any("err", err))
		return nil, err
	}

	conn, err := grpc.NewClient(s.Endpoint,
		grpc.WithTransportCredentials(transportCreds),
		// we rely on connection liveness to avoid having to spam the ledger repeatedly for package status
		// under the assumption that once a ledger has packages uploaded to it, they will never be removed,
		// BUT if the admin ledger is restarted, there's a chance that it's actually a completely new ledger
		// so packages might need to be uploaded
		grpc.WithKeepaliveParams(keepalive.ClientParameters{PermitWithoutStream: true}))
	if err != nil {
		slog.ErrorContext(ctx, "failed to connect", slog.Any("err", err))
		return nil, err
	}

	conn.Connect()
	return conn, nil
}

func closeConn(ctx context.Context, conn *grpc.ClientConn) {
	slog.InfoContext(ctx, "closing connection")
	if err := conn.Close(); err != nil {
		slog.ErrorContext(ctx, "failed to close connection cleanly", slog.Any("err", err))
	}
}

// runSync checks target Ledger API instances for the specified set of packages,
// and uploads them if they are missing. It also makes sure they're all vetted.
func (s *Service) runSync(ctx context.Context, conn *grpc.ClientConn) error {
	if !s.Ready.Load() { // only upload packages once
		err := Sync(ctx, conn, s.PackageDb, s.VetDars)
		if err != nil {
			slog.ErrorContext(ctx, "failed to sync", slog.Any("err", err))
			return err
		}
	}

	err := MaintainVets(ctx, conn, s.VetDars)
	if err != nil {
		slog.ErrorContext(ctx, "failed to maintain vets", slog.Any("err", err))
		return err
	}

	s.Ready.Store(true)
	return nil
}

func MaintainVets(ctx context.Context, conn *grpc.ClientConn, vet bool) error {
	if !vet {
		return nil
	}
	// TODO
	return nil
}
