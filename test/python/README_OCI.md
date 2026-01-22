# OCI v1 Integration Tests

This directory contains integration tests for OCI v1 Distribution Specification support in Kraken.

## Test Coverage

The `test_oci.py` file includes the following integration tests:

1. **test_oci_manifest_accept_headers** - Verifies that the registry accepts OCI media types in Accept headers
2. **test_oci_compatible_push_pull** - Tests pushing and pulling OCI-compatible images through the full distribution pipeline
3. **test_oci_manifest_list_compatibility** - Verifies manifest list handling with OCI media types
4. **test_oci_blob_retrieval** - Tests blob retrieval for OCI images
5. **test_oci_catalog_compatibility** - Verifies catalog endpoint works with OCI images
6. **test_oci_tags_list** - Tests listing tags for OCI images
7. **test_oci_high_availability_with_restart** - Tests OCI distribution with component restarts (HA scenario)

## Prerequisites

1. **Python 3.7+** with pip
2. **Docker** installed and running
3. **Go 1.13+** for building Kraken components

## Setup

### 1. Install Python Dependencies

```bash
# From the project root
python3 -m venv venv
source venv/bin/activate
pip install -r requirements-tests.txt
```

### 2. Build Kraken Components

```bash
# From the project root
make images
```

### 3. Start Kraken Devcluster

```bash
# From the project root
make devcluster
```

This will start:
- Origin servers (3 instances)
- Tracker
- Build-index
- Proxy
- TestFS backend

Wait for all components to be healthy (about 10-30 seconds).

## Running Tests

### Run All OCI Tests

```bash
cd test/python
python3 -m pytest test_oci.py -v
```

### Run Specific Test

```bash
cd test/python
python3 -m pytest test_oci.py::test_oci_manifest_accept_headers -v
```

### Run with More Output

```bash
cd test/python
python3 -m pytest test_oci.py -v -s
```

## Running Unit Tests

Unit tests for OCI support can be run without the devcluster:

```bash
# From project root

# Test OCI pathers
go test -v ./lib/backend/namepath -run "OCI"

# Test OCI manifest parsing
go test -v ./utils/dockerutil -run "OCI"

# Test OCI path regex
go test -v ./lib/dockerregistry -run "TestBlobsPath|TestRepositoriesPath|TestLayersPathGetDigest|TestManifestsPathGetDigest"

# Run all tests
make test
```

## Test Architecture

The integration tests use the existing test infrastructure:

- **Components** (`components.py`) - Manages Kraken component lifecycles
- **Fixtures** (`conftest.py`) - Provides pytest fixtures for proxy, agent, etc.
- **Test helpers** (`utils.py`) - Utility functions for testing

The OCI tests reuse the same fixtures as the Docker tests, ensuring that OCI support works seamlessly with the existing infrastructure.

## Troubleshooting

### Tests Hang or Timeout

If tests hang, check that the devcluster is running:

```bash
# Check if containers are running
docker ps | grep kraken

# Check logs for a component
docker logs kraken-proxy-01
```

### Port Conflicts

If you see port binding errors, make sure nothing else is using ports 16000-16100:

```bash
lsof -i :16000-16100
```

### Image Pull Failures

Ensure Docker can pull images:

```bash
docker pull alpine:latest
```

## Cleaning Up

Stop the devcluster:

```bash
# From project root
make stop-devcluster

# Or manually
docker ps -a | grep kraken | awk '{print $1}' | xargs docker rm -f
```

## CI/CD Integration

To run these tests in CI:

```yaml
# Example GitHub Actions workflow
- name: Setup Python
  uses: actions/setup-python@v4
  with:
    python-version: '3.9'

- name: Install test dependencies
  run: pip install -r requirements-tests.txt

- name: Start Kraken devcluster
  run: make devcluster

- name: Run OCI integration tests
  run: |
    cd test/python
    pytest test_oci.py -v
```

## Contributing

When adding new OCI-related tests:

1. Follow the existing test patterns in `test_docker.py`
2. Use descriptive test names starting with `test_oci_`
3. Add docstrings explaining what the test verifies
4. Ensure tests clean up after themselves
5. Test both success and failure cases where applicable
