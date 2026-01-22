# Codebase Redundancy & Deep Scan Report

**Date**: 2026-01-22
**Status**: Analysis Complete
**Scope**: Server (Go), Web (React), Plugins

## Executive Summary
A deep scan of the codebase reveals distinct areas of redundancy, primarily due to recent refactoring efforts (Introduction of AI features) that have not yet fully deprecated older modules. The most significant redundancy exists in the Backend AI implementation and Schedule Recurrence logic. The frontend exhibits typical utility fragmentation.

## 1. Backend Redundancy

### 1.1 AI Implementation (Critical)
**Conflict**: `server/ai` vs `plugin/ai`

*   **Logic**:
    *   `server/ai`: Contains a legacy, OpenAI-specific implementation (`provider.go`). It uses hardcoded models and older environment variable patterns.
    *   `plugin/ai`: Contains the modern, multi-provider implementation (OpenAI, DeepSeek, Ollama) using `langchaingo` and proper profile-based configuration.
*   **Usage**: The active `server/runner/embedding` uses `plugin/ai`.
*   **Recommendation**:
    *   **Action**: Delete `server/ai` entirely.
    *   **Verify**: Ensure no other parts of `server/` (e.g., `server/router`) import `server/ai`.

### 1.2 Schedule Recurrence (High)
**Conflict**: `server/scheduler/rrule` vs `plugin/ai/schedule`

*   **Logic**:
    *   `server/scheduler/rrule`: A robust, RFC 5545 compliant parser/generator.
    *   `plugin/ai/schedule`: Re-implements a "Simplified" recurrence logic (`recurrence.go`) with custom JSON serialization and generation logic.
*   **Risk**: The system now has two "languages" for recurring events. The AI agent speaks the "Simplified" language, while the core system likely expects or should normally use RFC standard.
*   **Recommendation**:
    *   **Consolidate**: Refactor `plugin/ai/schedule` to use `server/scheduler/rrule`.
    *   **Action**: Parse Natural Language into RFC 5545 string using `rrule` package, rather than a custom JSON struct. This makes the stored data standard-compliant and interoperable with other calendar systems (iCal).

## 2. Frontend Redundancy

### 2.1 Utility Fragmentation (Medium)
**Conflict**: `web/src/helpers` vs `web/src/utils` vs `web/src/lib`

*   **Observation**:
    *   `web/src/helpers/utils.ts`: Contains generic DOM/Browser helpers (`absolutifyLink`, `downloadFileFromUrl`).
    *   `web/src/lib/utils.ts`: Contains Tailwind helpers (`cn`).
    *   `web/src/utils/`: Contains domain-specific utils.
*   **Recommendation**:
    *   **Consolidate**: Move generic helpers from `helpers/utils.ts` to `web/src/utils/browser.ts` or similar.
    *   **Rename**: `web/src/helpers` is ambiguous. If it contains data constants, name it `constants`. If it contains business logic, move to `lib` or `utils`.

## 3. Structural Observations

### 3.1 "Agent" Logic Location
*   **Observation**: Agent logic is currently nested in `plugin/ai/agent`.
*   **Status**: This seems correct for the new architecture, but care must be taken that `server/runner` does not implement conflicting task logic. Currently `server/runner` handles background tasks (OCR, Embedding), while `plugin/ai/agent` handles interactive agents. This separation is clean *if* maintained strictly.

## 4. Next Steps

1.  **Immediate**: Remove `server/ai` to prevent confusion.
2.  **Refactor**: standardizing Recurrence Rule in `plugin/ai/schedule`.
3.  **Cleanup**: Move frontend helpers to `utils`.
