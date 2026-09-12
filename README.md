# tfp

`tfp` turns `terraform plan` output into something you can actually
review: changes grouped by module and collapsible like a code-folding
tree, a summary of how much changed, and a hotkey to silence the same
noisy attribute change (a label bump, say) across every resource that
carries it — instead of scrolling past it 80 times.

## Install

```sh
go install github.com/gusandrioli/tfp/cmd/tfp@latest
```

Requires Go 1.26+ and a `terraform` binary on `PATH` (only needed for
`tfp plan`; `tfp show` works against an already-generated plan file
without it).

## Quickstart

```sh
cd your-terraform-project
tfp
```

That's it — `tfp` with no arguments runs `terraform plan` and opens the
interactive view. Pass terraform's own flags after `--`:

```sh
tfp -- -var-file=prod.tfvars -target=module.foo
```

Already have a plan file (e.g. from CI)?

```sh
terraform show -json tfplan.binary > plan.json
tfp show plan.json

# or pipe it directly
terraform show -json tfplan.binary | tfp show -
```

## The TUI

```
┌ tfp ─ plan.json ─ 14 to add, 6 to change, 2 to destroy ─────────────────┐
│ Tree (40%)                        │ Detail (60%)                        │
│ ─────────────────────────────────  │ ──────────────────────────────────  │
│ ▾ module.eks_sp_foundation         │ kubernetes_role_binding.elastic_agent│
│   ▾ module.elastic_agent  ~3       │ will be updated in-place             │
│     ~ kubernetes_role_binding.…    │                                      │
│   ▸ module.otel_collector  +2 -1   │  ~ metadata.labels["app.kubernetes   │
│ ▾ module.vpc                        │      .io/version"] = "9.2.2" ->     │
│   + aws_subnet.private[0]          │      "9.2.4"                        │
├─────────────────────────────────────────────────────────────────────────┤
│ 0 filter(s) active            j/k move · tab switch · f/F filter · q quit│
└─────────────────────────────────────────────────────────────────────────┘
```

| Key | Action |
| --- | --- |
| `j`/`k`, `↓`/`↑` | Move the cursor |
| `g` / `G` | Jump to top / bottom |
| `tab` | Switch between the tree and detail pane |
| `enter` / `space` | Collapse/expand a module, or open a resource's detail |
| `f` | Hide this attribute change everywhere it occurs |
| `F` | Hide it, but only for this resource type |
| `s` | Toggle expand: show every attribute of the selected resource, not just its diffs |
| `p` | Toggle the active-filters panel (remove one with `d`) |
| `ctrl+r` | Clear all filters |
| `?` | Full keybinding help |
| `q` | Quit |

Filtering only ever hides diff lines from view — the summary counts and
per-module rollup badges always reflect the real, unfiltered plan.

## Non-interactive use (CI, scripts)

`tfp` skips the TUI automatically when stdout isn't a terminal, or when
`--output` is given explicitly:

```sh
tfp show plan.json --output text   # plain-text render, like the TUI's detail pane
tfp show plan.json --output json   # machine-readable: totals, per-module counts, diffs
```

Exit codes: `0` no changes, `3` changes present (text/json mode only —
lets a script branch without parsing output), `1` a real error, `2` bad
flags/args.

## Documentation

Design and implementation notes live in [`docs/`](docs/):

- [`ARCHITECTURE.md`](docs/ARCHITECTURE.md) — system design, package layout, data flow
- [`DATA_MODEL.md`](docs/DATA_MODEL.md) — domain types and the diff algorithm
- [`CLI.md`](docs/CLI.md) — commands, flags, exit codes
- [`TUI.md`](docs/TUI.md) — screens, keybindings, filter UX
- [`ROADMAP.md`](docs/ROADMAP.md) — phased implementation plan and what's left

## Development

```sh
make build   # bin/tfp
make test
make lint    # requires golangci-lint
```
