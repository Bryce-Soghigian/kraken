# OCI v1 Distribution Specification Support

Kraken supports the [OCI (Open Container Initiative) Distribution Specification v1](https://github.com/opencontainers/distribution-spec) alongside the traditional Docker Registry v2 format. This enables Kraken to serve as a registry for OCI artifacts beyond just Docker images, including:

- Helm charts
- Kubernetes manifests
- Kustomize specifications
- WASM modules
- Sigstore signatures
- SBOMs (Software Bill of Materials)
- Any other OCI-compliant artifacts

## Features

- **Dual-format support**: Kraken seamlessly handles both Docker Registry v2 and OCI v1 formats without configuration changes
- **OCI manifest support**: Full support for OCI Image Manifests (`application/vnd.oci.image.manifest.v1+json`) and OCI Image Indexes (`application/vnd.oci.image.index.v1+json`)
- **Backward compatible**: Existing Docker Registry v2 clients continue to work without modification
- **Path flexibility**: Supports both full paths and relative paths for maximum compatibility
- **Standard-compliant**: Implements the OCI Distribution Specification for maximum interoperability

## Architecture

### Supported Media Types

Kraken accepts and serves the following manifest media types:

- `application/vnd.docker.distribution.manifest.v2+json` - Docker Image Manifest v2
- `application/vnd.docker.distribution.manifest.list.v2+json` - Docker Manifest List v2
- `application/vnd.oci.image.manifest.v1+json` - OCI Image Manifest v1
- `application/vnd.oci.image.index.v1+json` - OCI Image Index v1

### Storage Layout

Kraken maintains two parallel storage hierarchies to support both Docker and OCI formats:

#### Docker Registry v2 Layout
```
<root>/docker/registry/v2/
  repositories/
    <name>/
      _manifests/
        revisions/
          sha256/<digest>/link
        tags/<tag>/
          current/link
          index/sha256/<digest>/link
      _layers/
        sha256/<digest>/link
      _uploads/<uuid>/
        data
        startedat
        hashstates/<algorithm>/<offset>
  blobs/
    sha256/<aa>/<digest>/data
```

#### OCI v1 Layout
```
<root>/oci/v1/
  repositories/
    <name>/
      _manifests/
        revisions/
          sha256/<digest>/link
        tags/<tag>/
          current/link
          index/sha256/<digest>/link
      _layers/
        sha256/<digest>/link
      _uploads/<uuid>/
        data
        startedat
        hashstates/<algorithm>/<offset>
  blobs/
    sha256/<aa>/<digest>/data
```

The layouts are structurally identical, differing only in the root prefix (`docker/registry/v2` vs `oci/v1`). This allows Kraken to use the same internal logic for both formats.

## Implementation Details

### Manifest Parsing

Kraken automatically detects the manifest format when content is uploaded:

1. Attempts to parse as Docker v2 manifest
2. If that fails, attempts OCI manifest
3. If that fails, attempts Docker v2 manifest list
4. Finally attempts OCI index

This fallback chain ensures maximum compatibility with various client implementations.

### Path Routing

Kraken's path routing uses flexible regular expressions that support:

- Full absolute paths (e.g., `/var/lib/registry/docker/registry/v2/...`)
- Relative paths with prefix (e.g., `docker/registry/v2/...` or `oci/v1/...`)
- Short form paths (e.g., `v2/...`)

This flexibility ensures compatibility with various storage backends and deployment configurations.

### Namepath Pathers

Two new pather implementations were added to handle OCI paths:

#### OciTagPather
Generates paths for OCI tags in the format:
```
oci/v1/repositories/<repo>/_manifests/tags/<tag>/current/link
```

Converts between path format and `repo:tag` name format.

#### ShardedOciBlobPather
Generates sharded paths for OCI blobs in the format:
```
oci/v1/blobs/sha256/<first-2-chars>/<full-digest>/data
```

Blobs are sharded by the first two characters of their SHA256 digest to prevent filesystem limitations with large numbers of files in a single directory.

## Configuration

No special configuration is required to enable OCI support. Kraken automatically:

- Accepts OCI media types in the `Accept` header
- Routes requests to appropriate storage paths based on client capabilities
- Serves content in the format requested by the client

### Registry Backend Configuration

When using a registry backend (e.g., Docker Hub, Harbor, ECR), Kraken will:

1. Automatically include OCI media types in manifest requests via the `Accept` header
2. Handle authentication challenges from the upstream registry
3. Pass through OCI content transparently

Example configuration remains unchanged:
```yaml
backends:
  - namespace: library/*
    backend:
      registry:
        address: https://index.docker.io
        security:
          basic:
            username: <username>
            password: <password>
```

## Usage Examples

### Pushing OCI Artifacts

Using standard OCI-compliant tools:

```bash
# Using ORAS (OCI Registry As Storage) to push a Helm chart
oras push localhost:30081/helm/mychart:1.0.0 \
  --artifact-type application/vnd.cncf.helm.chart.content.v1.tar+gzip \
  mychart-1.0.0.tgz

# Using docker/buildx with OCI format
docker buildx build --output type=oci,dest=image.tar .
docker load < image.tar
docker tag <image> localhost:30081/myapp:latest
docker push localhost:30081/myapp:latest
```

### Pulling OCI Artifacts

```bash
# Pull using ORAS
oras pull localhost:30081/helm/mychart:1.0.0

# Pull using Docker (works for both Docker and OCI formats)
docker pull localhost:30081/myapp:latest
```

### Using with Helm

Configure Helm to use Kraken as an OCI registry:

```bash
# Add repository (OCI registries don't use 'helm repo add')
helm registry login localhost:30081

# Push chart
helm push mychart-1.0.0.tgz oci://localhost:30081/helm

# Install chart
helm install myrelease oci://localhost:30081/helm/mychart --version 1.0.0
```

## Migration from Docker Registry v2

No migration is required. Kraken maintains both Docker Registry v2 and OCI v1 support concurrently:

- Existing Docker images remain accessible via Docker Registry v2 paths
- New OCI artifacts are stored in OCI v1 paths
- Clients can request content in either format via the `Accept` header
- The same blob storage is shared between both formats (blobs are content-addressed)

## Compatibility

### Client Compatibility

Kraken's OCI support is compatible with:

- Docker CLI (v20.10+)
- containerd
- Podman
- ORAS (OCI Registry As Storage)
- Helm (v3.8+ for OCI support)
- kubectl with OCI image references
- Skopeo
- Cosign
- Any OCI Distribution Spec compliant client

### Backend Compatibility

When using Kraken with a registry backend, OCI support works with:

- Docker Hub
- Google Container Registry (GCR)
- Amazon Elastic Container Registry (ECR)
- Azure Container Registry (ACR)
- Harbor (v2.0+)
- GitHub Container Registry (GHCR)
- GitLab Container Registry
- JFrog Artifactory
- Any OCI-compliant registry

## Limitations

1. **Tag Mutation**: Similar to Docker Registry v2 limitations in Kraken, mutating tags with OCI artifacts is subject to the same caching constraints. See [index.md Limitations](index.md#limitations) for details.

2. **Replication**: Cross-cluster replication works for both Docker and OCI formats, but relies on the tag structure. Ensure your replication rules account for OCI artifact paths if needed.

3. **Storage**: OCI artifacts use the same storage backend as Docker images. Ensure your storage backend has sufficient capacity for both types of content.

## Troubleshooting

### 415 Unsupported Media Type

If you receive a `415 Unsupported Media Type` error:

1. Verify your client is sending the correct `Content-Type` header
2. Check that your client supports OCI media types
3. Ensure you're using a recent version of your client tool

### Manifest Not Found

If manifests are not found:

1. Verify the path format matches either Docker Registry v2 or OCI v1
2. Check storage backend permissions
3. Review Kraken logs for path resolution errors

### Authentication Failures

For authentication issues with OCI artifacts:

1. Ensure credentials are configured for the correct namespace
2. Verify upstream registry supports OCI artifacts (if using registry backend)
3. Check that authentication tokens include necessary scopes for OCI operations

## Technical References

- [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec)
- [OCI Image Specification](https://github.com/opencontainers/image-spec)
- [ORAS Project](https://oras.land/)
- [Docker Registry v2 API](https://docs.docker.com/registry/spec/api/)

## Implementation Details

For detailed implementation information, see the following components:

- **Path handling**: `lib/dockerregistry/paths.go` - Regex patterns for parsing both Docker v2 and OCI v1 paths
- **Manifest parsing**: `utils/dockerutil/dockerutil.go` - Support for OCI manifest and index types
- **Pather implementations**: `lib/backend/namepath/pather.go` - OciTagPather and ShardedOciBlobPather
- **Storage driver**: `lib/dockerregistry/storage_driver.go` - Dual path layout documentation and error handling
