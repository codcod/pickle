package install

// T-009: agent enablement — the --agent contract and the opencode + pi
// scaffolds (install / upgrade / uninstall symmetry).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAgents(t *testing.T) {
	cases := []struct {
		in      string
		want    Agents
		wantErr bool
	}{
		{in: "claude", want: Agents{Claude: true}},
		{in: "opencode", want: Agents{Opencode: true}},
		{in: "pi", want: Agents{Pi: true}},
		{in: "claude,opencode,pi", want: Agents{Claude: true, Opencode: true, Pi: true}},
		{in: " pi , opencode ", want: Agents{Opencode: true, Pi: true}},
		{in: "opencode,opencode", want: Agents{Opencode: true}},
		{in: "zed", wantErr: true},
		{in: "", wantErr: true},
		{in: "claude,", wantErr: true},
		{in: "Claude", wantErr: true}, // names are lowercase, like the flag docs say
	}
	for _, tc := range cases {
		got, err := ParseAgents(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseAgents(%q) = %+v, want error", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseAgents(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseAgents(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestInstallAgentScaffolds(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".",
		Agents: Agents{Claude: true, Opencode: true, Pi: true}}
	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}

	// The reviewer prompt ships with the skill payload itself.
	mustExist(t, filepath.Join(root, ".agents/skills/ticket-flow/resources/docs-readability.prompt.md"))

	// opencode.jsonc is written whole from the embedded template.
	got, err := os.ReadFile(filepath.Join(root, OpencodeConfigFile))
	if err != nil {
		t.Fatalf("opencode.jsonc not written: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(payloadRoot(), filepath.FromSlash(OpencodeAsset)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "docs-readability") || !strings.Contains(string(got), `"git push`) {
		t.Errorf("opencode.jsonc missing the reviewer or the guardrails:\n%s", got)
	}
	if string(got) != string(want) {
		t.Error("opencode.jsonc differs from the embedded template")
	}

	// Pi scaffolds are written from the embedded assets.
	for _, f := range PiScaffolds {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Installed)))
		if err != nil {
			t.Errorf("%s not written: %v", f.Installed, err)
			continue
		}
		asset, err := os.ReadFile(filepath.Join(payloadRoot(), filepath.FromSlash(f.Asset)))
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != string(asset) {
			t.Errorf("%s differs from embedded %s", f.Installed, f.Asset)
		}
	}
}

// TestInstallDefaultLaysNoAgentScaffolds pins the default: claude only, no
// autodetection, nothing opencode/pi-specific on disk.
func TestInstallDefaultLaysNoAgentScaffolds(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".",
		Agents: Agents{Claude: true}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(root, OpencodeConfigFile)); !os.IsNotExist(err) {
		t.Error("opencode.jsonc written without --agent opencode")
	}
	if _, err := os.Lstat(filepath.Join(root, ".pi")); !os.IsNotExist(err) {
		t.Error(".pi/ written without --agent pi")
	}
}

// TestInstallPreservesExistingOpencodeConfig pins decision 5: pickle never
// parses or merges user JSONC — an existing file survives byte-for-byte and the
// template is surfaced as a note instead.
func TestInstallPreservesExistingOpencodeConfig(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	mine := "// mine\n{}\n"
	if err := os.WriteFile(filepath.Join(root, OpencodeConfigFile), []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".",
		Agents: Agents{Opencode: true}})
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, OpencodeConfigFile)); string(b) != mine {
		t.Errorf("existing opencode.jsonc was rewritten:\n%s", b)
	}
	var skipped bool
	for _, s := range res.Skipped {
		if strings.Contains(s, OpencodeConfigFile) {
			skipped = true
		}
	}
	if !skipped {
		t.Error("expected a skip entry for the existing opencode.jsonc")
	}
	var noted bool
	for _, n := range res.Notes {
		if strings.Contains(n, "docs-readability") && strings.Contains(n, "never merges JSONC") {
			noted = true
		}
	}
	if !noted {
		t.Errorf("expected a manual-merge note carrying the template; notes: %q", res.Notes)
	}
}

// TestUpgradeRefreshesPiScaffolds pins decision 6: upgrade probes the
// filesystem — a present (possibly drifted) pi scaffold is refreshed, an absent
// one is not created, and opencode.jsonc is never touched.
func TestUpgradeRefreshesPiScaffolds(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".",
		Agents: Agents{Pi: true, Opencode: true}}); err != nil {
		t.Fatal(err)
	}

	drifted := filepath.Join(root, filepath.FromSlash(PiScaffolds[0].Installed))
	if err := os.WriteFile(drifted, []byte("// drifted"), 0o644); err != nil {
		t.Fatal(err)
	}
	custom := "// customized by the user\n{}\n"
	if err := os.WriteFile(filepath.Join(root, OpencodeConfigFile), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Upgrade(payload, root, "v2"); err != nil {
		t.Fatal(err)
	}

	asset, err := os.ReadFile(filepath.Join(payloadRoot(), filepath.FromSlash(PiScaffolds[0].Asset)))
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(drifted); string(b) != string(asset) {
		t.Errorf("drifted pi scaffold not refreshed by upgrade:\n%s", b)
	}
	if b, _ := os.ReadFile(filepath.Join(root, OpencodeConfigFile)); string(b) != custom {
		t.Errorf("upgrade touched the user-owned opencode.jsonc:\n%s", b)
	}

	// Absent scaffolds stay absent: a claude-only install gains no .pi on upgrade.
	root2 := t.TempDir()
	if _, err := Run(payload, root2, "v1", Options{ProjectName: "demo", ProjectPath: ".",
		Agents: Agents{Claude: true}}); err != nil {
		t.Fatal(err)
	}
	if _, err := Upgrade(payload, root2, "v2"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(root2, ".pi")); !os.IsNotExist(err) {
		t.Error("upgrade created .pi/ that install never laid down")
	}
}

// TestUninstallAgentScaffolds pins the uninstall symmetry: pickle-owned pi
// files go (and their dirs are pruned when empty); a pristine opencode.jsonc
// (still byte-identical to the template) goes; a modified one stays.
func TestUninstallAgentScaffolds(t *testing.T) {
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".",
		Agents: Agents{Claude: true, Opencode: true, Pi: true}}

	// Pristine: everything pickle wrote is removed.
	root := t.TempDir()
	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(payload, root, UninstallOptions{}); err != nil {
		t.Fatal(err)
	}
	for _, f := range PiScaffolds {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(f.Installed))); !os.IsNotExist(err) {
			t.Errorf("%s survived uninstall", f.Installed)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, ".pi")); !os.IsNotExist(err) {
		t.Error(".pi/ not pruned after uninstall")
	}
	if _, err := os.Lstat(filepath.Join(root, OpencodeConfigFile)); !os.IsNotExist(err) {
		t.Error("pristine opencode.jsonc survived uninstall")
	}

	// Modified opencode.jsonc and user pi extensions are the user's: they stay.
	root2 := t.TempDir()
	if _, err := Run(payload, root2, "v1", opts); err != nil {
		t.Fatal(err)
	}
	mine := "// mine\n{}\n"
	if err := os.WriteFile(filepath.Join(root2, OpencodeConfigFile), []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}
	userExt := filepath.Join(root2, filepath.FromSlash(PiExtensionsDir), "my-extension.ts")
	if err := os.WriteFile(userExt, []byte("// user code"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Uninstall(payload, root2, UninstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(root2, OpencodeConfigFile)); string(b) != mine {
		t.Error("modified opencode.jsonc did not survive uninstall")
	}
	var skipNoted bool
	for _, s := range res.Skipped {
		if strings.Contains(s, "user-modified") {
			skipNoted = true
		}
	}
	if !skipNoted {
		t.Errorf("expected a user-modified skip entry; skipped: %q", res.Skipped)
	}
	mustExist(t, userExt) // and its directory with it
}

// TestUninstallAgentScaffoldsDryRun: the dry run lists the agent scaffolds and
// mutates nothing.
func TestUninstallAgentScaffoldsDryRun(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".",
		Agents: Agents{Claude: true, Opencode: true, Pi: true}}); err != nil {
		t.Fatal(err)
	}
	res, err := Uninstall(payload, root, UninstallOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(res.Removed, "\n")
	for _, want := range []string{PiScaffolds[0].Installed, PiScaffolds[1].Installed, OpencodeConfigFile} {
		if !strings.Contains(joined, want) {
			t.Errorf("dry-run did not list %s; removed: %q", want, res.Removed)
		}
	}
	mustExist(t, filepath.Join(root, OpencodeConfigFile))
	for _, f := range PiScaffolds {
		mustExist(t, filepath.Join(root, filepath.FromSlash(f.Installed)))
	}
}
