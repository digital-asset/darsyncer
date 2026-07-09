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
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	v30 "github.com/digital-asset/dazl-client/v8/go/api/com/digitalasset/canton/admin/participant/v30"
	pblf "github.com/digital-asset/dazl-client/v8/go/api/com/digitalasset/daml/lf/archive"
	"github.com/jjeffery/stringset"
	"github.com/samber/lo"
	"google.golang.org/protobuf/proto"
)

type PackageDatabase struct {
	Entries []*Entry
}

type Entry struct {
	Name    string
	DarFile []byte
	Hashes  []string
}

// NewFromFS creates a PackageDatabase from a path to a dar or a directory of dars
func NewFromFS(path string) (*PackageDatabase, error) {
	// get the file info
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// resolve the list of dar files
	darsFiles := []string{path}
	if info.IsDir() {
		darsFiles, err = filepath.Glob(filepath.Join(path, "*.dar"))
		if err != nil {
			return nil, err
		}
	}

	// create the package database
	db := &PackageDatabase{}
	for _, dar := range darsFiles {
		bytes, err := os.ReadFile(dar)
		if err != nil {
			return nil, err
		}
		if err := db.Add(dar, bytes); err != nil {
			return nil, err
		}
	}

	return db, nil
}

func (p *PackageDatabase) Add(name string, darFile []byte) error {
	r := bytes.NewReader(darFile)
	reader, err := zip.NewReader(r, int64(len(darFile)))
	if err != nil {
		return err
	}

	entry := &Entry{Name: name, DarFile: darFile}
	for _, f := range reader.File {
		if strings.HasSuffix(f.Name, ".dalf") {
			var hash string
			hash, err = extractHashFromDalfZip(f)
			if err != nil {
				return err
			}

			entry.Hashes = append(entry.Hashes, hash)
		}
	}

	p.Entries = append(p.Entries, entry)
	return nil
}

func (p *PackageDatabase) AllPackageIds() []string {
	var packageIds stringset.Set
	for _, entry := range p.Entries {
		packageIds = packageIds.Add(entry.Hashes...)
	}
	return packageIds.Values()
}

func (p *PackageDatabase) MinimalEntriesForPackages(packageIds ...string) []*Entry {
	// return ALL the DARs that contain the specified package ID. We could be much smarter about
	// which DARs we upload, since there are some shared DALFs that are more part of the standard
	// library than specific DARs; this will trigger over-uploading, but only until the ledger
	// has all the DARs we need
	return lo.Filter(p.Entries, func(item *Entry, index int) bool {
		return lo.Some(packageIds, item.Hashes)
	})
}

func (p *PackageDatabase) getMissingPackageIds(response *v30.ListPackagesResponse) []string {
	missingPackages, _ := lo.Difference(p.AllPackageIds(), lo.Map(response.GetPackageDescriptions(), getPackageId))
	return missingPackages
}

func extractHashFromDalfZip(f *zip.File) (string, error) {
	fileReader, err := f.Open()
	if err != nil {
		return "", err
	}

	archive, err := parseDalf(fileReader)
	if err != nil {
		return "", err
	}

	return archive.Hash, nil
}

func parseDalf(reader io.Reader) (*pblf.Archive, error) {
	dalfBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var a pblf.Archive
	err = proto.Unmarshal(dalfBytes, &a)
	if err != nil {
		return nil, err
	}

	return &a, nil
}
