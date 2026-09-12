# dist/

This folder is a build output, not a committed binary cache.

- `make build` writes `bin/wallop` (and optionally copies here if you ask CI to).
- Without Go, run `python3 scripts/bootstrap.py --download` to fetch the `nightly` (or `v*`) asset from GitHub Releases into `bin/`.

Do not commit `wallop-*` binaries. They bloat clone and have no useful diff.
