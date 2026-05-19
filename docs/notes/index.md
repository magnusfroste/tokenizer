# Learning Notes Index

This index is a retrieval map for reusable project lessons. Open only notes relevant to the current task.

## High-Signal Recurring Lessons

- 2026-05-19 - [Streaming Adapter Compatibility](2026-05-19-streaming-adapter-compatibility.md): open when changing provider interfaces, streaming adapters, SSE framing, or first-token fallback boundaries.
- 2026-05-19 - [Immutable Registry Snapshots](2026-05-19-immutable-registry-snapshots.md): open when changing registry/profile snapshot fields, lookup helpers, capability filters, or health overlays.
- 2026-05-19 - [Context Processor Timeout Isolation](2026-05-19-context-processor-timeout-isolation.md): open when adding mutable request processors, nested request cloning, timeout handling, or fail-open pipeline behavior.
- 2026-05-19 - [Backlog Triage State Sync](2026-05-19-backlog-triage-state-sync.md): open when issue labels, `.ai/tasks.json`, and next-task selection might disagree.

## Routing

- 2026-05-19 - [Immutable Registry Snapshots](2026-05-19-immutable-registry-snapshots.md): immutable in-memory registry snapshots; open when adding model metadata, provider lookups, provider-model mappings, capability filters, or health overlays.

## Policy

No notes yet.

## Providers

- 2026-05-19 - [Streaming Adapter Compatibility](2026-05-19-streaming-adapter-compatibility.md): optional streaming provider interface; open when adding adapters, stream support, or unsupported-stream behavior.

## Auth

No notes yet.

## Testing

No notes yet.

## Operations

- 2026-05-19 - [Backlog Triage State Sync](2026-05-19-backlog-triage-state-sync.md): sync `05-issues/` triage labels with canonical `.ai` state on `main`; open when `continue` would pick the wrong task.

## Performance

- 2026-05-19 - [Streaming Adapter Compatibility](2026-05-19-streaming-adapter-compatibility.md): first emitted stream chunk is the fallback boundary; do not retry after data is flushed.
- 2026-05-19 - [Immutable Registry Snapshots](2026-05-19-immutable-registry-snapshots.md): clone map/slice metadata on snapshot ingress and egress while keeping lookups deterministic and in-memory.
- 2026-05-19 - [Context Processor Timeout Isolation](2026-05-19-context-processor-timeout-isolation.md): hard timeouts do not stop goroutines; deeply clone mutable request state and commit only on successful completion.

## Security

No notes yet.

## Chronological Notes

| Date | Title | Retrieval cues |
| --- | --- | --- |
| 2026-05-19 | [Streaming Adapter Compatibility](2026-05-19-streaming-adapter-compatibility.md) | provider, streaming, SSE, optional interface, first token boundary |
| 2026-05-19 | [Immutable Registry Snapshots](2026-05-19-immutable-registry-snapshots.md) | registry, snapshot, map clone, slice clone, capability filter, health overlay |
| 2026-05-19 | [Backlog Triage State Sync](2026-05-19-backlog-triage-state-sync.md) | issue state, triage labels, .ai/tasks.json, next task drift |
| 2026-05-19 | [Context Processor Timeout Isolation](2026-05-19-context-processor-timeout-isolation.md) | contextproc, hard timeout, goroutine, race, nested metadata/tools clone |
