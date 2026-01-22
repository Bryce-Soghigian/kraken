package dockerutil_test

import (
	"bytes"
	"testing"

	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/stretchr/testify/require"
	"github.com/uber/kraken/utils/dockerutil"
)

var testManifestListBytes = []byte(`{
	"schemaVersion": 2,
	"mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
	"manifests": [
	   {
		  "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
		  "size": 985,
		  "digest": "sha256:1a9ec845ee94c202b2d5da74a24f0ed2058318bfa9879fa541efaecba272e86b",
		  "platform": {
			 "architecture": "amd64",
			 "os": "linux",
			 "features": [
				"sse4"
			 ]
		  }
	   },
	   {
		  "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
		  "size": 2392,
		  "digest": "sha256:6346340964309634683409684360934680934608934608934608934068934608",
		  "platform": {
			 "architecture": "sun4m",
			 "os": "sunos"
		  }
	   }
	]
 }`)

var testManifestBytes = []byte(`{
	"schemaVersion": 2,
	"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
	"config": {
	   "mediaType": "application/vnd.docker.container.image.v1+json",
	   "size": 985,
	   "digest": "sha256:1a9ec845ee94c202b2d5da74a24f0ed2058318bfa9879fa541efaecba272e86b"
	},
	"layers": [
	   {
		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
		  "size": 153263,
		  "digest": "sha256:62d8908bee94c202b2d35224a221aaa2058318bfa9879fa541efaecba272331b"
	   }
	]
 }`)

var testOciManifestBytes = []byte(`{
	"schemaVersion": 2,
	"mediaType": "application/vnd.oci.image.manifest.v1+json",
	"config": {
	   "mediaType": "application/vnd.oci.image.config.v1+json",
	   "size": 985,
	   "digest": "sha256:1a9ec845ee94c202b2d5da74a24f0ed2058318bfa9879fa541efaecba272e86b"
	},
	"layers": [
	   {
		  "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
		  "size": 153263,
		  "digest": "sha256:62d8908bee94c202b2d35224a221aaa2058318bfa9879fa541efaecba272331b"
	   }
	]
 }`)

var testOciIndexBytes = []byte(`{
	"schemaVersion": 2,
	"mediaType": "application/vnd.oci.image.index.v1+json",
	"manifests": [
	   {
		  "mediaType": "application/vnd.oci.image.manifest.v1+json",
		  "size": 985,
		  "digest": "sha256:1a9ec845ee94c202b2d5da74a24f0ed2058318bfa9879fa541efaecba272e86b",
		  "platform": {
			 "architecture": "amd64",
			 "os": "linux"
		  }
	   },
	   {
		  "mediaType": "application/vnd.oci.image.manifest.v1+json",
		  "size": 2392,
		  "digest": "sha256:6346340964309634683409684360934680934608934608934608934068934608",
		  "platform": {
			 "architecture": "arm64",
			 "os": "linux"
		  }
	   }
	]
 }`)

func TestParseManifestV2List(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name          string
		hasError      bool
		manifestBytes []byte
	}{
		{
			name:          "success",
			hasError:      false,
			manifestBytes: testManifestListBytes,
		},
		{
			name:          "wrong manifest type",
			hasError:      true,
			manifestBytes: testManifestBytes,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, d, err := dockerutil.ParseManifestV2List(tt.manifestBytes)
			if tt.hasError {
				require.Error(err)
				return
			}

			require.NoError(err)
			mediaType, _, err := manifest.Payload()
			require.NoError(err)
			require.EqualValues(manifestlist.MediaTypeManifestList, mediaType)
			require.Equal("sha256", d.Algo())
		})
	}
}

func TestParseOCIManifest(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name          string
		hasError      bool
		manifestBytes []byte
	}{
		{
			name:          "success",
			hasError:      false,
			manifestBytes: testOciManifestBytes,
		},
		{
			name:          "wrong manifest type",
			hasError:      true,
			manifestBytes: testManifestListBytes,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, d, err := dockerutil.ParseOCIManifest(tt.manifestBytes)
			if tt.hasError {
				require.Error(err)
				return
			}

			require.NoError(err)
			mediaType, _, err := manifest.Payload()
			require.NoError(err)
			require.EqualValues("application/vnd.oci.image.manifest.v1+json", mediaType)
			require.Equal("sha256", d.Algo())
		})
	}
}

func TestParseOCIIndex(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name          string
		hasError      bool
		manifestBytes []byte
	}{
		{
			name:          "success",
			hasError:      false,
			manifestBytes: testOciIndexBytes,
		},
		{
			name:          "wrong manifest type",
			hasError:      true,
			manifestBytes: testManifestBytes,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, d, err := dockerutil.ParseOCIIndex(tt.manifestBytes)
			if tt.hasError {
				require.Error(err)
				return
			}

			require.NoError(err)
			mediaType, _, err := manifest.Payload()
			require.NoError(err)
			require.EqualValues("application/vnd.oci.image.index.v1+json", mediaType)
			require.Equal("sha256", d.Algo())
		})
	}
}

func TestParseManifest(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name          string
		hasError      bool
		manifestBytes []byte
		expectedMedia string
	}{
		{
			name:          "docker v2 manifest",
			hasError:      false,
			manifestBytes: testManifestBytes,
			expectedMedia: "application/vnd.docker.distribution.manifest.v2+json",
		},
		{
			name:          "docker v2 manifest list",
			hasError:      false,
			manifestBytes: testManifestListBytes,
			expectedMedia: "application/vnd.docker.distribution.manifest.list.v2+json",
		},
		{
			name:          "oci manifest",
			hasError:      false,
			manifestBytes: testOciManifestBytes,
			expectedMedia: "application/vnd.oci.image.manifest.v1+json",
		},
		{
			name:          "oci index",
			hasError:      false,
			manifestBytes: testOciIndexBytes,
			expectedMedia: "application/vnd.oci.image.index.v1+json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, d, err := dockerutil.ParseManifest(bytes.NewReader(tt.manifestBytes))
			if tt.hasError {
				require.Error(err)
				return
			}

			require.NoError(err)
			mediaType, _, err := manifest.Payload()
			require.NoError(err)
			require.EqualValues(tt.expectedMedia, mediaType)
			require.Equal("sha256", d.Algo())
		})
	}
}

func TestGetSupportedManifestTypes(t *testing.T) {
	require := require.New(t)

	supportedTypes := dockerutil.GetSupportedManifestTypes()

	// Should include all four media types
	require.Contains(supportedTypes, "application/vnd.docker.distribution.manifest.v2+json")
	require.Contains(supportedTypes, "application/vnd.docker.distribution.manifest.list.v2+json")
	require.Contains(supportedTypes, "application/vnd.oci.image.manifest.v1+json")
	require.Contains(supportedTypes, "application/vnd.oci.image.index.v1+json")
}
