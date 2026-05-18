package provision

import (
	"testing"
	"yeast/internal/config"
)

func TestBuildPlanEmpty(t *testing.T) {
	plan := BuildPlan(config.Instance{Name: "web"}, nil)

	if plan.InstanceName != "web" {
		t.Fatalf("expected instance name web, got %q", plan.InstanceName)
	}
	if !plan.Empty() {
		t.Fatal("expected empty plan")
	}
}

func TestBuildPlanProjectOnly(t *testing.T) {
	plan := BuildPlan(config.Instance{Name: "web"}, &config.ProvisionConfig{
		Packages: []string{"caddy"},
		Files: []config.FileProvision{
			{Source: "./site", Destination: "/srv/site", Permissions: "0644"},
		},
		Shell: []string{"systemctl enable --now caddy"},
	})

	if len(plan.Packages) != 1 || plan.Packages[0].Name != "caddy" || plan.Packages[0].Origin != OriginProject {
		t.Fatalf("unexpected project package steps: %#v", plan.Packages)
	}
	if len(plan.Files) != 1 || plan.Files[0].Source != "./site" || plan.Files[0].Origin != OriginProject {
		t.Fatalf("unexpected project file steps: %#v", plan.Files)
	}
	if len(plan.Shell) != 1 || plan.Shell[0].Command != "systemctl enable --now caddy" || plan.Shell[0].Origin != OriginProject {
		t.Fatalf("unexpected project shell steps: %#v", plan.Shell)
	}
}

func TestBuildPlanInstanceOnly(t *testing.T) {
	plan := BuildPlan(config.Instance{
		Name: "web",
		Provision: &config.ProvisionConfig{
			Packages: []string{"git"},
			Files: []config.FileProvision{
				{Source: "./app.env", Destination: "/etc/app.env"},
			},
			Shell: []string{"touch /tmp/ready"},
		},
	}, nil)

	if len(plan.Packages) != 1 || plan.Packages[0].Name != "git" || plan.Packages[0].Origin != OriginInstance {
		t.Fatalf("unexpected instance package steps: %#v", plan.Packages)
	}
	if len(plan.Files) != 1 || plan.Files[0].Destination != "/etc/app.env" || plan.Files[0].Origin != OriginInstance {
		t.Fatalf("unexpected instance file steps: %#v", plan.Files)
	}
	if len(plan.Shell) != 1 || plan.Shell[0].Command != "touch /tmp/ready" || plan.Shell[0].Origin != OriginInstance {
		t.Fatalf("unexpected instance shell steps: %#v", plan.Shell)
	}
}

func TestBuildPlanMergesProjectThenInstanceOrder(t *testing.T) {
	plan := BuildPlan(config.Instance{
		Name: "web",
		Provision: &config.ProvisionConfig{
			Packages: []string{"git"},
			Files: []config.FileProvision{
				{Source: "./instance-site", Destination: "/srv/site"},
			},
			Shell: []string{"echo instance >> /tmp/provision.log"},
		},
	}, &config.ProvisionConfig{
		Packages: []string{"caddy", "curl"},
		Files: []config.FileProvision{
			{Source: "./project-site", Destination: "/srv/site"},
		},
		Shell: []string{"echo project > /tmp/provision.log"},
	})

	if len(plan.Packages) != 3 {
		t.Fatalf("expected 3 package steps, got %d", len(plan.Packages))
	}
	if plan.Packages[0].Name != "caddy" || plan.Packages[0].Origin != OriginProject {
		t.Fatalf("unexpected first package step: %#v", plan.Packages[0])
	}
	if plan.Packages[1].Name != "curl" || plan.Packages[1].Origin != OriginProject {
		t.Fatalf("unexpected second package step: %#v", plan.Packages[1])
	}
	if plan.Packages[2].Name != "git" || plan.Packages[2].Origin != OriginInstance {
		t.Fatalf("unexpected third package step: %#v", plan.Packages[2])
	}

	if len(plan.Files) != 2 {
		t.Fatalf("expected 2 file steps, got %d", len(plan.Files))
	}
	if plan.Files[0].Source != "./project-site" || plan.Files[0].Origin != OriginProject {
		t.Fatalf("unexpected first file step: %#v", plan.Files[0])
	}
	if plan.Files[1].Source != "./instance-site" || plan.Files[1].Origin != OriginInstance {
		t.Fatalf("unexpected second file step: %#v", plan.Files[1])
	}

	if len(plan.Shell) != 2 {
		t.Fatalf("expected 2 shell steps, got %d", len(plan.Shell))
	}
	if plan.Shell[0].Command != "echo project > /tmp/provision.log" || plan.Shell[0].Origin != OriginProject {
		t.Fatalf("unexpected first shell step: %#v", plan.Shell[0])
	}
	if plan.Shell[1].Command != "echo instance >> /tmp/provision.log" || plan.Shell[1].Origin != OriginInstance {
		t.Fatalf("unexpected second shell step: %#v", plan.Shell[1])
	}
}
