# Example plan

`plan.json` is a hand-authored `terraform show -json` payload — no real
Terraform config behind it, just enough structure to exercise every
feature: nested modules two levels deep, a create/delete/update/replace
mix, a sensitive value, and a computed-until-apply attribute.

It's built around the exact noisy-label scenario from
[`../docs/PLAN.md`](../docs/PLAN.md): the label
`app.kubernetes.io/version` bumps from `9.2.2` to `9.2.4` across 8
resources spread over `module.eks_sp_foundation`'s three child modules.
Seven of those resources have *only* that change; one
(`kubernetes_role_binding.elastic_agent`) has a second, real change
alongside it, so filtering the label doesn't make it disappear.

## Try it

```sh
go build -o bin/tfp ./cmd/tfp

# Interactive TUI (needs a real terminal)
./bin/tfp show examples/plan.json

# Non-interactive
./bin/tfp show examples/plan.json --output text
./bin/tfp show examples/plan.json --output json
```

In the TUI:

- `module.eks_sp_foundation` is open by default; its two nested modules
  start collapsed — `enter` on either to expand it.
- Drill into any of the label-churn resources (`enter` on a resource
  row), move onto the `app.kubernetes.io/version` diff line, and press
  `f`. All 8 occurrences vanish from view at once; the resources that
  had nothing else changed show as `N change(s) filtered` instead of
  disappearing entirely, and the summary counts in the status bar don't
  move — the plan didn't get smaller, you just stopped looking at that
  one attribute.
- `kubernetes_role_binding.elastic_agent` keeps its `role_ref.name`
  line even after filtering — it has more than just the label change.
- `module.rds.aws_db_instance.primary` shows a replace with
  `(known after apply)` on `id`/`arn` alongside a real
  `engine_version` change, to exercise that rendering path.
- Both replace resources (`kubernetes_service.otel_collector` and
  `aws_db_instance.primary`) mark the one attribute that actually forced
  the replacement with `# forces replacement` — the label change on the
  service is real but incidental, so it's *not* flagged, only
  `spec.port[0].port` is.
- `kubernetes_secret.fluentbit_token` shows `(sensitive value)` on both
  sides of its diff.
