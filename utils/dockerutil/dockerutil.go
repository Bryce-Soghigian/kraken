// Copyright (c) 2016-2019 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package dockerutil

import (
	"errors"
	"fmt"
	"io"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/ocischema"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/uber/kraken/core"
)

const (
	_v2ManifestType     = "application/vnd.docker.distribution.manifest.v2+json"
	_v2ManifestListType = "application/vnd.docker.distribution.manifest.list.v2+json"
	_ociManifestType    = "application/vnd.oci.image.manifest.v1+json"
	_ociIndexType       = "application/vnd.oci.image.index.v1+json"
)

func ParseManifest(r io.Reader) (distribution.Manifest, core.Digest, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, core.Digest{}, fmt.Errorf("read: %s", err)
	}

	// Try Docker v2 manifest first
	manifest, d, err := ParseManifestV2(b)
	if err == nil {
		return manifest, d, err
	}

	// Try OCI manifest
	manifest, d, err = ParseOCIManifest(b)
	if err == nil {
		return manifest, d, err
	}

	// Try Docker v2 manifest list
	manifest, d, err = ParseManifestV2List(b)
	if err == nil {
		return manifest, d, err
	}

	// Try OCI index
	return ParseOCIIndex(b)
}

// ParseManifestV2 returns a parsed v2 manifest and its digest.
func ParseManifestV2(bytes []byte) (distribution.Manifest, core.Digest, error) {
	manifest, desc, err := distribution.UnmarshalManifest(schema2.MediaTypeManifest, bytes)
	if err != nil {
		return nil, core.Digest{}, fmt.Errorf("unmarshal manifest: %s", err)
	}
	deserializedManifest, ok := manifest.(*schema2.DeserializedManifest)
	if !ok {
		return nil, core.Digest{}, errors.New("expected schema2.DeserializedManifest")
	}
	version := deserializedManifest.Manifest.Versioned.SchemaVersion
	if version != 2 {
		return nil, core.Digest{}, fmt.Errorf("unsupported manifest version: %d", version)
	}
	d, err := core.ParseSHA256Digest(string(desc.Digest))
	if err != nil {
		return nil, core.Digest{}, fmt.Errorf("parse digest: %s", err)
	}
	return manifest, d, nil
}

// ParseManifestV2List returns a parsed v2 manifest list and its digest.
func ParseManifestV2List(bytes []byte) (distribution.Manifest, core.Digest, error) {
	manifestList, desc, err := distribution.UnmarshalManifest(manifestlist.MediaTypeManifestList, bytes)
	if err != nil {
		return nil, core.Digest{}, fmt.Errorf("unmarshal manifestlist: %s", err)
	}
	deserializedManifestList, ok := manifestList.(*manifestlist.DeserializedManifestList)
	if !ok {
		return nil, core.Digest{}, errors.New("expected manifestlist.DeserializedManifestList")
	}
	version := deserializedManifestList.ManifestList.Versioned.SchemaVersion
	if version != 2 {
		return nil, core.Digest{}, fmt.Errorf("unsupported manifest list version: %d", version)
	}
	d, err := core.ParseSHA256Digest(string(desc.Digest))
	if err != nil {
		return nil, core.Digest{}, fmt.Errorf("parse digest: %s", err)
	}
	return manifestList, d, nil
}

// ParseOCIManifest returns a parsed OCI manifest and its digest.
func ParseOCIManifest(bytes []byte) (distribution.Manifest, core.Digest, error) {
	manifest, desc, err := distribution.UnmarshalManifest(_ociManifestType, bytes)
	if err != nil {
		return nil, core.Digest{}, fmt.Errorf("unmarshal oci manifest: %s", err)
	}
	deserializedManifest, ok := manifest.(*ocischema.DeserializedManifest)
	if !ok {
		return nil, core.Digest{}, errors.New("expected ocischema.DeserializedManifest")
	}
	version := deserializedManifest.Manifest.Versioned.SchemaVersion
	if version != 2 {
		return nil, core.Digest{}, fmt.Errorf("unsupported oci manifest version: %d", version)
	}
	d, err := core.ParseSHA256Digest(string(desc.Digest))
	if err != nil {
		return nil, core.Digest{}, fmt.Errorf("parse digest: %s", err)
	}
	return manifest, d, nil
}

// ParseOCIIndex returns a parsed OCI index and its digest.
func ParseOCIIndex(bytes []byte) (distribution.Manifest, core.Digest, error) {
	// Try using manifestlist for OCI index since they have similar structure
	index, desc, err := distribution.UnmarshalManifest(_ociIndexType, bytes)
	if err != nil {
		return nil, core.Digest{}, fmt.Errorf("unmarshal oci index: %s", err)
	}
	// OCI index should implement the same interface as manifest list
	d, err := core.ParseSHA256Digest(string(desc.Digest))
	if err != nil {
		return nil, core.Digest{}, fmt.Errorf("parse digest: %s", err)
	}
	return index, d, nil
}

// GetManifestReferences returns a list of references by a V2 or OCI manifest
func GetManifestReferences(manifest distribution.Manifest) ([]core.Digest, error) {
	var refs []core.Digest
	for _, desc := range manifest.References() {
		d, err := core.ParseSHA256Digest(string(desc.Digest))
		if err != nil {
			return nil, fmt.Errorf("parse digest: %w", err)
		}
		refs = append(refs, d)
	}
	return refs, nil
}

func GetSupportedManifestTypes() string {
	return fmt.Sprintf("%s,%s,%s,%s", _v2ManifestType, _v2ManifestListType, _ociManifestType, _ociIndexType)
}
