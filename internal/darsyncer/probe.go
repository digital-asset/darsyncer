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
	"fmt"
	"log/slog"
	"net/http"
)

func (s *Service) handler(w http.ResponseWriter, r *http.Request) {
	if s.Ready.Load() {
		fmt.Fprint(w, "ok")
		return
	}
	http.Error(w, "not ready", http.StatusInternalServerError)
}

func (s *Service) StartProbe(ctx context.Context) {
	go func() {
		http.HandleFunc("/readyz", s.handler)
		slog.ErrorContext(ctx,
			"http service crashed",
			slog.Any("err", http.ListenAndServe(s.ProbeEndpoint, nil)))
	}()
}
