## Purpose and priorities

This repository is a TypeScript Node.js Discord bot using discord.js.

When making decisions, prioritize:
1. Correctness and preservation of existing behavior.
2. Readability and simplicity.
3. Small, focused, modular code with clear responsibilities.
4. Consistency with existing repository conventions.
5. Reuse where it provides a meaningful architectural benefit.

Modularity does not mean maximizing the number of files, interfaces, or layers. Prefer the simplest design that maintains clear boundaries.

## Agent workflow

- Before editing, inspect the relevant files, adjacent modules, existing patterns, dependencies, and available project scripts.
- Follow established repository conventions unless the task explicitly requires changing them.
- Ask clarifying questions before making architectural decisions when requirements are ambiguous. Do not silently choose a major design.
- Keep changes strictly within the requested scope. Do not perform unrelated refactors, formatting changes, dependency upgrades, or cleanups.
- Preserve existing behavior and contracts unless a breaking change is explicitly authorized.
- Do not introduce speculative infrastructure for future requirements.
- After editing, run relevant tests, TypeScript checks, and linting when available.
- If a check fails, stop and ask before attempting additional fixes. Report the command, failure, and whether it appears related to the change; do not silently repair it.
- In the final report, explain each meaningful change and its design rationale, the verification performed and results, and any remaining risks or questions.

## Architecture and modules

- Organize code in layers, with domain-specific subfolders where useful (for example, commands, events, services, and repositories).
- Give each file one primary responsibility. There is no fixed line limit; split files when doing so improves clarity.
- Keep Discord command and event handlers thin: parse and validate inputs, call application services, and produce Discord responses.
- Put business logic in focused services or domain modules, not in Discord interaction handlers.
- Prefer functions. Use classes when state, lifecycle, or a concrete object-oriented requirement makes them clearer.
- Use a centralized dependency container to register and assemble application dependencies. Prefer resolving dependencies at composition boundaries and passing them explicitly to functions; avoid hidden service lookups throughout business logic.
- Define interfaces or contracts at meaningful boundaries, such as external APIs, persistence, or interchangeable implementations. Do not add interfaces merely to mirror a single implementation.
- Design for reuse where a real boundary is evident, but do not add factories, wrappers, or abstraction layers without a clear benefit.
- Extract repeated logic when it represents a stable shared responsibility. Small, incidental duplication is preferable to a misleading abstraction.
- Do not add a persistence layer until the application needs persistence. When it does, keep storage-specific details behind an appropriate boundary.
- Avoid circular dependencies and unnecessary coupling.

## TypeScript and code style

- Use strict but pragmatic TypeScript. Prefer type inference when the type is obvious; explicitly type public boundaries and values where clarity or safety benefits.
- Avoid `any` when practical. If unavoidable, keep its scope narrow and justify it.
- Use camelCase filenames and function/variable names; use PascalCase for types, interfaces, and classes.
- Use named or default exports as appropriate to the module and existing codebase.
- Prefer descriptive names, early returns, guard clauses, and flat control flow.
- Write self-documenting code. Comments should explain non-obvious reasoning, constraints, or workarounds—not restate what the code does.
- Prefer native APIs and existing dependencies, but add a small, proven dependency when it materially simplifies the implementation. Check what is already installed first.
- Do not introduce clever patterns or generalized utilities that make ordinary behavior harder to follow.

## Discord.js conventions

- Keep command and event handlers focused on Discord-specific concerns and delegation.
- Handle expected command failures with friendly ephemeral responses when the interaction supports them; do not expose stack traces, tokens, or internal error details.
- Account for interaction acknowledgement state before replying, deferring, or following up.
- Keep Discord-specific objects out of core business logic where practical; pass only the data and capabilities needed.
- Use explicit `await` and controlled concurrency. Only run operations concurrently when they are independent and safe to parallelize.
- Handle Discord API errors and rate limits using existing library behavior and application-level boundaries rather than ad hoc retries scattered across handlers.

## Errors and results

- Use a hybrid error model:
  - Expected operational or domain failures return typed success/error results.
  - Unexpected bugs and exceptional failures may throw and are handled at command, event, or task boundaries.
- Keep result types consistent across the codebase; reuse an existing result pattern before introducing a new one.
- Centralize classification and handling of unexpected errors.
- Log failures with appropriate context. Do not silently swallow exceptions.
- Isolate failures where safe so one command or event does not unnecessarily stop the bot. Do not continue after an unrecoverable or corrupted application state.
- Avoid catch-and-log-then-rethrow patterns that duplicate logs unless the catch adds useful context or performs necessary cleanup.

## Centralized logging and critical alerts

- Use one centralized structured logger rather than calling `console` throughout application modules.
- Support consistent levels, timestamps, machine-readable fields, and relevant error stack traces.
- Include useful context when available: command or event name, user ID, guild ID, and correlation/request ID.
- Never log secrets, tokens, credentials, or unnecessarily sensitive payloads. Redact sensitive fields.
- Send normal application logs to the console.
- Send only critical or unhandled errors to a configured Discord alert channel through a logging/alerting integration, not through ad hoc calls in business modules.
- Make alert delivery best-effort: prevent alert loops, bound or rate-limit repeated alerts, and do not let a failed alert trigger further alerts.
- Keep alert-channel configuration centralized and optional so logging still works if Discord alerts are unavailable.
- Avoid reporting expected user-facing validation errors as critical incidents.

## Configuration and external inputs

- Access environment variables through a centralized, typed configuration module.
- Validate required configuration once at startup and fail clearly on invalid required values.
- Reuse existing validation tools. Introduce a validation dependency only when it meaningfully improves safety or simplicity.
- Validate untrusted external input at application boundaries.

## Testing and verification

- Test meaningful business logic and important edge cases; avoid tests that merely restate trivial implementation details.
- Use the testing framework already present in the repository. Do not introduce a competing framework without approval.
- Favor focused tests with controlled dependencies over tests that require a live Discord connection.
- Run relevant tests, TypeScript checks, and linting when available after changes.
- If any verification fails, stop, report the failure, and ask before making further fixes.

## Change discipline

- Implement only what the task requires.
- Prefer minimal, reviewable diffs.
- Preserve compatibility unless explicitly told otherwise.
- Do not rewrite neighboring code merely to make it match a preferred style.
- Ask before making architectural decisions with material tradeoffs or introducing significant new infrastructure.
- Explain meaningful design decisions and tradeoffs in the completion report.

## Code style preferences

- Prefer readable, concise code over clever or overly abstract solutions. If a straightforward approach works, use it.
- Use verbose, descriptive variable names. Avoid single-letter names, abbreviations, or terse shorthand. A name should clearly communicate what the variable holds without needing surrounding context to decode it.
