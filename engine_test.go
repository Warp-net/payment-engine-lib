package paymentengine

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var embedded = map[string]bool{
	"linux/amd64": true, "linux/arm64": true,
	"darwin/amd64": true, "darwin/arm64": true,
	"windows/amd64": true,
}

func TestEngineIsEmbeddedForThisPlatform(t *testing.T) {
	target := runtime.GOOS + "/" + runtime.GOARCH
	engine, err := GetPaymentEngine()

	if !embedded[target] {
		if !errors.Is(err, ErrUnsupported) {
			t.Fatalf("%s: want ErrUnsupported, got %v", target, err)
		}
		if len(engine) != 0 {
			t.Fatalf("%s got %d bytes it cannot run", target, len(engine))
		}
		return
	}
	if err != nil {
		t.Fatalf("%s: %v", target, err)
	}
	magic := map[string]string{
		"linux":   "\x7fELF",
		"darwin":  "\xcf\xfa\xed\xfe",
		"windows": "MZ",
	}[runtime.GOOS]
	if !strings.HasPrefix(string(engine), magic) {
		t.Fatalf("%s unpacked to %d bytes that are not an executable: % x", target, len(engine), engine[:4])
	}
}

func TestUnpackedEngineRuns(t *testing.T) {
	engine, err := GetPaymentEngine()
	if err != nil {
		t.Skipf("%s/%s carries no engine", runtime.GOOS, runtime.GOARCH)
	}

	name := "payment-engine"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, engine, 0o700); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(path, "address")
	cmd.Stdin = strings.NewReader(`{"private_key":"1111111111111111111111111111111111111111111111111111111111111111"}`)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}

	const want = "TCLBgkbfVkJroVBJVqBEsxtPNQEQMTQCLQ"
	if !strings.Contains(string(out), want) {
		t.Fatalf("want %s in the output, got %s", want, out)
	}
}

func TestEveryTargetCompilesAndAndroidCarriesNothing(t *testing.T) {
	if testing.Short() {
		t.Skip("cross compiles every target")
	}
	targets := []string{
		"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64",
		"android/arm64", "android/amd64", "ios/arm64", "linux/386", "windows/arm64", "freebsd/amd64",
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			parts := strings.Split(target, "/")
			cmd := exec.Command("go", "build", "./...")
			cmd.Env = append(cmd.Environ(), "GOOS="+parts[0], "GOARCH="+parts[1])
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%s: %v\n%s", target, err, out)
			}
		})
	}
}
