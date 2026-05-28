# Learning Notes Index

This index is a retrieval map for reusable project lessons. Open only notes relevant to the current task.

## High-Signal Recurring Lessons

- 2026-05-19 - [Streaming Adapter Compatibility](2026-05-19-streaming-adapter-compatibility.md): open when changing provider interfaces, streaming adapters, SSE framing, or first-token fallback boundaries.
- 2026-05-19 - [Immutable Registry Snapshots](2026-05-19-immutable-registry-snapshots.md): open when changing registry/profile snapshot fields, lookup helpers, capability filters, or health overlays.
- 2026-05-19 - [Context Processor Timeout Isolation](2026-05-19-context-processor-timeout-isolation.md): open when adding mutable request processors, nested request cloning, timeout handling, or fail-open pipeline behavior.
- 2026-05-19 - [Job Descriptor Trust Boundaries](2026-05-19-job-descriptor-trust-boundaries.md): open when mapping client metadata/headers into routing, policy, classifier, or descriptor logging fields.
- 2026-05-28 - [Compiled Policy Cache Semantics](2026-05-28-compiled-policy-cache-semantics.md): open when changing policy compilation, route hint mapping, cache reload, or constraint aggregation.
- 2026-05-19 - [Backlog Triage State Sync](2026-05-19-backlog-triage-state-sync.md): open when issue labels, `.ai/tasks.json`, and next-task selection might disagree.
- 2026-05-19 - [Classifier Keyword Boundaries](2026-05-19-classifier-keyword-boundaries.md): open when adding or widening classifier keyword rules for task, risk, sensitivity, or feature extraction.

## Routing

- 2026-05-19 - [Immutable Registry Snapshots](2026-05-19-immutable-registry-snapshots.md): immutable in-memory registry snapshots; open when adding model metadata, provider lookups, provider-model mappings, capability filters, or health overlays.
- 2026-05-19 - [Job Descriptor Trust Boundaries](2026-05-19-job-descriptor-trust-boundaries.md): trusted auth context vs untrusted client hints; open when building or consuming `JobDescriptor`.

## Policy

- 2026-05-28 - [Compiled Policy Cache Semantics](2026-05-28-compiled-policy-cache-semantics.md): compiled policy snapshots, tenant/project cache lookup, route hint normalization, reload safety, constraint merge semantics.
- 2026-05-19 - [Job Descriptor Trust Boundaries](2026-05-19-job-descriptor-trust-boundaries.md): client `risk_level`, `task_type`, and sensitivity metadata stay hints until classifier/policy truth is applied.

## Providers

- 2026-05-19 - [Streaming Adapter Compatibility](2026-05-19-streaming-adapter-compatibility.md): optional streaming provider interface; open when adding adapters, stream support, or unsupported-stream behavior.

## Auth

No notes yet.

## Testing

- 2026-05-19 - [Classifier Keyword Boundaries](2026-05-19-classifier-keyword-boundaries.md): keyword boundary regression coverage; open when classifier terms can overlap product/package names.

## Operations

- 2026-05-19 - [Backlog Triage State Sync](2026-05-19-backlog-triage-state-sync.md): sync `05-issues/` triage labels with canonical `.ai` state on `main`; open when `continue` would pick the wrong task.

## Performance

- 2026-05-28 - [Compiled Policy Cache Semantics](2026-05-28-compiled-policy-cache-semantics.md): validate and compile before atomic cache swap; fast-path evaluation should do only in-memory matcher checks.
- 2026-05-19 - [Classifier Keyword Boundaries](2026-05-19-classifier-keyword-boundaries.md): deterministic in-memory keyword matching; prefer bounded scans and explicit terms over broad prompt-derived outputs.
- 2026-05-19 - [Streaming Adapter Compatibility](2026-05-19-streaming-adapter-compatibility.md): first emitted stream chunk is the fallback boundary; do not retry after data is flushed.
- 2026-05-19 - [Immutable Registry Snapshots](2026-05-19-immutable-registry-snapshots.md): clone map/slice metadata on snapshot ingress and egress while keeping lookups deterministic and in-memory.
- 2026-05-19 - [Context Processor Timeout Isolation](2026-05-19-context-processor-timeout-isolation.md): hard timeouts do not stop goroutines; deeply clone mutable request state and commit only on successful completion.
- 2026-05-19 - [Job Descriptor Trust Boundaries](2026-05-19-job-descriptor-trust-boundaries.md): descriptor construction stays in-memory and avoids logging raw metadata or prompt/message text.

## Security

- 2026-05-19 - [Classifier Keyword Boundaries](2026-05-19-classifier-keyword-boundaries.md): avoid false auth/secret/security sensitivity from substring matches like `token` inside `tokenizer`.
- 2026-05-19 - [Job Descriptor Trust Boundaries](2026-05-19-job-descriptor-trust-boundaries.md): safe descriptor logs should expose derived fields and counts, not raw metadata values, metadata key names, prompt text, or message content.

## Chronological Notes

| Date | Title | Retrieval cues |
| --- | --- | --- |
| 2026-05-19 | [Streaming Adapter Compatibility](2026-05-19-streaming-adapter-compatibility.md) | provider, streaming, SSE, optional interface, first token boundary |
| 2026-05-19 | [Immutable Registry Snapshots](2026-05-19-immutable-registry-snapshots.md) | registry, snapshot, map clone, slice clone, capability filter, health overlay |
| 2026-05-19 | [Backlog Triage State Sync](2026-05-19-backlog-triage-state-sync.md) | issue state, triage labels, .ai/tasks.json, next task drift |
| 2026-05-19 | [Context Processor Timeout Isolation](2026-05-19-context-processor-timeout-isolation.md) | contextproc, hard timeout, goroutine, race, nested metadata/tools clone |
| 2026-05-19 | [Job Descriptor Trust Boundaries](2026-05-19-job-descriptor-trust-boundaries.md) | JobDescriptor, trusted auth, untrusted metadata hints, risk hint, safe logging |
| 2026-05-19 | [Classifier Keyword Boundaries](2026-05-19-classifier-keyword-boundaries.md) | classifier, keywords, word boundary, false positive, sensitivity hints |
| 2026-05-28 | [Compiled Policy Cache Semantics](2026-05-28-compiled-policy-cache-semantics.md) | policy compiler, cache reload, route hints, constraint merge, tenant project scope |
