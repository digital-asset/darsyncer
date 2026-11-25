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
	"fmt"
	"log/slog"
	"time"

	v30 "github.com/digital-asset/dazl-client/v8/go/api/com/digitalasset/canton/admin/participant/v30"
	topologyv30 "github.com/digital-asset/dazl-client/v8/go/api/com/digitalasset/canton/topology/admin/v30"
	"github.com/samber/lo"
	"google.golang.org/grpc"
)

// syncVets continuously makes sure all our packages are vetted
func (s *Service) syncVets(ctx context.Context, conn *grpc.ClientConn, participantId string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := s.vet(ctx, conn, participantId); err != nil {
			slog.ErrorContext(ctx, "failed to vet", slog.Any("err", err))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.RetryDelay):
		}
	}
}

func (s *Service) vet(ctx context.Context, conn *grpc.ClientConn, participantId string) error {
	vettedResponse, err := topologyv30.NewTopologyManagerReadServiceClient(conn).ListVettedPackages(ctx,
		&topologyv30.ListVettedPackagesRequest{
			BaseQuery: &topologyv30.BaseQuery{
				TimeQuery: &topologyv30.BaseQuery_HeadState{},
			},
			FilterParticipant: participantId,
		})
	if err != nil {
		return fmt.Errorf("failed to list vetted packages: %w", err)
	}

	vetted := make(map[string]struct{})
	for _, result := range vettedResponse.GetResults() {
		for _, p := range result.GetItem().GetPackages() {
			vetted[p.GetPackageId()] = struct{}{}
		}
	}

	unvetted := lo.Filter(s.PackageDb.AllPackageIds(), func(id string, _ int) bool {
		_, ok := vetted[id]
		return !ok
	})
	if len(unvetted) == 0 {
		slog.InfoContext(ctx, "all packages are vetted", slog.Any("vettedPackageCount", len(vetted)))
		return nil
	}
	slog.InfoContext(ctx, "detected unvetted packages", slog.Any("unvettedPackages", unvetted))

	packageService := v30.NewPackageServiceClient(conn)
	var errs []error
	for _, entry := range s.PackageDb.MinimalEntriesForPackages(unvetted...) {
		_, err := packageService.VetDar(ctx, &v30.VetDarRequest{MainPackageId: entry.MainPackageId, Synchronize: true})
		if err != nil {
			slog.ErrorContext(ctx, "failed to vet dar", slog.Any("darFile", entry.Name), slog.Any("mainPackageId", entry.MainPackageId), slog.Any("err", err))
			errs = append(errs, err)
			continue
		}
		slog.InfoContext(ctx, "vetted dar", slog.Any("darFile", entry.Name), slog.Any("mainPackageId", entry.MainPackageId))
	}
	return errors.Join(errs...)
}
