# SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

# End-to-end tests for artifact fixtures: the raw index and the canned
# verification results both have to reach the rule through the real
# artifact ingester. TestMultiArchNoNetwork runs this file with outbound
# HTTP disabled; it lives outside testdata/ so TestTestDir does not also
# run it without that guard.

ENTITY = {"owner": "acme", "name": "app", "type": "container"}

def index(platforms):
    manifests = []
    for i, p in enumerate(platforms):
        os, arch = p.split("/")
        manifests.append({
            "mediaType": "application/vnd.oci.image.manifest.v1+json",
            "digest": "sha256:" + (str(i) * 64),
            "size": 1,
            "platform": {"os": os, "architecture": arch},
        })
    return json.encode({
        "schemaVersion": 2,
        "mediaType": "application/vnd.oci.image.index.v1+json",
        "manifests": manifests,
    })

MULTIARCH = index(["linux/amd64", "linux/arm64"])

def run(manifest, verification):
    return eval(
        rule = "multiarch_rule",
        entity = ENTITY,
        artifacts = artifact(
            registry = "registry.example.test",
            versions = [version(tags = ["v1"], manifest = manifest, verification = verification)],
        ),
    )

def test_multiarch_signed():
    res = run(MULTIARCH, signed(san = "https://github.com/acme/app/.github/workflows/build.yml@refs/heads/main"))
    assert.eq(res["status"], "pass")

def test_multiarch_unsigned():
    res = run(MULTIARCH, unsigned())
    assert.eq(res["status"], "fail")
    assert.contains(res["details"], "image is not signed and verified")

def test_multiarch_signed_without_certificate_fields():
    # signed() with no certificate arguments used to leave Signature nil,
    # which crashed the ingester when the result was marked verified.
    res = run(MULTIARCH, signed())
    assert.eq(res["status"], "pass")

def test_multiarch_missing_arm64():
    res = run(index(["linux/amd64"]), signed())
    assert.eq(res["status"], "fail")
