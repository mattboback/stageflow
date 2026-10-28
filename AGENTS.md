# StageFlow agent guide

## Changes and acceptance

- Establish expected behavior from the request, an existing contract, or a confirmed defect before editing. Current implementation, generated output, and agreeing mocks are not independent acceptance evidence.
- Keep the smallest coherent change in the existing owner/module. Avoid unrelated refactors, speculative abstractions, duplicate implementations, and unnecessary dependencies. Verify proposed dependency provenance and version-sensitive APIs against resolved versions and authoritative documentation before use.
- Before deleting or consolidating code or tests, account for distinct current contracts and callers, including runtime/configuration discovery. Similar names or an empty search do not prove redundancy. Preserve concurrent changes and stored data.
- Do not add speculative tests or pursue coverage targets. Add tests only for an explicit request or confirmed regression; prefer integration/E2E at the affected boundary. Run existing relevant checks. Do not weaken assertions, bypass guards, special-case fixtures, swallow failures, or fabricate outcomes to get green. Independently justify and disclose any intended contract/check change.

## Verify the actual pipeline safely

- Trace the affected real caller through intake, queue/orchestration, isolated scanner, report persistence, and CLI gate as applicable. Confirm observable content and status, not merely helper return values, command strings, or file existence. Preserve failed/incomplete coverage as unavailable evidence, never a clean scan.
- For changed guards, establish valid preconditions and the specific rejection reason; unrelated fixture failures do not verify the intended guard. Distinguish mocked browser/API integration from a real backend/scanner round trip. Podman isolation claims require actual isolated runtime evidence, not request snapshots or host-network success.
- Use disposable storage, synthetic credentials, and enforced isolated destinations. Missing mocks, DNS stubs, or services must fail closed, not scan live targets or reach paid providers, accounts, or deployment systems. Keep separately authorized live verification distinct from routine tests.
- Read [CONTRIBUTING.md](CONTRIBUTING.md) for checks/workflow; [docs/architecture.md](docs/architecture.md) for ownership. Report exact commands, outcomes, executed/skipped lanes, and remaining boundaries. Static inspection, build success, runtime integration, and deployment are separate claims; report blocked checks honestly.
