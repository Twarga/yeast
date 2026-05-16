package output

import (
	"fmt"
	"io"
	"sort"
	"yeast/internal/app"
)

func RenderHuman(w io.Writer, command string, data any) error {
	switch value := data.(type) {
	case string:
		_, err := fmt.Fprintln(w, value)
		return err
	case app.InitResult:
		if _, err := fmt.Fprintf(w, "Created %s\n", value.ConfigPath); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Created %s\n", value.MetadataPath); err != nil {
			return err
		}
		_, err := fmt.Fprintf(w, "Project ID: %s\n", value.ProjectID)
		return err
	case app.PullResult:
		if value.List {
			for _, image := range value.Images {
				if _, err := fmt.Fprintln(w, image); err != nil {
					return err
				}
			}
			return nil
		}
		if _, err := fmt.Fprintf(w, "Pulled %s\n", value.ImageName); err != nil {
			return err
		}
		_, err := fmt.Fprintf(w, "Saved %s\n", value.ImagePath)
		return err
	case app.DoctorResult:
		for _, check := range value.Checks {
			if _, err := fmt.Fprintf(w, "[%s] %s: %s\n", check.Status, check.Name, check.Details); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "Blockers: %d\n", value.Blockers); err != nil {
			return err
		}
		_, err := fmt.Fprintf(w, "Warnings: %d\n", value.Warnings)
		return err
	case app.UpResult:
		for _, instance := range value.Instances {
			if _, err := fmt.Fprintf(w, "Started %s (%s)\n", instance.Name, instance.SSHAddress); err != nil {
				return err
			}
		}
		return nil
	case app.StatusResult:
		instances := append([]app.StatusInstanceResult(nil), value.Instances...)
		sort.Slice(instances, func(i, j int) bool { return instances[i].Name < instances[j].Name })
		for _, instance := range instances {
			if instance.SSHPort > 0 {
				if _, err := fmt.Fprintf(w, "%s\t%s\t127.0.0.1:%d\n", instance.Name, instance.Status, instance.SSHPort); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "%s\t%s\n", instance.Name, instance.Status); err != nil {
				return err
			}
		}
		return nil
	case app.DownResult:
		for _, instance := range value.Instances {
			if _, err := fmt.Fprintf(w, "%s\t%s\n", instance.Name, instance.Status); err != nil {
				return err
			}
		}
		return nil
	case app.DestroyResult:
		for _, instance := range value.Instances {
			if _, err := fmt.Fprintf(w, "%s\t%s\n", instance.Name, instance.Status); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported human render type for %s: %T", command, data)
	}
}
