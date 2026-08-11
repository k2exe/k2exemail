# K2EXEmail Code Review Instructions

K2EXEmail is a native Winlink email client written in Go using Fyne.

Review changed code for correctness first. Avoid requesting unrelated refactors.

## Reliability and Mail Safety

- Offline mailbox operations must remain durable.
- Never lose queued Outbox mail because of connection or application failure.
- Watch for stale message snapshots overwriting newer message state.
- Check asynchronous mailbox mutations for ordering bugs, races, and lost updates.
- Internal UI bookkeeping must not impersonate a new user action.
- Errors should preserve useful context and must not silently discard mail.

## Go Concurrency

- Blocking work must stay off the Fyne UI thread.
- UI changes originating from background goroutines must use the appropriate Fyne UI dispatch mechanism.
- Review shared mutable state for data races.
- Check cancellation paths for goroutine, connection, modem, or session leaks.
- Prefer context.Context for cancellable operations.
- Do not introduce unlimited retries.

## Winlink and Transport Architecture

- Do not invent undocumented Winlink protocol behavior.
- Before important protocol or architecture decisions, inspect current wl2k-go and Pat source, tests, specifications, or reference implementations; do not rely only on README files.
- Prefer existing behavior from wl2k-go or Pat when applicable.
- Do not reimplement a Winlink protocol when a suitable proven implementation already exists.
- Winlink message exchange should remain transport-independent.
- Keep transport-specific behavior out of unrelated UI and mailbox code.
- Prefer standard interfaces such as net.Conn, io.Reader, and io.Writer where practical.
- Treat AREDN as a distinct network type; do not assume Internet access.

## Radio Safety

- Never key a transmitter merely because the application starts.
- Transmit only as part of an intentional user operation.
- Cancellation or failure must stop transmit activity.
- Do not add unlimited RF retries or repeated PTT attempts after errors.
- Do not assume a configured frequency is legal or appropriate.

## Security and Diagnostics

- Never log passwords, secure-login responses, authentication secrets, or other credentials.
- Diagnostics must redact secrets.
- Review persistent configuration changes for migration compatibility.
- Do not destroy valid older configuration during migration.

## Fyne UI

- Do not use Fyne widgets as the application data model.
- Preserve desktop-first behavior.
- Watch for stale asynchronous callbacks updating a view that is no longer active.
- Internal list selection/restoration must not cause duplicate message-open side effects.
- Keep expensive or blocking work away from the UI thread.

## Cross-Platform

Changed behavior should work on Windows, Linux, and macOS unless intentionally platform-specific.

Keep OS-specific serial, filesystem, or device behavior isolated.

## Tests

New behavior and bug fixes should include focused regression tests when practical.

Normal CI must not require radio hardware. Use mocks, simulated connections, modem/TNC simulations, and separate hardware-specific tests when needed.

Pay particular attention to:

- race-detector failures
- asynchronous selection changes
- slow or failed storage operations
- dropped connections
- cancellation
- filesystem persistence
- transport mocks
- message state preservation
