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

// Copied from https://github.com/digital-asset/dpm/blob/main/pkg/ocipusher/darpusher/package_id.go

package darsyncer

import (
	"archive/zip"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/samber/lo"
)

var (
	mainDalfRegexp   = regexp.MustCompile(`(?m)^Main-Dalf:[\s\S]*?\.dalf`)
	whitespaceRegexp = regexp.MustCompile(`\s+`)
)

// GetMainPackageId unpacks a dar and extracts its main-package-id
func GetMainPackageId(reader *zip.Reader) (string, error) {
	manifest, err := readDar(reader)
	if err != nil {
		return "", err
	}

	hash := extractHash(manifest)
	if hash == "" {
		return "", fmt.Errorf("could not extract Main-Dalf hash from the dar's manifest")
	}
	return hash, nil
}

// readDar extracts the manifest out of a dar
func readDar(reader *zip.Reader) (string, error) {
	for _, f := range reader.File {
		if strings.EqualFold(f.Name, "META-INF/MANIFEST.MF") {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer func() { _ = rc.Close() }()

			data, err := io.ReadAll(rc)
			if err != nil {
				return "", err
			}

			return string(data), nil
		}
	}

	return "", fmt.Errorf("invalid dar: META-INF/MANIFEST.MF not found")
}

func extractHash(manifest string) string {
	raw := mainDalfRegexp.FindString(manifest)
	if raw == "" {
		return ""
	}
	mainDalf := whitespaceRegexp.ReplaceAllString(raw, "")
	hash, _ := lo.Last(strings.Split(mainDalf, "-"))
	return strings.TrimSuffix(hash, ".dalf")
}
