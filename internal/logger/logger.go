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

package logger

import (
	"context"
	"log/slog"
	"runtime/debug"
)

type wrappedHandler struct {
	baseHandler slog.Handler
}

var _ slog.Handler = (*wrappedHandler)(nil)

func Wrap(handler slog.Handler) slog.Handler {
	return &wrappedHandler{baseHandler: handler}
}

func (w *wrappedHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return w.baseHandler.Enabled(ctx, level)
}

func (w *wrappedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return Wrap(w.baseHandler.WithAttrs(attrs))
}

func (w *wrappedHandler) WithGroup(name string) slog.Handler {
	return Wrap(w.baseHandler.WithGroup(name))
}

func (w *wrappedHandler) Handle(ctx context.Context, record slog.Record) error {
	name := Name(ctx)
	if name != "" {
		record.AddAttrs(slog.String("logger", name))
	}

	if record.Level >= slog.LevelError {
		record.AddAttrs(slog.String("stacktrace", string(debug.Stack())))
	}

	return w.baseHandler.Handle(ctx, record)
}
