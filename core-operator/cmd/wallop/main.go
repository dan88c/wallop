package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dan88c/wallop/core-operator/pkg/audit"
	"github.com/dan88c/wallop/core-operator/pkg/config"
	"github.com/dan88c/wallop/core-operator/pkg/doctor"
	"github.com/dan88c/wallop/core-operator/pkg/gateway"
	"github.com/dan88c/wallop/core-operator/pkg/masker"
	"github.com/dan88c/wallop/core-operator/pkg/pointer"
)

const (
	Version  = "2.0.000"
	maxDepth = 3
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "wallop")
	stateDir := filepath.Join(homeDir, ".local", "state")
	shareDir := filepath.Join(homeDir, ".local", "share", "wallop")

	switch os.Args[1] {
	case "check":
		handleCheck(configDir, os.Args[2:])
	case "run":
		handleRun(configDir, stateDir, os.Args[2:])
	case "escalate":
		handleEscalate(configDir, shareDir, os.Args[2:])
	case "doctor":
		if err := doctor.RunDiagnostics(configDir, stateDir); err != nil {
			os.Exit(1)
		}
	case "init":
		cfg, err := config.LoadConfig()
		if err != nil || cfg == nil {
			fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
			os.Exit(1)
		}
		if err := config.RunInteractiveSetup(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
			os.Exit(1)
		}
	case "register":
		handleRegister(configDir, os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("wallop %s\n", Version)
	default:
		printUsage()
		os.Exit(2)
	}
}

func handleCheck(configDir string, args []string) {
	query := ""
	if len(args) > 0 {
		query = strings.ToLower(args[0])
	}

	tools, err := pointer.LoadAndValidatePointers(configDir)
	if err != nil {
		enc := json.NewEncoder(os.Stderr)
		_ = enc.Encode(map[string]string{"error": err.Error()})
		os.Exit(1)
	}

	var filtered []pointer.ToolPointer
	for _, t := range tools {
		if query == "" || strings.Contains(strings.ToLower(t.Name), query) || strings.Contains(strings.ToLower(t.Description), query) {
			filtered = append(filtered, t)
		}
	}

	output, _ := json.MarshalIndent(filtered, "", "  ")
	fmt.Println(string(output))
}

