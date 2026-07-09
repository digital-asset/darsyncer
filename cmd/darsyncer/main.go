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

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/digital-asset/darsyncer/cmd/darsyncer/internal/tokensources"
	"github.com/digital-asset/darsyncer/internal/darsyncer"
	"github.com/digital-asset/darsyncer/internal/logger"
	"github.com/urfave/cli/v3"
	"golang.org/x/oauth2/clientcredentials"
)

const transportSecurityFlag = "transport-security"

func main() {
	var justPrint bool
	var darsPath,
		tokenPath,
		clientID,
		clientSecret,
		oauthDomain,
		audience,
		scope string

	service := &darsyncer.Service{}

	root := cli.Command{
		Name:  "darsyncer",
		Usage: "Sync dars to a participant",
		MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
			{
				Flags: [][]cli.Flag{
					{
						&cli.BoolFlag{
							Name:        "just-print",
							Usage:       "just print the package ids and exit",
							OnlyOnce:    true,
							Destination: &justPrint,
						},
					},
					{
						&cli.StringFlag{
							Name:        "endpoint",
							Usage:       "participant endpoint to sync to",
							OnlyOnce:    true,
							Destination: &service.Endpoint,
							Sources:     cli.EnvVars("ENDPOINT"),
						},
						&cli.BoolFlag{
							Name:        "check",
							Usage:       "check whether the ledger has all dars and exit",
							OnlyOnce:    true,
							Destination: &service.Check,
						},
						&cli.BoolFlag{
							Name:        "wait",
							Usage:       "wait until the ledger has all dars and exit",
							OnlyOnce:    true,
							Destination: &service.Wait,
						},

						&cli.DurationFlag{
							Name:        "retry-delay",
							Usage:       "interval between retries",
							OnlyOnce:    true,
							Destination: &service.RetryDelay,
							Value:       15 * time.Second,
						},
						&cli.StringFlag{
							Name:        "probe-endpoint",
							Usage:       "address to bind the health probe server to",
							OnlyOnce:    true,
							Destination: &service.ProbeEndpoint,
							Value:       "0.0.0.0:8080",
						},
					},
				},
				Required: true,
				Category: "Mode",
			},
			{
				Flags: [][]cli.Flag{
					// static token flags
					{
						&cli.StringFlag{
							Name:        "token-path",
							Usage:       "path to jwks token file",
							OnlyOnce:    true,
							Destination: &tokenPath,
							Sources:     cli.EnvVars("TOKEN_PATH"),
						},
					},
					// oauth flags
					{
						&cli.StringFlag{
							Name:        "client-id",
							Usage:       "oauth client id",
							OnlyOnce:    true,
							Destination: &clientID,
							Sources:     cli.EnvVars("CLIENT_ID"),
						},
						&cli.StringFlag{
							Name:        "client-secret",
							Usage:       "oauth client secret",
							OnlyOnce:    true,
							Destination: &clientSecret,
							Sources:     cli.EnvVars("CLIENT_SECRET"),
						},
						&cli.StringFlag{
							Name:        "oauth-domain",
							Usage:       "oauth domain",
							OnlyOnce:    true,
							Destination: &oauthDomain,
							Sources:     cli.EnvVars("OAUTH_DOMAIN"),
						},
						&cli.StringFlag{
							Name:        "audience",
							Usage:       "oauth audience",
							OnlyOnce:    true,
							Destination: &audience,
							Sources:     cli.EnvVars("AUDIENCE"),
						},
						&cli.StringFlag{
							Name:        "scope",
							Usage:       "oauth scope",
							OnlyOnce:    true,
							Destination: &scope,
							Sources:     cli.EnvVars("SCOPE"),
						},
					},
				},
				Category: "Auth",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  transportSecurityFlag,
				Value: darsyncer.TLSWithPlaintextFallback.String(),
				Usage: "participant connection security: https-with-plaintext-fallback or https-only",
			},
			&cli.StringFlag{
				Name:        "dars",
				Usage:       "path to a directory of dars or a single dar",
				OnlyOnce:    true,
				Required:    true,
				TakesFile:   true,
				Destination: &darsPath,
				Sources:     cli.EnvVars("DARS"),
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			var err error

			// connection security
			service.TransportSecurityPolicy, err = ParseTransportSecurityPolicy(
				c.String("transport-security"),
			)
			if err != nil {
				return err
			}
			if service.TransportSecurityPolicy == darsyncer.TLSWithPlaintextFallback {
				slog.Warn(fmt.Sprintf("the --%s flag was not set and will default to %s", transportSecurityFlag, darsyncer.TLSWithPlaintextFallback))
			}

			// create the db
			service.PackageDb, err = darsyncer.NewFromFS(darsPath)
			if err != nil {
				return err
			}

			// output the package ids for each dar file
			for _, entry := range service.PackageDb.Entries {
				slog.InfoContext(ctx, "DAR package", slog.Any("name", entry.Name), slog.Any("packageIds", entry.Hashes))
			}

			// quit if we are just printing
			if justPrint {
				return nil
			}

			// pick auth type
			authLog := slog.StringValue("none")
			if tokenPath != "" {
				authLog = slog.GroupValue(
					slog.String("type", "static token"),
					slog.String("path", tokenPath),
				)
				slog.InfoContext(ctx, "creating static token source", slog.Any("auth", authLog))
				if ts, err := tokensources.StaticFromFs(ctx, tokenPath); err != nil {
					slog.ErrorContext(ctx, "could not get static token source", slog.Any("err", err))
					return err
				} else {
					service.TokenSource = ts
				}
			} else if clientID != "" && clientSecret != "" && oauthDomain != "" && audience != "" && scope != "" {
				audiences := strings.Split(audience, " ")
				scopes := strings.Split(scope, " ")
				authLog = slog.GroupValue(
					slog.String("type", "oauth"),
					slog.String("client id", clientID),
					slog.String("domain", oauthDomain),
					slog.String("audience", audience),
					slog.String("scope", scope),
				)

				oidcProvider, err := oidc.NewProvider(ctx, oauthDomain)
				if err != nil {
					slog.ErrorContext(ctx, "could not create OIDC provider", slog.Any("err", err))
					return err
				}

				slog.InfoContext(ctx, "creating client credentials token source", slog.Any("auth", authLog))
				clientCredentialsConfig := clientcredentials.Config{
					ClientID:     clientID,
					ClientSecret: clientSecret,
					TokenURL:     oidcProvider.Endpoint().TokenURL,
					Scopes:       scopes,
					EndpointParams: url.Values{
						"audience": audiences,
					},
				}

				service.TokenSource = new(clientCredentialsConfig.TokenSource(ctx))
			}

			// start the syncer
			slog.InfoContext(
				ctx,
				"starting darsyncer",
				slog.String("dars path", darsPath),
				slog.String("endpoint", service.Endpoint),
				slog.Any("auth", authLog),
			)

			return service.Run(ctx)
		},
	}

	logger.Setup(&root)

	ctx := context.Background()
	if err := root.Run(ctx, os.Args); err != nil {
		slog.ErrorContext(ctx, "fatal error", slog.Any("err", err))
		os.Exit(1)
	}
}

func ParseTransportSecurityPolicy(s string) (darsyncer.TransportSecurityPolicy, error) {
	switch s {
	case darsyncer.TLSWithPlaintextFallback.String():
		return darsyncer.TLSWithPlaintextFallback, nil
	case darsyncer.TLSOnly.String():
		return darsyncer.TLSOnly, nil
	default:
		return 0, fmt.Errorf(
			"invalid transport security policy %q, must be one of: %s, %s",
			s,
			darsyncer.TLSWithPlaintextFallback,
			darsyncer.TLSOnly,
		)
	}
}
