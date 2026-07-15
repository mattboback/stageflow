# Documentation

Each subject has one canonical owner. Component READMEs cover only local build, test, and implementation details.

## Start Here

| Need                                | Document                                              |
| ----------------------------------- | ----------------------------------------------------- |
| Product overview and quick start    | [Repository README](../README.md)                     |
| System design and trust boundaries  | [Architecture](architecture.md)                       |
| CLI workflows                       | [CLI guide](cli.md)                                   |
| Local deployment and self-hosting   | [Self-hosting](self-hosting.md)                       |
| Environment variables               | [Configuration reference](reference/configuration.md) |
| Product principles                  | [Product](product.md)                                 |
| Interface language                  | [Design](design.md)                                   |
| Engineering decisions and tradeoffs | [Case study](case-study.md)                           |
| Hosted-demo data handling           | [Privacy](privacy.md)                                 |
| Reviewer-oriented repository tour   | [Code tour](code-tour.md)                             |
| Contributor workflow                | [Contributing](../CONTRIBUTING.md)                    |
| Security policy                     | [Security](../SECURITY.md)                            |
| Dependency security exceptions      | [Exceptions](dependency-exceptions.md)                |

## Reference

The [CLI command reference](reference/cli/stageflow/stageflow.md) is generated from Cobra. Do not edit those pages by hand. Regenerate them after command or flag changes:

```bash
go run ./clients/cli docs --out-dir docs/reference/cli/stageflow
```

JSON Schemas under `libs/contracts/*/schema/` are the field-level contract reference. Their committed fixtures are executable examples.

## Ownership

- Update exact configuration defaults in `.env.example` and then the configuration reference.
- Update exact web tokens in `clients/web/app/styles/instrument.css` or `report.css`; keep the design guide semantic.
- Keep hosted production automation out of this repository's self-hosting instructions. The public service shares application code, not deployment topology.
- Keep temporary plans, handoffs, QA output, and generated critiques in ignored output paths or issue/PR history, not the product documentation tree.
- Keep hosted-demo data-handling claims in `privacy.md` aligned with the configured object-storage lifecycle.
