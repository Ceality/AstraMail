package version_test

import (
	"strings"
	"testing"

	"Astramail/internal/version"
)

func TestVersionIsSemver(t *testing.T) {
	t.Parallel()

	// 版本号硬编码在源码中，必须是三段式语义化版本。
	parts := strings.Split(version.Version, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3-part semantic version, got %q", version.Version)
	}

	for _, part := range parts {
		if part == "" {
			t.Fatalf("version %q contains an empty segment", version.Version)
		}

		for _, r := range part {
			if r < '0' || r > '9' {
				t.Fatalf("version %q contains a non-numeric segment %q", version.Version, part)
			}
		}
	}
}

func TestVersionIsNotEmpty(t *testing.T) {
	t.Parallel()

	if strings.TrimSpace(version.Version) == "" {
		t.Fatal("version must not be empty")
	}
}

func TestStringContainsVersionAndCommit(t *testing.T) {
	t.Parallel()

	got := version.String()
	if !strings.HasPrefix(got, version.Version+"+") {
		t.Errorf("expected %q to start with %q+", got, version.Version)
	}

	if !strings.Contains(got, version.Commit) {
		t.Errorf("expected %q to contain commit %q", got, version.Commit)
	}
}

func TestLongContainsBuildDate(t *testing.T) {
	t.Parallel()

	got := version.Long()
	if !strings.Contains(got, version.Version) {
		t.Errorf("expected %q to contain version %q", got, version.Version)
	}

	if !strings.Contains(got, version.BuildDate) {
		t.Errorf("expected %q to contain build date %q", got, version.BuildDate)
	}
}
