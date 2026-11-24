package ui

import "fmt"

// Voice is the central copy deck for human-facing lines. Keep it warm and
// precise: a little yeast-budding personality on starts and successes, and
// absolutely no jokes inside error blocks — errors are plain and actionable.
var Voice = struct {
	// Command starts — gentle budding metaphor, one line each.
	Up             func(machines int) string
	Down           func() string
	Destroy        func() string
	Snapshot       func() string
	Restore        func() string
	Doctor         func() string
	Provision      func() string
	Pull           func() string
	Clean          func() string
	DeleteSnapshot func() string
	Fork           func() string
	// Successes.
	UpDone   func(machines int) string
	DownDone func() string
	SnapDone func() string
	RestDone func() string
	DoctorOK func() string
	// Idle / home flavor.
	HomeTag  string
	OneMore  string
	Sleeping func() string
	// Errors: pointer line only — the card carries the detail.
	Explain func(code string) string
}{
	Up: func(machines int) string {
		if machines == 0 {
			return "Budding machines…"
		}
		if machines == 1 {
			return "Budding 1 machine…"
		}
		return fmt.Sprintf("Budding %d machines…", machines)
	},
	Down:           func() string { return "Putting machines to sleep…" },
	Destroy:        func() string { return "Composting machines…" },
	Snapshot:       func() string { return "Jarring a snapshot…" },
	Restore:        func() string { return "Waking the culture back up…" },
	Doctor:         func() string { return "Checking the culture…" },
	Provision:      func() string { return "Feeding machines their packages…" },
	Pull:           func() string { return "Fetching base images…" },
	Clean:          func() string { return "Wiping down the jar…" },
	DeleteSnapshot: func() string { return "Pouring out a jar…" },
	Fork:           func() string { return "Budding off siblings…" },

	UpDone: func(machines int) string {
		if machines == 1 {
			return "Machine is humming."
		}
		return fmt.Sprintf("%d machines humming.", machines)
	},
	DownDone: func() string { return "All quiet in the jar." },
	SnapDone: func() string { return "Sealed and labeled." },
	RestDone: func() string { return "Culture revived." },
	DoctorOK: func() string { return "Culture looks healthy." },

	HomeTag:  "local machines, alive in seconds",
	OneMore:  "One more? One more!",
	Sleeping: func() string { return "Everything's resting." },

	Explain: func(code string) string {
		return fmt.Sprintf("run `yeast explain %s` for the full fix path", code)
	},
}
