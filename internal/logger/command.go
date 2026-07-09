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
	"os"

	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func Setup(cmd *cli.Command) {
	level := &slog.LevelVar{}

	// Append logging flags
	cmd.Flags = append(cmd.Flags,
		&cli.StringFlag{
			Name:  "log",
			Value: "info",
			Usage: "log level",
			Action: func(_ context.Context, _ *cli.Command, value string) error {
				return level.UnmarshalText([]byte(value))
			},
			OnlyOnce: true,
			Validator: func(value string) error {
				var check slog.Level
				return check.UnmarshalText([]byte(value))
			},
		},
	)

	// add our before function
	before := cmd.Before
	cmd.Before = func(ctx context.Context, c *cli.Command) (context.Context, error) {
		var baseHandler slog.Handler
		if term.IsTerminal(int(os.Stdout.Fd())) {
			baseHandler = slog.NewTextHandler(c.Writer, &slog.HandlerOptions{Level: level})
		} else {
			baseHandler = slog.NewJSONHandler(c.Writer, &slog.HandlerOptions{Level: level})
		}

		slog.SetDefault(slog.New(Wrap(baseHandler)))

		if before != nil {
			return before(ctx, c)
		}
		return nil, nil
	}
}
