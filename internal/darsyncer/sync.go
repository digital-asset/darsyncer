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
	"log/slog"
	"time"

	"github.com/digital-asset/darsyncer/internal/stopwatch"
	v30 "github.com/digital-asset/dazl-client/v8/go/api/com/digitalasset/canton/admin/participant/v30"
	"google.golang.org/grpc"
)

func Sync(ctx context.Context, conn grpc.ClientConnInterface, db *PackageDatabase) (err error) {
	packageManagementService := v30.NewPackageServiceClient(conn)

	var response *v30.ListPackagesResponse
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		response, err = packageManagementService.ListPackages(ctx, &v30.ListPackagesRequest{})
		if err != nil {
			return err
		}

		missingPackages := db.getMissingPackageIds(response)
		if len(missingPackages) == 0 {
			break
		}

		slog.Info("detected missing packages",
			slog.Any("preexistingPackageCount", len(response.GetPackageDescriptions())),
			slog.Any("missingPackages", missingPackages))

		hadErrors := false
		packageEntries := db.MinimalEntriesForPackages(missingPackages...)
		for _, entry := range packageEntries {
			var duration time.Duration

			req := &v30.UploadDarRequest{
				Dars: []*v30.UploadDarRequest_UploadDarData{
					{
						Bytes:       entry.DarFile,
						Description: &entry.Name,
					},
				},
				VetAllPackages:     true,
				SynchronizeVetting: true,
			}
			err, duration = stopwatch.Time(func() error {
				_, uploadErr := packageManagementService.UploadDar(ctx, req)
				return uploadErr
			})

			if err != nil {
				slog.Warn("failed to upload (will be retried)", slog.Any("darFile", entry.Name), slog.Any("err", err))
				hadErrors = true
			} else {
				slog.Info("successfully uploaded", slog.Any("darFile", entry.Name), slog.Any("elapsedTime", duration))
			}
		}

		if !hadErrors {
			break
		}

		// if any part of our package upload failed, wait 15 seconds, and try again
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second * 15):
		}
	}

	slog.InfoContext(ctx, "the ledger has all of our packages, as expected", slog.Any("packageCount", len(response.GetPackageDescriptions())))
	return nil
}

func getPackageId(item *v30.PackageDescription, _ int) string {
	return item.PackageId
}