func handleRun(configDir, stateDir string, args []string) {
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	toolName := runCmd.String("tool", "", "Name of target tool")
	payloadFile := runCmd.String("payload", "", "Path to payload JSON")
	_ = runCmd.Parse(args)

	if *toolName == "" || *payloadFile == "" {
		fmt.Fprintln(os.Stderr, "Error: --tool and --payload are required")
		os.Exit(2)
	}

	depth := 0
	if val := os.Getenv("WALLOP_DEPTH"); val != "" {
		depth, _ = strconv.Atoi(val)
	}
	if depth >= maxDepth {
		fmt.Fprintf(os.Stderr, "Error: recursion limit exceeded (depth %d)\n", depth)
		os.Exit(1)
	}

	pointerPath, err := pointer.PointerFile(configDir, *toolName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
	data, err := os.ReadFile(pointerPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: tool '%s' not found\n", *toolName)
		os.Exit(3)
	}

	var tool pointer.ToolPointer
	if err := json.Unmarshal(data, &tool); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid pointer JSON: %v\n", err)
		os.Exit(1)
	}

	payloadData, err := os.ReadFile(*payloadFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot read payload: %v\n", err)
		os.Exit(4)
	}

	if err := pointer.ValidatePayload(tool.Schema, payloadData); err != nil {
		fmt.Fprintf(os.Stderr, "Guard validation failed: %v\n", err)
		os.Exit(1)
	}

	startTime := time.Now()
	resolvedBin, err := pointer.ResolveCommandTarget(tool.Command, tool.Workdir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(3)
	}

	spawn := pointer.BuildSpawnCommand(resolvedBin, tool.Args)
	cmd := exec.Command(spawn.Binary, spawn.Args...)
	if tool.Workdir != "" {
		cmd.Dir = tool.Workdir
	}
	cmd.Stdin = bytes.NewReader(payloadData)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("WALLOP_DEPTH=%d", depth+1))

	runErr := cmd.Run()
	duration := time.Since(startTime).Milliseconds()

	exitCode := 0
	errMsg := ""
	if runErr != nil {
		if exitError, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
		errMsg = runErr.Error()
	}

	_ = audit.LogEvent(stateDir, audit.ExecutionEvent{
		Tool:       *toolName,
		Status:     map[bool]string{true: "success", false: "failure"}[exitCode == 0],
		ExitCode:   exitCode,
		DurationMs: duration,
		Depth:      depth,
		Error:      errMsg,
	})

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func handleEscalate(configDir, shareDir string, args []string) {
	escCmd := flag.NewFlagSet("escalate", flag.ExitOnError)
	reason := escCmd.String("reason", "", "Escalation reason")
	contextFile := escCmd.String("context", "", "Path to context file")
	_ = escCmd.Parse(args)

	if *reason == "" || *contextFile == "" {
		fmt.Fprintln(os.Stderr, "Error: --reason and --context are required")
		os.Exit(2)
	}

	cfg, err := config.LoadConfig()
	if err != nil || cfg == nil || !cfg.Confirmed {
		fmt.Fprintln(os.Stderr, "Error: local de-identification model not confirmed. Run 'wallop init' first.")
		os.Exit(1)
	}

	rawContext, err := os.ReadFile(*contextFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read context: %v\n", err)
		os.Exit(1)
	}

	sanitizedReason, err := masker.MaskContext(cfg, *reason)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Privacy masking error: %v\n", err)
		os.Exit(1)
	}
	sanitizedContext, err := masker.MaskContext(cfg, string(rawContext))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Privacy masking error: %v\n", err)
		os.Exit(1)
	}

	secrets, err := gateway.LoadSecrets(configDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		secrets = map[string]string{}
	}
	apiToken := os.Getenv("CLOUD_DEVELOPER_TOKEN")
	if token, ok := secrets["CLOUD_DEVELOPER_TOKEN"]; ok && apiToken == "" {
		apiToken = token
	}
	apiURL := os.Getenv("CLOUD_DEVELOPER_URL")
	if url, ok := secrets["CLOUD_DEVELOPER_URL"]; ok && apiURL == "" {
		apiURL = url
	}

	if apiToken == "" || apiURL == "" {
		fmt.Fprintln(os.Stderr, "Error: CLOUD_DEVELOPER_TOKEN or CLOUD_DEVELOPER_URL missing")
		os.Exit(1)
	}

	scriptContent, err := gateway.SynthesizeTool(apiURL, apiToken, sanitizedReason, sanitizedContext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cloud synthesis failed: %v\n", err)
		os.Exit(1)
	}

	scriptPath, err := gateway.WriteSynthesizedToolSafely(shareDir, scriptContent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Atomic commit failed: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(map[string]string{"status": "synthesized", "script_path": scriptPath})
}

func handleRegister(configDir string, args []string) {
	regCmd := flag.NewFlagSet("register", flag.ExitOnError)
	name := regCmd.String("name", "", "Tool identifier")
	desc := regCmd.String("desc", "", "Tool description")
	cmdPath := regCmd.String("cmd", "", "Path to executable target")
	workdir := regCmd.String("workdir", "", "Working directory")
	probe := regCmd.String("probe", "", "Healthcheck probe arg (optional, e.g. --version)")
	_ = regCmd.Parse(args)

	if *name == "" || *cmdPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --name and --cmd are required")
		os.Exit(2)
	}

	var probes []string
	if *probe != "" {
		probes = append(probes, *probe)
	}

	tool := pointer.ToolPointer{
		Name:        *name,
		Description: *desc,
		Command:     *cmdPath,
		Workdir:     *workdir,
		Healthcheck: probes,
	}

	if err := pointer.RegisterTool(configDir, tool); err != nil {
		fmt.Fprintf(os.Stderr, "Registration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully registered tool '%s'\n", *name)
}

func printUsage() {
	fmt.Println("Usage: wallop <check|run|escalate|doctor|init|register|version> [flags]")
}
