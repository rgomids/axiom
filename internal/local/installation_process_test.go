package local

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
)

const crashProjectID = "123e4567-e89b-42d3-a456-426614174000"

func TestInstallationCrashBoundaryRequiresRecovery(t *testing.T) {
	for _, point := range []string{"stage", "published"} {
		t.Run(point, func(t *testing.T) {
			directory := privateTestRoot(t)
			source := filepath.Join(directory, "source")
			if err := os.Mkdir(source, 0o700); err != nil {
				t.Fatal(err)
			}
			value, issues := project.New(project.State{SchemaVersion: 1, ID: crashProjectID, Slug: "sample", Name: "Sample"})
			if len(issues) != 0 {
				t.Fatal(issues)
			}
			wire, issues := manifest.Encode(value)
			if len(issues) != 0 {
				t.Fatal(issues)
			}
			if err := os.WriteFile(filepath.Join(source, manifestName), wire, 0o600); err != nil {
				t.Fatal(err)
			}
			state := filepath.Join(directory, "state")
			store, err := NewInstallationStore(state)
			if err != nil {
				t.Fatal(err)
			}
			command := exec.Command(os.Args[0], "-test.run=^TestInstallationProcessHelper$")
			command.Env = append(os.Environ(), "AXIOM_INSTALL_HELPER="+point, "AXIOM_INSTALL_SOURCE="+source, "AXIOM_INSTALL_STATE="+state)
			output, err := command.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = command.Process.Kill()
				_ = command.Wait()
			}()
			line, err := bufio.NewReader(output).ReadString('\n')
			if err != nil || strings.TrimSpace(line) != point {
				t.Fatalf("crash barrier not reached: %q, %v", line, err)
			}
			if err := command.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = command.Wait()
			result := store.Reopen(context.Background(), source)
			if result.Status != InstallationFailed || result.Category != "recovery_required" {
				t.Fatalf("crash at %s accepted: %+v", point, result)
			}
			record := filepath.Join(state, "projects", crashProjectID, "installation.json")
			_, err = os.Stat(record)
			if point == "stage" && !os.IsNotExist(err) {
				t.Fatalf("precommit record visible: %v", err)
			}
			if point == "published" && err != nil {
				t.Fatalf("postcommit record missing: %v", err)
			}
		})
	}
}

func TestInstallationProcessHelper(t *testing.T) {
	point := os.Getenv("AXIOM_INSTALL_HELPER")
	if point == "" {
		return
	}
	store, err := NewInstallationStore(os.Getenv("AXIOM_INSTALL_STATE"))
	if err != nil {
		t.Fatal(err)
	}
	store.beforePublication = func() {
		if point == "stage" {
			fmt.Fprintln(os.Stdout, point)
			select {}
		}
	}
	store.afterPublication = func() {
		if point == "published" {
			fmt.Fprintln(os.Stdout, point)
			select {}
		}
	}
	result := store.Install(context.Background(), os.Getenv("AXIOM_INSTALL_SOURCE"))
	if result.Status != InstallationApplied {
		t.Fatal(result)
	}
}
