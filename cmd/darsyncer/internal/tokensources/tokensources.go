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

package tokensources

import (
	"context"
	"log/slog"

	"github.com/digital-asset/darsyncer/cmd/darsyncer/internal/fstoken"
	"golang.org/x/oauth2"
)

func getTokenFromFs(ctx context.Context, tokenPath string) (string, error) {
	fsStore := &fstoken.Store{
		Path: tokenPath,
	}
	return fsStore.Token(ctx)
}

func StaticFromFs(ctx context.Context, tokenPath string) (*oauth2.TokenSource, error) {
	token, err := getTokenFromFs(ctx, tokenPath)
	if err != nil {
		slog.ErrorContext(ctx, "could not get token from FS", slog.Any("err", err))
		return nil, err
	}

	staticTokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	return &staticTokenSource, nil
}
