# Prebuilt `wallop` binaries

CI refreshes these files on every push to `main`.

| File | Copy to |
|------|----------|
| `wallop-windows-amd64.exe` | `bin/wallop.exe` |
| `wallop-darwin-arm64` | `bin/wallop` |
| `wallop-linux-amd64` | `bin/wallop` |

```bash
mkdir -p bin
cp dist/wallop-linux-amd64 bin/wallop && chmod +x bin/wallop
./bin/wallop toc
```
