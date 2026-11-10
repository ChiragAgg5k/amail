package mail

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testScriptSources(t *testing.T) (string, map[string]string) {
	t.Helper()
	const scriptDirectory = "script"
	const preludeName = "prelude.js"
	entries, err := os.ReadDir(scriptDirectory)
	if err != nil {
		t.Fatalf("read %s: %v", scriptDirectory, err)
	}
	var prelude string
	scripts := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".js" {
			continue
		}
		source, err := os.ReadFile(filepath.Join(scriptDirectory, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		if entry.Name() == preludeName {
			prelude = string(source)
			continue
		}
		scripts[entry.Name()] = string(source)
	}
	if prelude == "" {
		t.Fatalf("%s/%s is missing or empty", scriptDirectory, preludeName)
	}
	if len(scripts) == 0 {
		t.Fatalf("no scripts besides %s in %s", preludeName, scriptDirectory)
	}
	return prelude, scripts
}

func TestScriptStructure(t *testing.T) {
	const preludeName = "prelude.js"
	const runDeclaration = "function run("
	const mainDeclaration = "function main("
	prelude, scripts := testScriptSources(t)
	if count := strings.Count(prelude, runDeclaration); count != 1 {
		t.Errorf("%s declares %q %d times, want exactly 1", preludeName, runDeclaration, count)
	}
	for name, source := range scripts {
		if !strings.Contains(source, mainDeclaration) {
			t.Errorf("%s does not declare %q", name, mainDeclaration)
		}
		if strings.Contains(source, runDeclaration) {
			t.Errorf("%s declares %q; only %s may define the entry point", name, runDeclaration, preludeName)
		}
	}
}

func TestScriptsCompile(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("osacompile is only available on macOS")
	}
	compiler, err := exec.LookPath("osacompile")
	if err != nil {
		t.Skip("osacompile not found in PATH")
	}
	prelude, scripts := testScriptSources(t)
	for name, source := range scripts {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			directory := t.TempDir()
			path := filepath.Join(directory, name)
			if err := os.WriteFile(path, []byte(prelude+"\n"+source), 0o600); err != nil {
				t.Fatalf("write %s: %v", path, err)
			}
			output := filepath.Join(directory, strings.TrimSuffix(name, ".js")+".scpt")
			combined, err := exec.Command(compiler, "-l", "JavaScript", "-o", output, path).CombinedOutput()
			if err != nil {
				t.Fatalf("osacompile %s: %v\n%s", name, err, combined)
			}
		})
	}
}
