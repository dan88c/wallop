Harness tests live in `cmds_operator_test.go` (local until the next CLI commit).

Child processes are injected via `runProcess` and `lookPathFn` so tests never spawn a real tool.

Run:

```
cd core-operator && go test ./...
```
