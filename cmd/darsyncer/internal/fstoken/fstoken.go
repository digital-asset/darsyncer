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

package fstoken

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"log/slog"
	"os"
	"strings"

	"github.com/digital-asset/darsyncer/internal/logger"
	"github.com/digital-asset/darsyncer/internal/tokenstore"
)

type Store struct {
	Path string
}

var _ tokenstore.TokenStore = (*Store)(nil)

func (s *Store) Token(ctx context.Context) (string, error) {
	log := slog.Default().With(slog.String("path", s.Path))
	ctx = logger.W(ctx, "fstoken")
	log.InfoContext(ctx, "picking up auth token")

	// read token from filesystem
	b, err := os.ReadFile(s.Path)
	if err != nil {
		log.ErrorContext(ctx, "failed to read token file", slog.Any("err", err))
		return "", err
	}

	// fingerprint the token to help with troubleshooting
	sha := sha1.New()
	sha.Write(b)
	log.InfoContext(ctx, "token successfully read from filesystem", slog.String("sha1 fingerprint", hex.EncodeToString(sha.Sum(nil))))

	return strings.TrimSpace(string(b)), nil
}
