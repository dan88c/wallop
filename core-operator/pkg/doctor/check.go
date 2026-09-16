package doctor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/dan88c/wallop/core-operator/pkg/config"
	"github.com/dan88c/wallop/core-operator/pkg/pointer"
)

// RunDiagnostics retains original CLI behavior and writes directly to os.Stdout.
func RunDiagnostics(configDir, stateDir string) error {
	return RunDiagnosticsWithWriter(os.Stdout, configDir, stateDir)
}

// RunDiagnosticsWithWriter outputs diagnostics to an arbitrary io.Writer for testing or custom capture.
func RunDiagnosticsWithWriter(w io.Writer, configDir, stateDir string) error {
	fmt.Fprintln(w, "==================================================")
	fmt.Fprintf(w, "Wallop Doctor | OS: %s | Arch: %s | Go: %s\n", runtime.GOOS, runtime.GOARCH, runtime.Version())
	fmt.Fprintln(w, "==================================================")

	failed := false
	fail := func(format string, args ...any) {
		fmt.Fprintf(w, format, args...)
		failed = true
	}

	secretPath := filepath.Join(configDir, "secrets.env")
	if info, err := os.Stat(secretPath); err == nil {
		if pointer.IsStrictSecretPerm(info) {
			fmt.Fprintln(w, "[PASS] secrets.env permissions are securely restricted.")
		} else {
			fail("[FAIL] secrets.env permissions are insecure (%04o). Expected 0600.\n", info.Mode().Perm())
			fmt.Fprintln(w, "       Fix suggestion: Run 'chmod 600 ~/.config/wallop/secrets.env'")
		}
	} else {
		fmt.Fprintln(w, "[WARN] secrets.env not found. 'escalate' will require active environment variables.")
	}

	testDirs := []struct {
		name string
		path string
	}{
		{"State Directory", filepath.Join(stateDir, "wallop")},
		{"Tools Directory", filepath.Join(configDir, "tools")},
	}

	for _, d := range testDirs {
		if err := os.MkdirAll(d.path, 0755); err != nil {
			fail("[FAIL] Cannot create/access %s (%s): %v\n", d.name, d.path, err)
		} else {
			testFile := filepath.Join(d.path, ".doctor_probe")
			if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
				fail("[FAIL] %s is not writable: %v\n", d.name, err)
			} else {
				_ = os.Remove(testFile)
				fmt.Fprintf(w, "[PASS] %s is writable: %s\n", d.name, d.path)
			}
		}
	}

	auxTools := []string{"jq"}
	if runtime.GOOS == "windows" {
		auxTools = []string{"powershell"}
	}
	for _, tool := range auxTools {
		if path, err := exec.LookPath(tool); err == nil {
			fmt.Fprintf(w, "[PASS] CLI dependency '%s' found at: %s\n", tool, path)
		} else {
			fmt.Fprintf(w, "[WARN] CLI dependency '%s' not found in PATH.\n", tool)
		}
	}

	tools, err := pointer.LoadAndValidatePointers(configDir)
	if err != nil {
		fail("[FAIL] Failed to load tools: %v\n", err)
	} else {
		for _, t := range tools {
			if len(t.Healthcheck) == 0 {
				continue
			}
			if err := pointer.RunHealthcheck(t); err != nil {
				fail("[FAIL] Healthcheck failed for '%s': %v\n", t.Name, err)
			} else {
				fmt.Fprintf(w, "[PASS] Healthcheck passed for '%s'\n", t.Name)
			}
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fail("[FAIL] Failed to parse config: %v\n", err)
		fmt.Fprintln(w, "==================================================")
		return fmt.Errorf("doctor found failures")
	}

	fmt.Fprintf(w, "[INFO] Testing local engine endpoint: %s\n", cfg.LocalEndpoint)
	start := time.Now()
	models, err := config.ProbeLocalModels(cfg.LocalEndpoint)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		fail("[FAIL] Inference engine unreachable at %s (%v)\n", cfg.LocalEndpoint, err)
		fmt.Fprintln(w, "       Fix suggestion: Ensure your engine is running (e.g. 'ollama serve' or 'llama-server').")
	} else {
		fmt.Fprintf(w, "[PASS] Inference engine online (Latency: %d ms). Models found: %d\n", latency, len(models))

		if !cfg.Confirmed {
			fmt.Fprintln(w, "[WARN] De-identification model not confirmed. Run 'wallop init' to authorize a model.")
		} else {
			modelReady := false
			for _, m := range models {
				if m == cfg.MaskerModel {
					modelReady = true
					break
				}
			}

			if modelReady {
				fmt.Fprintf(w, "[PASS] Designated de-identification model '%s' is loaded and ready.\n", cfg.MaskerModel)
			} else {
				fail("[FAIL] Designated model '%s' was NOT found in active endpoint models list.\n", cfg.MaskerModel)
				fmt.Fprintf(w, "       Fix suggestion: Download/load the model or run 'wallop init' to reconfigure.\n")
			}
		}
	}

	fmt.Fprintln(w, "==================================================")
	if failed {
		return fmt.Errorf("doctor found failures")
	}
	return nil
}