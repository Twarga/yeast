package app

import "fmt"

// ExplainResult is the offline knowledge card for a typed error code.
type ExplainResult struct {
	Code     string   `json:"code"`
	Title    string   `json:"title"`
	Cause    string   `json:"cause"`
	Diagnose []string `json:"diagnose"`
	Fix      []string `json:"fix"`
}

// explainTable covers every typed ErrorCode the engine emits.
var explainTable = map[ErrorCode]ExplainResult{
	ErrorCodeInvalidArgument: {
		Code:  "invalid_argument",
		Title: "Bad argument or config value",
		Cause: "A flag, argument, or yeast.yaml field has a value Yeast cannot use.",
		Diagnose: []string{
			"re-read the failing command's usage: yeast <command> --help",
			"validate yeast.yaml indentation and instance names",
			"re-run with --json to see error.details for the exact field",
		},
		Fix: []string{
			"correct the flagged argument or field and retry",
			"compare against a fresh template: yeast init --templates",
		},
	},
	ErrorCodeNotFound: {
		Code:  "not_found",
		Title: "Resource not found",
		Cause: "The project, instance, snapshot, or image Yeast looked for does not exist.",
		Diagnose: []string{
			"list what exists: yeast status (machines), yeast snapshots <instance>",
			"check you are inside the right project directory",
		},
		Fix: []string{
			"create the project first: yeast init",
			"bring machines up: yeast up",
			"fix the name — instance names are exact",
		},
	},
	ErrorCodeConflict: {
		Code:  "conflict",
		Title: "Resource already exists or is in use",
		Cause: "The target already exists, is already running, or is locked by another operation.",
		Diagnose: []string{
			"inspect current state: yeast status",
			"check for a stale lock if a previous run was killed",
		},
		Fix: []string{
			"stop it first: yeast down — or destroy it: yeast destroy",
			"choose a different name for snapshots/projects",
			"if a lock is stale after a crash: yeast clean",
		},
	},
	ErrorCodePrecondition: {
		Code:  "failed_precondition",
		Title: "Host precondition failed",
		Cause: "The host lacks something Yeast requires — most often usable KVM access.",
		Diagnose: []string{
			"run: yeast doctor — blockers are listed with fixes",
			"check /dev/kvm exists and is readable+writable: ls -l /dev/kvm",
		},
		Fix: []string{
			"apply guided fixes: yeast doctor --fix --yes",
			"KVM missing: enable virtualization in BIOS/UEFI, then load kvm_intel/kvm_amd",
			"WSL2: ensure nested virtualization is enabled (.wslconfig: nestedVirtualization=true)",
			"permission denied: add your user to the kvm group, then re-login",
		},
	},
	ErrorCodeTimeout: {
		Code:  "timeout",
		Title: "Operation timed out",
		Cause: "A step ran past its deadline — usually SSH readiness or a boot wait.",
		Diagnose: []string{
			"watch the console: yeast logs <instance>",
			"verify the host is not CPU/IO starved while several VMs boot",
		},
		Fix: []string{
			"retry — first boots can be slow on cold image caches",
			"re-pull the base image if it may be corrupted: yeast pull",
			"boot without provisioning to isolate: yeast up --no-provision",
		},
	},
	ErrorCodeRuntime: {
		Code:  "runtime_error",
		Title: "QEMU runtime failure",
		Cause: "QEMU exited or the runtime could not complete an operation.",
		Diagnose: []string{
			"read the VM log: yeast logs <instance>",
			"run yeast doctor to confirm the QEMU install is healthy",
		},
		Fix: []string{
			"re-run the command — transient races self-heal",
			"check disk space for the runtime dir (df -h ~/.yeast)",
			"clean half-dead state after a crash: yeast clean",
		},
	},
	ErrorCodeProvisioning: {
		Code:  "provisioning_failed",
		Title: "Provisioning failed",
		Cause: "Package installation or a provision step inside the guest failed.",
		Diagnose: []string{
			"read the provision log path printed in yeast status",
			"check guest network egress: yeast exec <instance> -- ping -c1 archive.ubuntu.com",
		},
		Fix: []string{
			"fix the package/script error, then retry: yeast provision",
			"force a re-run: yeast up --reprovision",
		},
	},
	ErrorCodeGuest: {
		Code:  "guest_error",
		Title: "Command failed inside the guest",
		Cause: "A command Yeast ran over SSH exited non-zero.",
		Diagnose: []string{
			"re-run it yourself to see output: yeast exec <instance> -- <cmd>",
			"check guest logs: yeast logs <instance>",
		},
		Fix: []string{
			"fix the command or guest state and retry",
			"open an interactive shell: yeast ssh <instance>",
		},
	},
	ErrorCodeInternal: {
		Code:  "internal",
		Title: "Internal engine error",
		Cause: "Yeast hit a state it did not expect — likely a bug.",
		Diagnose: []string{
			"re-run with --json and keep the full envelope",
			"note the yeast version: yeast version",
		},
		Fix: []string{
			"retry once — then file an issue with the JSON output attached",
		},
	},
	ErrorCodeUnknown: {
		Code:  "unknown",
		Title: "Unclassified error",
		Cause: "The error did not carry a typed code.",
		Diagnose: []string{
			"re-run with --json — the error.message field holds the raw detail",
		},
		Fix: []string{
			"if it repeats, run yeast doctor and file an issue with both outputs",
		},
	},
}

// Explain returns the knowledge card for a typed error code.
func Explain(code string) (ExplainResult, error) {
	if r, ok := explainTable[ErrorCode(code)]; ok {
		return r, nil
	}
	return ExplainResult{}, WrapError(ErrorCodeNotFound,
		fmt.Sprintf("no explanation for error code %q — try `yeast explain` without args to list codes", code), nil)
}

// ExplainCodes lists every error code with an explanation entry.
func ExplainCodes() []string {
	codes := make([]string, 0, len(explainTable))
	for c := range explainTable {
		codes = append(codes, string(c))
	}
	return codes
}
