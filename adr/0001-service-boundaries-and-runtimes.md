# ADR 0001: Service boundaries and runtimes

- Status: Accepted for Phase 0
- Date: 2026-11-09

## Context

The project demonstrates a network in Go and parametrized independent bank instances in Java/Spring Boot. User interfaces are explicitly later work.

## Decision

Build the network as a Go modular monolith and the bank core as one Java/Spring Boot product deployed three times with separate identity, configuration, PostgreSQL database, credentials, and storage. Keep the gateway optional and later. Use Compose before Kubernetes.

## Consequences

This preserves institutional isolation without duplicating banking logic, reduces early distributed-systems surface area, and leaves extraction decisions evidence-driven. It requires explicit API boundaries and prevents the network from writing bank tables.
