
# Tag and Manifest Entities for OCI Artifacts -- Design

**Author:** Shreya Baranwal
**Mentor:** Evan Anderson
**Status:** Draft
**Related issue:**
[#6954](https://github.com/mindersec/minder/issues/6954)

---

## Abstract


**TL;DR:** Today a DockerHub/Quay artifact entity is one `repo:tag`, but it evaluates every tag in the repository and mixes a mutable pointer (the tag) with immutable content (the digest). GHCR uses a different, package-based model. This document compares three shapes against six policy scenarios: keeping today's model (A), separate Tag and Manifest entities (B), and Manifest-only entities with tags stored as data (C). Option B, with platform data kept inline (B1), looks like the strongest starting point. Under every option, discovery, reconciliation and tag-move detection still need new mechanisms.

### Decisions needed

1. Option B (with B1) or C2 as the direction to prototype.
2. New Tag and Manifest entity types, or `ENTITY_ARTIFACTS` with a `kind` property.
3. Order of work: Repository entities first, or Tag/Manifest entities first.
4. Signature checks on the top-level index only for the first version.
5. How to detect tag movement: comparing a recorded digest, or a property-change event.

Code references point to `main` at [`900dae34d`](https://github.com/mindersec/minder/tree/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65).

Throughout this document, statements about current behaviour are marked as follows:

- **Verified:** Confirmed by code, a PR, an issue, or fetched documentation.
- **Inferred:** Based on code inspection but not verified by running it.
- **Unverified:** Not yet confirmed by code inspection, execution, or documentation.

## 1. Context

Since [#6808](https://github.com/mindersec/minder/pull/6808), DockerHub and Quay artifact entities represent a repository and a tag. The entity name is typically `repo:tag`, while its `upstream_id` is `repo@<digest the tag points to>` ([`dockerhub.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/dockerhub/dockerhub.go#L170-L193)). **Verified.**

GitHub Container Registry (GHCR) uses a different model: an artifact entity represents a package containing multiple versions ([`artifact.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/github/properties/artifact.go#L156-L171)). **Verified.**

These models create several problems:

| Problem | Evidence and status |
|---|---|
| DockerHub and Quay entities combine a mutable tag reference with an immutable digest. When a tag moves, the `upstream_id` changes while the entity remains the same. | #6954; [`service.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/entities/properties/service/service.go#L212-L221). **Verified.** |
| A `repo:tag` entity evaluates every tag in the repository rather than only the tag named by the entity. | [`oci.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/oci.go#L211). **Inferred.** |
| A digest referenced by multiple tags can be evaluated repeatedly on DockerHub and Quay. | [`oci.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/oci.go#L248-L249); #6775 review. **Verified.** |
| Untagged manifests are not discovered on DockerHub and Quay, or are filtered out on GitHub. | [`versionsfilter.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/artifact/versionsfilter.go#L71-L74). **Verified.** |
| Evaluation history does not record the digest that was evaluated, making it difficult to identify the exact content associated with a historical result. | [`artifact.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/engine/ingester/artifact/artifact.go#L125-L132). **Verified.** |
| DockerHub and Quay artifacts are only registered manually. There are no artifact push webhooks, and the reminder service currently covers repositories rather than artifacts. | [`dockerhub.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/dockerhub/dockerhub.go#L237-L242); [`reminder.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/reminder/reminder.go#L206-L207). **Verified.** |

This is consistent with the discussion in the existing issues and PRs. Evan Anderson suggested in #6929 that tags and manifests should have separate entities. In #6808, the question of how to model multiple tags pointing to the same digest was left open.

The goal of this design is to determine which entity model best represents OCI artifacts and how it supports policy evaluation, history, discovery, and lifecycle management.

## 2. Goals and non-goals

### Goals

- Define how OCI repositories, tags, manifests, and indexes should be represented.
- Establish a stable `upstream_id` for each entity type.
- Define how tag movement affects entity identity and evaluation history.
- Compare the proposed models against concrete policy scenarios before implementation.
- Describe how registry push events and GitHub package events map to the selected model.
- Identify migration requirements for existing entity data.

Repository entities and webhook support are planned as a subsequent milestone under [#6436](https://github.com/mindersec/minder/issues/6436).

### Non-goals

- Implementing the design or making schema and protobuf changes.
- Implementing webhook registration or handlers.
- Adding support for new registries such as Harbor, ECR, GAR, or ACR.
- Adding repository-settings policies, such as immutable-tag or MFA enforcement.
- Fixing the separate issues listed in Section 14.

## 3. Terminology

| Term | Definition |
|---|---|
| Repository | A named collection of artifacts within a registry, such as `docker.io/library/alpine`. In the OCI Distribution API, this corresponds to the repository name in `/v2/<name>/...`. |
| Tag | A mutable reference within a repository that points to a manifest or index, such as `latest`. |
| Manifest | An immutable document describing an image, including its configuration and layer references. It is identified by its digest. |
| Index | An immutable manifest that references other manifests, typically for different platforms. Also known as a manifest list. |
| Platform manifest | A manifest referenced by an index for a particular platform, such as `linux/arm64`. |
| Digest | A content identifier derived from a cryptographic hash, such as `sha256:294b683c…`. The digest identifies the specific content being referenced. |

For this design, the distinction between a tag and a manifest is important: **a tag can change what it points to, but a manifest's digest identifies immutable content.**

## 4. Current model

DockerHub, Quay, and GHCR currently use different entity models.

| Aspect | DockerHub / Quay | GitHub Container Registry |
|---|---|---|
| Entity | One entity per registered `repo[:tag]`. **Verified.** | One entity per package. **Verified.** |
| Name | For example, `alpine:latest`. A bare `alpine` defaults to `latest` during resolution but is stored as `alpine`. See [`oci/artifact.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/artifact.go#L32-L41). | `owner/name`. See [`artifact.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/github/properties/artifact.go#L80-L100). |
| `upstream_id` | `repo@<tag's top-level digest>`. Since #6862, this is the index digest when the tag resolves to a multi-platform index. See [`oci.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/oci.go#L301-L309). | Numeric package ID. See [`artifact.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/github/properties/artifact.go#L158). |
| What gets evaluated | Every tag in the repository, subject to six-month retention and the rule's `tags` or `tag_regex` filters. One version is produced per tag. See [`oci.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/oci.go#L206-L255). | Every active package version, with one version per digest and all associated tags, subject to the same filters. See [`common.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/github/common.go#L955-L991). |
| Evaluation history | Keyed by `(rule_id, entity_instance_id)`, with one status per evaluation. The checkpoint stores a timestamp. See [`history/service.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/history/service.go#L101-L127). | Same history model. |
| Parent relationship | No parent link. | `originated_from` points to the repository when the entity is created by the `package` webhook. See [`handlers_packages.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/github/webhook/handlers_packages.go#L95-L100). |
| Creation | Manual `RegisterEntity` only. | `package.published` webhook or manual `RegisterEntity`. |

The current model therefore treats the tag as the entity's identity while also using the tag's resolved digest to identify the content. The following options explore ways to separate those concerns.

## 5. Design options

### Option A: Keep the current model — one entity per tag

Each artifact entity represents a repository and a tag.

<img width="230" alt="Option A: two artifact entities, alpine:latest and alpine:3.20, with the same upstream digest" src="https://github.com/user-attachments/assets/57071d9c-be21-4d30-8102-f965631a0deb" />

Both entities can point to the same digest, but each is evaluated independently.

| Entity | `upstream_id` | Multi-platform behaviour |
|---|---|---|
| Artifact (`repo:tag`) | `repo@digest`, which changes when the tag moves. | The index digest is used as the version's `Sha`. Platform manifests remain inside the raw index passed to Rego. See [`oci.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/oci.go#L164-L179). |

**Advantages**

- Requires the fewest conceptual changes to the existing model.
- Keeps tag-based policy selection straightforward.

**Limitations**

- Mutable tag identity and immutable content identity remain coupled.
- The same digest can be evaluated repeatedly through multiple tags.
- Tag movement is not represented as an explicit state transition.
- Content-level policies remain dependent on tag-based discovery.

### Option B: Introduce separate Tag and Manifest entities

Under this model, a Tag represents a mutable reference, while a Manifest represents immutable content. Multiple Tags can point to the same Manifest, and a Manifest can exist without any Tags.

<img width="760" alt="Option B: a repository with Tag entities that point to a Manifest entity, which lists its platform manifests" src="https://github.com/user-attachments/assets/b488824e-c0a7-46ce-bf89-7f430173f29e" />

#### Candidate identities

| Entity | Candidate `upstream_id` | Notes |
|---|---|---|
| Tag | `repo:tag` | Registries do not expose a separate tag ID, so the repository and tag name form the identity. Deleting and recreating the same tag would reuse that identity. The current target digest could be stored in a property such as `oci/digest`. |
| Manifest | `repo@digest` for DockerHub and Quay. For GHCR, either a package version ID or `repo@digest` remains under consideration. | The digest identifies immutable content. Including the repository keeps copies of the same digest in different repositories distinct, addressing the identity concern raised in #6808. |
| Platform manifest | B1: Keep platform manifests as data inside the index Manifest. B2: Represent each platform manifest as its own `repo@digest` entity. | B2 requires a relationship between an index and its child manifests. A child may be referenced by multiple indexes. |

**Advantages**

- Separates mutable references from immutable content.
- Provides a natural place for tag-specific policies and tag-movement history.
- Allows content-level checks to be associated with a digest rather than repeated for every tag.
- Provides a model for manifests that exist without tags, provided they can be discovered.

**Limitations**

- Requires explicit lifecycle handling for Tags and Manifests.
- Requires a mechanism to detect changes to a Tag's target digest.
- Reusing Manifest evaluation results from Tag rules requires additional cross-entity lookup or equivalent functionality.
- B2 introduces relationships that the current `originated_from` model cannot represent.

### Option C: Introduce Manifest entities only, with tags stored as data

This model does not introduce separate Tag entities. Instead, tag information is maintained either on the Manifest or on a Repository entity.

Two variants are possible:

- **C1:** A Manifest stores its current tags in a property such as `oci/tags`.
- **C2:** A Repository stores a mapping from tag names to digests.

<img width="760" alt="Option C: a repository with a tag map and Manifest entities that carry their tags as data" src="https://github.com/user-attachments/assets/3dc6ede0-2a99-43a2-907f-b02a4c87fe88" />

| Entity | Candidate `upstream_id` | Notes |
|---|---|---|
| Manifest | `repo@digest` | Same identity approach as Option B. |
| Tags | No separate entity | When a tag moves, the old Manifest loses the tag and the new Manifest gains it under C1. Under C2, the corresponding Repository mapping is updated. |
| Platform manifest | Same B1/B2 decision as Option B. | |

**Advantages**

- Keeps the entity model smaller.
- Supports content-level evaluation and digest-based deduplication.
- C2 can centralise tag state and potentially simplify repository-level policies.

**Limitations**

- Tag movement is not naturally represented as a first-class entity transition.
- C1 requires updating tag information on the affected Manifests.
- C2 requires Repository-level state and policy support.
- Historical tag movement still requires persisted previous state or an equivalent event history.

## 6. Policy scenarios

The following scenarios help compare the options against actual policy requirements. The table distinguishes what the entity model enables from what still depends on discovery, event delivery, and reconciliation.

| Scenario | A: One entity per tag | B: Tag + Manifest | C: Manifest only |
|---|---|---|---|
| **a. Every image under `platform/supported/` must be signed** | Each `repo:tag` must be registered manually. Selection can use `oci/repository` (#6929). Each registered entity may re-check every tag in the repository. New pushes are not automatically discovered. | Manifest rules can select by `oci/repository` and evaluate each discovered digest. Complete coverage depends on repository enumeration, push-event processing, and reconciliation. | Same content-level model as B. Complete coverage depends on discovery and reconciliation. |
| **b. `latest` must never point to an unsigned manifest** | A rule can filter with `tags: [latest]`, but a tag move is not automatically detected or evaluated. | A rule can select the Tag named `latest` and evaluate its current target digest. Push processing and reconciliation determine how quickly a change is observed. | C1 can select Manifests whose current tags include `latest`; C2 can evaluate the Repository's tag map. Both require tag-state updates and evaluation triggers. |
| **c. Alert when a tag moves to a new digest** | A change to `upstream_id` is not recorded as an explicit movement event. The history does not retain the evaluated digest. | Compare the Tag's previously observed digest with the newly resolved digest. A property-change trigger or persisted prior state is required; neither exists today. | Requires previous and current tag state, either through the Repository map (C2) or equivalent persisted information. |
| **d. Every image must support amd64 and arm64** | Possible through the raw index, but the same content may be checked once per tag. An index without `linux/amd64` may also break version listing. | Under B1, evaluate the index's platform descriptors once per digest. Under B2, platform manifests can be evaluated independently. | Same content-level approach as B, with the same B1/B2 choice. |
| **e. One digest has three tags: how many evaluations occur?** | Three registered entities may each evaluate the same digest. Estimated: three status rows and nine verifications, depending on the evaluation path. **Inferred.** | Manifest-level rules can evaluate the digest once per evaluation cycle; Tag-level rules remain per Tag. Cross-entity reuse is not automatic. | Manifest-level rules can evaluate the digest once per evaluation cycle. |
| **f. An image is pushed by digest without a tag** | Not discovered by the current tag-based model. | Can represent a Manifest without Tags if its digest is discovered through another mechanism. Existing push payloads do not provide sufficient information to guarantee discovery of every untagged Manifest. | Same limitation as B. |

### What the scenarios tell us

Options B and C provide a clearer model for content-level checks and can avoid repeated evaluation of the same digest. However, deduplication depends on rule configuration and evaluation semantics; it is not guaranteed merely by creating Manifest entities.

Option B gives mutable references a first-class identity, which makes tag-specific policies and movement tracking easier to express.

Scenario c still requires previous-state tracking or an equivalent change-detection mechanism regardless of the selected entity model.

The word **"never"** in scenario b also deserves clarification. An asynchronous webhook-based system can detect and report that a tag now points to unsigned content, but it cannot guarantee that the tag never briefly points to such content. Preventing publication of unsigned content would require an enforcement mechanism beyond post-publication evaluation.

## 7. Identity and evaluation history

The main identity question is what remains stable when a tag changes its target.

| Option | Stable `upstream_id` | Behaviour when a tag moves |
|---|---|---|
| A | Not stable: `repo@digest` follows the tag's current target. | The entity and history row remain the same. The `upstream_id` is updated during a subsequent non-stale refresh, such as a profile change or `GetArtifactById` call. See [`refresh_by_id.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/entities/handlers/strategies/entity/refresh_by_id.go#L55-L57). Historical results do not identify the digest that was evaluated. |
| B | Tag: `repo:tag`. Manifest: `repo@digest`. | The Tag keeps its identity while its target digest changes. Each Manifest retains its own identity and history. Recording the evaluated digest in the checkpoint's existing `Digest` field could make Tag history auditable; the field is currently unused by the artifact ingester. See [`checkpoints.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/pkg/entities/v1/checkpoints/checkpoints.go#L37-L39). |
| C | Manifest: `repo@digest`. | Manifest history remains stable, but the model does not inherently preserve what a tag pointed to in the past. That information must be captured separately, either through evaluation output under C1 or through persisted Repository state and history under C2. |

Under B, a useful distinction is:

- **Tag history:** Which digest did the tag point to when a particular evaluation occurred?
- **Manifest history:** What were the results of evaluating a specific immutable digest?

These are related but different questions. The implementation should make both answerable without relying on the current value of a mutable property.

## 8. Events and discovery

The proposed model depends on how each registry exposes tag and manifest changes.

### Existing event payloads

- **DockerHub push:** Includes `push_data.tag`, `pusher`, `pushed_at`, and repository information. The documented payload does not include the digest or a delete event. See the [Docker Hub webhook documentation](https://docs.docker.com/docker-hub/repos/manage/webhooks/), fetched 2026-10-25.
- **Quay Repository Push:** Includes repository and namespace information, `docker_url`, and `updated_tags[]`. The cited notification guide does not establish a digest or delete event in the payload. See the [Quay notification guide](https://docs.quay.io/guides/notifications.html). Newer Red Hat documentation remains unverified.
- **GitHub `package` event:** The current handler processes only `published`. The payload struct includes `package_version.container_metadata.tag.{name,digest}`, but the values are not currently used. Whether GitHub reliably populates the digest field remains unverified. See [`handlers_packages.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/github/webhook/handlers_packages.go#L45-L80).

### Event handling by option

| Event | A: One entity per tag | B: Tag + Manifest | C: Manifest only |
|---|---|---|---|
| DockerHub/Quay push | Upsert the `repo:tag` entity and refresh its properties. | Upsert the Tag, resolve its current digest, upsert the Manifest, update the Tag's target, and evaluate the relevant rules. | Resolve the digest, upsert the Manifest, and update its tag information or the Repository's tag map. Under C1, the previous Manifest may also need updating. |
| GitHub `package.published` | Keep the existing package-based flow. | Use the event's tag and digest if available, otherwise resolve the tag and follow the same upsert flow. | Same discovery and state-update requirements as B. |
| Tag or Manifest deletion | No deletion event is currently handled. | Detect removal through periodic reconciliation and update the Tag and Manifest lifecycle appropriately. | Detect removal through reconciliation and update the Manifest's tag data or Repository map. |

Under every option, deletion detection and drift correction require a reconciliation mechanism that compares stored state with registry state. The existing reminder service covers repositories only.

Push events alone do not guarantee complete discovery. In particular, an event that names a tag is not equivalent to an event that identifies every manifest in a repository.

## 9. Relationships

The current entity model supports an optional parent through `originated_from`, with `ON DELETE CASCADE` semantics. See [`000090_entity_properties.up.sql`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/database/migrations/000090_entity_properties.up.sql#L13).

Issue [#6105](https://github.com/mindersec/minder/issues/6105) mentions entity links as a future capability but does not define their design.

| Relationship | A | B | C | Compatibility with `originated_from` |
|---|---|---|---|---|
| Repository → Tag | Represented through the artifact entity | Yes | Not applicable | Fits the ownership relationship. Deleting a Repository should remove its Tag entities if this is the intended lifecycle. |
| Repository → Manifest | Not explicitly represented | Yes | Yes | Fits the ownership relationship, subject to the chosen retention and deletion rules. |
| Tag → Manifest | Not applicable | Yes, as a mutable many-to-one reference | Not applicable | Does not fit. The relationship changes over time and should not cause cascading deletion. A target-digest property is one possible initial approach. |
| Index → platform Manifest | Not explicitly represented | Only under B2 | Only under the corresponding C-B2 variant | Does not fit a single-parent ownership model because a child Manifest can appear in multiple indexes. |

For the initial implementation, keeping platform manifests inline avoids introducing a many-to-many relationship model. Tag-to-Manifest references should remain distinct from ownership relationships.

## 10. Migration

The migration requirements differ between DockerHub/Quay and GHCR.

### Current deployment state

- No release contains #6808.
- v0.3.2 corresponds to the #6775 merge commit, and DockerHub/Quay `SupportsEntity` returns false there. **Verified.**
- DockerHub/Quay artifact entities exist only on deployments running `main` since 2026-10-05.
- GitHub package entities are already released.

### Migration by option

| Option | DockerHub / Quay | GitHub / GHCR |
|---|---|---|
| A | No migration required. | No migration required. |
| B | Existing `repo:tag` entities can be converted to Tag entities while preserving their names, or deleted and registered again. Deletion may cascade to history. The implementation must decide whether Tags and Manifests become new entity types or use a `kind` property on `ENTITY_ARTIFACTS`. | Migration is optional if the package entity remains the top-level entity. Converting packages into Repository entities and versions into Manifest entities would require migrating released data. |
| C | Existing entities can be converted or deleted and recreated as Manifest entities, using the current `upstream_id` as a starting point for the new identity. | Same broad migration consideration as B. |

Other stored data is associated with entity UUIDs, including evaluation statuses, outputs, alerts, remediations, exceptions, and profiles whose selectors or `params.name` values assume the existing `repo:tag` model. See [`models.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/db/models.go#L551-L595) and [`000120_exceptions.up.sql`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/database/migrations/000120_exceptions.up.sql#L11).

A migration plan should explicitly address whether these records are preserved, transformed, or intentionally discarded.

## 11. Registry request cost

The following estimates describe the current DockerHub/Quay flow per artifact rule and evaluation. They are inferred from code inspection and `go-containerregistry` v0.22.1, rather than confirmed through request-level measurements.

| Operation | Approximate requests per tag | Reference |
|---|---|---|
| List tags | Paginated; performed once per repository | [`oci.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/oci.go#L49-L89) |
| `remote.Image` for creation time | One manifest GET, potentially one child-manifest GET for an index, and one config-blob request if no `created` annotation exists | [`oci.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/oci.go#L220-L230) |
| `GetDigest` | One manifest GET | [`oci.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/oci.go#L242) |
| `GetRawManifest` | One manifest GET | [`artifact.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/engine/ingester/artifact/artifact.go#L222) |
| Signature lookup | One manifest GET for the image and one for the `.sig` tag | [`container.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/verifier/sigstore/container/container.go#L409-L444) |
| **Estimated total** | **Approximately 5–7 requests per tag, including 5–6 manifest GETs in the described path** | Inferred |

### Cost multipliers

- Artifact ingestion is not cached, so each artifact rule repeats the ingestion work. See [`executor.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/engine/executor.go#L108-L112).
- Under Option A, registering multiple tags from the same repository can repeat repository-wide processing.

### Rate limits

Docker Hub documents limits of 100 pulls per six hours per IP for anonymous users and 200 for authenticated personal accounts. Manifest GET requests count toward these limits; HEAD requests do not. See the [Docker Hub usage and pull limits documentation](https://docs.docker.com/docker-hub/usage/pulls/), fetched 2026-10-25.

The current OCI listing and manifest calls do not provide credentials in the cited code paths. See [`oci.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/oci.go#L136). The resulting impact on anonymous rate limits is inferred.

For example, a repository with 20 tags inside the retention window may require approximately 100–120 manifest GETs for one rule. **This is an estimate, not a measured result.**

Quay's applicable limits remain unverified.

### Expected cost under B and C

- A Manifest evaluation is expected to require roughly three GET requests in the described path: one for the raw manifest and two for signature lookup.
- A Tag evaluation adds a digest-resolution request. This might be implemented with a HEAD request on Docker Hub, although #6862 retained GET for consistency.
- Event-driven evaluation could reduce repeated repository-wide scans, provided reconciliation and recovery from missed events remain available.

These estimates should be validated through request-level measurements before being used as implementation guarantees.

## 12. Tentative proposal

Option B currently appears to be the strongest starting point for discussion, using B1 for the initial scope.

The proposed model is:

- **Manifest entities:** Identified by `repo@digest`. They represent immutable content and carry content-level checks, including signatures, platform support, and SBOM-related policies.
- **Tag entities:** Identified by `repo:tag`. They represent mutable references and store the current target digest in a property such as `oci/digest`.
- **Repository relationships:** Both Tags and Manifests are children of a future Repository entity through `originated_from`.
- **Tag-to-Manifest reference:** Represented initially as a property rather than an ownership relationship.
- **Platform manifests:** Kept inline in the index under B1. Separate platform Manifest entities can be considered later if independent platform-level policies require them.

### Why Option B?

- It separates mutable tag identity from immutable content identity.
- It gives tag-specific policies and tag-movement history a natural place in the entity model.
- It supports digest-level policy evaluation without requiring each tag to represent the same content independently.
- Its event flow can work with push payloads that identify tags but require an additional request to resolve their current digest.
- It is consistent with the direction discussed in #6929.

### What could change this decision?

- If repository-level rules over a tag-to-digest map can express tag policies and movement detection cleanly, C2 may offer a simpler design with fewer entities.
- If registries cannot enumerate untagged manifests, complete discovery of those manifests remains unresolved under all three options, reducing one of B's advantages.
- If per-platform signatures or policies become a requirement, B2 may be necessary. This would also require a relationship model that supports a platform Manifest being referenced by multiple indexes.
- If maintainers want GHCR to converge on the same model, the migration cost of released GitHub package entities may favour keeping the package as the top-level entity initially.

The proposal remains tentative until the open questions below are resolved.

## 13. Open questions

1. **Entity types:** Should Tag and Manifest become separate entity types, or should `ENTITY_ARTIFACTS` use a `kind` property? This affects selectors, protobuf definitions, and the CLI.

2. **GHCR identity:** Should a GHCR Manifest use the package version ID or `repo@digest` as its `upstream_id`? Does GHCR expose platform manifests as separate untagged versions? This remains unverified; see #6829.

3. **Untagged manifest discovery:** Can any target registry enumerate untagged manifests? The OCI Distribution API's tag-listing endpoint alone does not establish complete manifest discovery.

4. **Tag movement:** Should changes be detected through a property-change trigger or through comparison with previously persisted state? Neither mechanism exists today.

5. **Cross-entity evaluation:** Should a Tag rule evaluate its target digest independently or reuse an existing Manifest result? No cross-entity result lookup exists today. **Inferred.**

6. **Retention:** Which retention or aging-out policy determines when Manifests should be removed? Under B and C, the number of Manifest entities grows with every push. See the #6779 discussion.

7. **Reconciliation:** Is periodic tag-list reconciliation acceptable given registry rate limits, and how frequently should it run?

8. **Signature scope:** Should signatures be checked on the top-level index, the individual platform manifests, or both? See #6829.

9. **Failure semantics:** How should temporary registry failures be distinguished from confirmed tag or manifest deletion?

10. **Event ordering:** How should duplicate and out-of-order push events be handled so that stale events cannot overwrite newer tag state?

11. **Sequencing:** What is the smallest useful first step: Manifest and Tag entities registered manually, or Repository entities with discovery first?

## 14. Related findings to track separately

These findings are related to the current OCI implementation but are outside the scope of this design.

| Finding | Status |
|---|---|
| `tags: [a, b]` may never match on DockerHub/Quay because each version carries one tag while the filter uses `HasAll`. See [`versionsfilter.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/artifact/versionsfilter.go#L115-L118) and [`oci.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/oci.go#L248). | Inferred. |
| OCI listing and manifest calls do not pass credentials in the cited code paths, potentially affecting private repositories and rate limits. See [`oci.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/oci.go#L136). | No authentication option in the cited code is verified; the impact is inferred. |
| An index without `linux/amd64` may cause `getImage` to fail and, consequently, the entire version listing to fail. See [`oci.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/providers/oci/oci.go#L220-L223). | Inferred. |
| `GetArtifactByName` requires `owner/repo/name`, which may prevent lookup of slash-free DockerHub names such as `alpine:latest`. See [`handlers_artifacts.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/controlplane/handlers_artifacts.go#L55-L68). | Inferred. |
| A repeated GitHub `package.published` event for an existing package may encounter the unique-name constraint and be dropped. See [`entity_creator.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/entities/service/entity_creator.go#L144-L166) and [`handler.go`](https://github.com/mindersec/minder/blob/900dae34d72c69bdc1fd9f86ace9aeeef03a9d65/internal/entities/handlers/handler.go#L78-L85). | Inferred. |


## 15. Does this generalize beyond OCI?

This question was raised by @intelligent-ears on #6954: at the Oct 22 community meeting, maintainers noted that release-style entities (e.g. npm, PyPI) have a similar gap, and Evan sketched a hierarchy along the lines of `Account → Distribution channel → Release → Artifact`.That hierarchy comes from the Oct 22, 2026 community meeting notes: ‘Evan came up with a quick set of possible entities around releases: Account (admin group within the distribution), Distribution channel (npm package, pypi package), Release (specific version), Artifact (component in that release, e.g. platform-specific binary).’ Verified.

**npm** has a concept that maps closely onto Tag and Manifest:

- A **dist-tag** (e.g. `latest`, `beta`) is "a human-readable label... used to provide an alias instead of version numbers" ([npm docs](https://docs.npmjs.com/adding-dist-tags-to-packages), [`npm dist-tag`](https://docs.npmjs.com/cli/v10/commands/npm-dist-tag)). **Verified.** A dist-tag maps to a single version, and `latest` moves to the newest version on every default `npm publish`. **Verified.** Any dist-tag, including `latest`, can be repointed with `npm dist-tag add`; the docs don't say outright that this replaces an existing mapping, but that follows directly from "maps to a single version." **Inferred.**
- A published `name@version` can never be republished or reused, even after `npm unpublish` removes it (subject to npm's unpublish policy). Its content is identified by a hash: `dist.integrity` (SHA-512) and the legacy `shasum` (SHA-1). **Verified.** `npm deprecate` is a reversible warning flag and does not touch the content. **Inferred.**
- A single version can carry more than one dist-tag at once: checking the live registry shows the `npm` package itself has both `latest` and `next-12` dist-tags pointing at the same version, `12.2.1`. **Verified** (observed directly, not stated in docs).

This maps onto the design the same way OCI does: **dist-tag ≈ Tag** (mutable, name-addressed, points to one target at a time, though a target can have several names), and **version ≈ Manifest** (immutable, content-addressed, can exist with zero or more tags pointing to it).

**PyPI** does not have an equivalent. There is no tag concept; files are grouped under each version, and each file carries a `sha256` and other hashes with a filename that can never be reused. **Verified.** Yanking a release hides it without deleting it, and can be undone. **Verified.** One caveat: PyPI files are fetched by filename and carry hashes for verification, not fetched by hash the way OCI blobs are, so calling them "content-addressed" in the same sense as a Manifest digest may overstate it. Whether new files can be added to an existing version after release remains **unverified**.

So the split doesn't generalize uniformly: it matches a registry that has a movable, name-based alias (npm, OCI) but has no direct counterpart where there's no such alias (PyPI, as checked here). That's consistent with treating Tag as "the mutable pointer a particular ecosystem happens to expose," rather than a feature every ecosystem must have.