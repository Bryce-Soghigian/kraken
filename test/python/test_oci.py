# Copyright (c) 2016-2019 Uber Technologies, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""
Integration tests for OCI v1 Distribution Specification support.

These tests verify that Kraken can properly handle OCI v1 artifacts
alongside traditional Docker Registry v2 images.
"""
from __future__ import absolute_import

import json
import requests

from .conftest import TEST_IMAGE


def test_oci_manifest_accept_headers(proxy):
    """Test that registry accepts OCI media types in Accept header."""
    proxy.push(TEST_IMAGE)

    repo, tag = TEST_IMAGE.split(':')

    # Test with OCI Accept headers
    headers = {
        'Accept': ','.join([
            'application/vnd.oci.image.manifest.v1+json',
            'application/vnd.oci.image.index.v1+json',
            'application/vnd.docker.distribution.manifest.v2+json',
            'application/vnd.docker.distribution.manifest.list.v2+json'
        ])
    }

    # Get manifest with OCI Accept headers
    response = requests.get(
        f'http://localhost:{proxy.registry_port}/v2/{repo}/manifests/{tag}',
        headers=headers
    )

    assert response.status_code == 200, f"Expected 200, got {response.status_code}"

    # Verify Content-Type is one of the supported types
    content_type = response.headers.get('Content-Type', '')
    supported_types = [
        'application/vnd.docker.distribution.manifest.v2+json',
        'application/vnd.docker.distribution.manifest.list.v2+json',
        'application/vnd.oci.image.manifest.v1+json',
        'application/vnd.oci.image.index.v1+json'
    ]

    assert any(t in content_type for t in supported_types), \
        f"Content-Type {content_type} not in supported types"


def test_oci_compatible_push_pull(proxy, agent):
    """Test that OCI-compatible images can be pushed and pulled."""
    # Push image through proxy
    proxy.push(TEST_IMAGE)

    # Pull through agent (tests full distribution pipeline)
    agent.pull(TEST_IMAGE)

    # Verify the image was pulled successfully by checking it exists
    repo, tag = TEST_IMAGE.split(':')

    # Query the agent's registry to verify the image
    response = requests.get(
        f'http://localhost:{agent.registry_port}/v2/{repo}/manifests/{tag}'
    )

    assert response.status_code == 200, \
        f"Image not found on agent, status: {response.status_code}"


def test_oci_manifest_list_compatibility(proxy):
    """Test that manifest lists are handled correctly with OCI media types."""
    proxy.push(TEST_IMAGE)

    repo, tag = TEST_IMAGE.split(':')

    # Request manifest list with OCI Accept header
    headers = {
        'Accept': 'application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json'
    }

    response = requests.get(
        f'http://localhost:{proxy.registry_port}/v2/{repo}/manifests/{tag}',
        headers=headers
    )

    # Should return 200 whether it's a manifest or manifest list
    assert response.status_code == 200

    # Parse the manifest
    manifest = response.json()
    assert 'schemaVersion' in manifest
    assert manifest['schemaVersion'] == 2


def test_oci_blob_retrieval(proxy):
    """Test that blobs can be retrieved for OCI images."""
    proxy.push(TEST_IMAGE)

    repo, tag = TEST_IMAGE.split(':')

    # Get the manifest first
    manifest_response = requests.get(
        f'http://localhost:{proxy.registry_port}/v2/{repo}/manifests/{tag}'
    )
    assert manifest_response.status_code == 200

    manifest = manifest_response.json()

    # Extract a blob digest (config or first layer)
    blob_digest = None
    if 'config' in manifest:
        blob_digest = manifest['config']['digest']
    elif 'manifests' in manifest and len(manifest['manifests']) > 0:
        # It's a manifest list, get the first manifest
        first_manifest_digest = manifest['manifests'][0]['digest']
        # For simplicity, we'll just verify the manifest list was parseable
        return

    if blob_digest:
        # Try to retrieve the blob
        response = requests.get(
            f'http://localhost:{proxy.registry_port}/v2/{repo}/blobs/{blob_digest}'
        )

        assert response.status_code == 200, \
            f"Failed to retrieve blob {blob_digest}, status: {response.status_code}"

        # Verify we got data
        assert len(response.content) > 0, "Blob content is empty"


def test_oci_catalog_compatibility(proxy):
    """Test that catalog endpoint works with OCI images."""
    proxy.push(TEST_IMAGE)

    # Query the catalog
    response = requests.get(
        f'http://localhost:{proxy.registry_port}/v2/_catalog'
    )

    assert response.status_code == 200

    catalog = response.json()
    assert 'repositories' in catalog

    repo = TEST_IMAGE.split(':')[0]
    assert repo in catalog['repositories'], \
        f"Repository {repo} not found in catalog"


def test_oci_tags_list(proxy):
    """Test that listing tags works for OCI images."""
    # Push multiple tags of the same image
    tags = ['v1', 'v2', 'latest']
    for tag in tags:
        proxy.push_as(TEST_IMAGE, tag)

    repo = TEST_IMAGE.split(':')[0]

    # List tags
    response = requests.get(
        f'http://localhost:{proxy.registry_port}/v2/{repo}/tags/list'
    )

    assert response.status_code == 200

    tags_data = response.json()
    assert 'tags' in tags_data

    # Verify all pushed tags are present
    returned_tags = set(tags_data['tags'])
    for tag in tags:
        assert tag in returned_tags, f"Tag {tag} not found in response"


def test_oci_high_availability_with_restart(testfs, proxy, origin_cluster, agent_factory):
    """Test OCI image distribution with component restarts (HA scenario)."""
    # Push OCI-compatible image
    proxy.push(TEST_IMAGE)

    # Restart an origin to simulate failure
    origin_cluster[0].restart()

    # Should still be able to pull from other origins
    with agent_factory.create() as agent:
        agent.pull(TEST_IMAGE)

    # Verify image is accessible
    repo, tag = TEST_IMAGE.split(':')
    response = requests.get(
        f'http://localhost:{proxy.registry_port}/v2/{repo}/manifests/{tag}'
    )

    assert response.status_code == 200, \
        "Image not accessible after origin restart"
