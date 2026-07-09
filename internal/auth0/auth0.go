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

package auth0

import (
	"context"
	"log/slog"
	"time"

	"github.com/auth0/go-auth0/authentication"
	"github.com/auth0/go-auth0/authentication/oauth"
	"github.com/digital-asset/darsyncer/internal/logger"
	"github.com/digital-asset/darsyncer/internal/tokenstore"
)

type Service struct {
	domain, clientID, clientSecret string

	get chan *oauth.TokenSet
}

var _ tokenstore.TokenStore = (*Service)(nil)

func New(domain, clientID, clientSecret string) *Service {
	return &Service{
		domain:       domain,
		clientID:     clientID,
		clientSecret: clientSecret,
		get:          make(chan *oauth.TokenSet),
	}
}

func (s *Service) getToken(ctx context.Context) (*oauth.TokenSet, error) {
	ctx = logger.W(ctx, "getToken")

	api, err := authentication.New(
		ctx,
		s.domain,
		authentication.WithClientID(s.clientID),
		authentication.WithClientSecret(s.clientSecret),
	)
	if err != nil {
		slog.WarnContext(ctx, "failed to build client", slog.Any("err", err))
		return nil, err
	}

	token, err := api.OAuth.LoginWithClientCredentials(ctx, oauth.LoginWithClientCredentialsRequest{
		Audience: "https://canton.network.global",
	}, oauth.IDTokenValidationOptions{})

	if err != nil {
		slog.WarnContext(ctx, "failed to login", slog.Any("err", err))
		return nil, err
	}

	slog.InfoContext(ctx, "successfully logged in")
	return token, nil
}

// tokenStream simply pushes fresh tokens down the returned channel.
// Login is retried 20 times, after a backoff of 1 hour is instated to prevent
// hitting any quotas. The token is automatically refreshed apoximently 2/3s
// through its life to ensure there is never a lapse in auth. For example,
// if a token lives for 24 hours, then attempts to refresh it will start at 16
// hours, within the same retry polices mentioned before.
func (s *Service) tokenStream(ctx context.Context) <-chan *oauth.TokenSet {
	ctx = logger.W(ctx, "tokenStreamer")

	r := make(chan *oauth.TokenSet)

	go func() {
		defer close(r)

		// retry every 5 seconds
		retires := 0
		retryTick := time.Second * 5

		// backoff for an hour if after 20 retries
		backoffTick := time.Hour
		backoffAtRetry := 20
		backingOff := false

		// assume retry
		ticker := time.NewTicker(retryTick)
		defer ticker.Stop()

		for {
			token, err := s.getToken(ctx)
			if err == nil {
				retires = 0

				// reset the ticker to 2/3s of the tokens life, then attempt to refresh
				when := time.Second * time.Duration(token.ExpiresIn*2/3)
				ticker.Reset(when)
				slog.InfoContext(ctx, "waiting to refresh token", slog.Time("when", time.Now().Add(when)))

				// send token
				select {
				case r <- token:
				case <-ctx.Done():
					slog.InfoContext(ctx, "context closed", slog.Any("err", ctx.Err()))
					return
				case <-ticker.C:
					slog.WarnContext(ctx, "token expired before it could be received")
				}
			} else {
				retires++
				slog.InfoContext(ctx, "failed to login, retrying", slog.Int("retries", retires))
			}

			// protect against rate limiting
			if retires >= backoffAtRetry {
				slog.WarnContext(ctx, "backing off retries, too many attempts", slog.Time("resuming at", time.Now().Add(backoffTick)))

				backingOff = true
				ticker.Reset(backoffTick)
			}

			// wait for next tick
			select {
			case <-ctx.Done():
				slog.InfoContext(ctx, "context closed", slog.Any("err", ctx.Err()))
				return
			case <-ticker.C:
			}

			// only backoff once
			if backingOff {
				ticker.Reset(retryTick)
			}
		}
	}()

	return r
}

// Start the token service
func (s *Service) Start(ctx context.Context) {
	ctx = logger.W(ctx, "service")

	go func() {
		// tokens with be provided initially once, and then apoximently 2/3s before it expires
		tokens := s.tokenStream(ctx)

		var cache *oauth.TokenSet
		// TODO - fix this ticker logic
		expired := time.NewTicker(time.Hour * 24)
		defer expired.Stop()

		for {

			// Block until cache is filled.
			slog.InfoContext(ctx, "waiting for token")
			select {
			case <-ctx.Done():
				slog.InfoContext(ctx, "context closed", slog.Any("err", ctx.Err()))
				return
			case c, ok := <-tokens:
				if !ok {
					slog.InfoContext(ctx, "token stream closed")
					return
				}
				when := time.Duration(c.ExpiresIn) * time.Second
				slog.InfoContext(ctx, "token received", slog.Time("expires", time.Now().Add(when)))
				cache = c
				expired.Reset(when)
			}

		loop:
			// Continue to serve the cached token until it is expired, then force a cache refresh.
			// This shouldn't happen in normal situations as all token refreshes also refresh
			// the cache here, but if for some reason the token does expire, there is no sense
			// in further hosting it, better to block until it refreshes (if ever).
			for {
				select {
				case <-ctx.Done():
					slog.InfoContext(ctx, "context closed", slog.Any("err", ctx.Err()))
					return

				// serve cached token
				case s.get <- cache:
					slog.InfoContext(ctx, "cached token sent")

				// refresh cache when token refreshes
				case c, ok := <-tokens:
					if !ok {
						slog.InfoContext(ctx, "token stream closed")
						return
					}

					when := time.Duration(c.ExpiresIn) * time.Second
					slog.InfoContext(ctx, "token received", slog.Time("expires", time.Now().Add(when)))
					cache = c
					expired.Reset(when)

				// break out of the loop and force a cache refresh if token expires
				case <-expired.C:
					slog.Warn("token expired before it could refresh")
					break loop
				}
			}
		}
	}()
}

// Token returns the current cached token or returns an error if the context is closed
func (s *Service) Token(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case token := <-s.get:
		return token.AccessToken, nil
	}
}
