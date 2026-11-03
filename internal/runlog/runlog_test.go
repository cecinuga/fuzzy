package runlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fuzzy/internal/fuzzer"
)

func readLog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestResultLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.log")
	log, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	log.Result(fuzzer.Result{Endpoint: "GET /a", Status: 404, Message: "nope", Input: "id=1", Duration: 1500 * time.Microsecond})
	log.Result(fuzzer.Result{Endpoint: "GET /b", Status: 500, Message: "boom"})
	log.Result(fuzzer.Result{Endpoint: "GET /c", Status: fuzzer.TransportError, Message: "refused"})
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(readLog(t, path)), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines: %q", lines)
	}
	for i, wants := range [][]string{
		{"level=INFO", `msg=request`, `endpoint="GET /a"`, "status=404", "duration=1.5ms", "message=nope", `input="id=1"`},
		{"level=ERROR", "status=500", "message=boom"},
		{"level=ERROR", "status=ERR", "message=refused"},
	} {
		for _, want := range wants {
			if !strings.Contains(lines[i], want) {
				t.Errorf("line %d lacks %q: %s", i, want, lines[i])
			}
		}
	}
}

func TestHostileTextCannotForgeALine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.log")
	log, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	log.Result(fuzzer.Result{Endpoint: "GET /a", Status: 200, Message: "x\nlevel=ERROR msg=forged \x1b[31m", Input: "\xff"})
	log.Close()

	got := readLog(t, path)
	if strings.Count(got, "\n") != 1 || strings.Contains(got, "\x1b") {
		t.Errorf("log = %q", got)
	}
}

func TestLogAppends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.log")
	for _, message := range []string{"first", "second"} {
		log, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		log.Info(message, "key", 1)
		if err := log.Close(); err != nil {
			t.Fatal(err)
		}
	}
	got := readLog(t, path)
	if !strings.Contains(got, "msg=first key=1") || !strings.Contains(got, "msg=second key=1") {
		t.Errorf("log = %q", got)
	}
}

func TestLinesReachTheFileBeforeClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.log")
	log, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	log.Warn("early")
	if got := readLog(t, path); !strings.Contains(got, "level=WARN msg=early") {
		t.Errorf("log = %q: a killed run would lose this line", got)
	}
}

func TestNilLoggerDoesNothing(t *testing.T) {
	var log *Logger
	log.Info("x")
	log.Warn("x")
	log.Error("x")
	log.Result(fuzzer.Result{})
	if err := log.Close(); err != nil {
		t.Error(err)
	}
}

func TestOpenFailsOnABadPath(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "no", "dir", "run.log")); err == nil {
		t.Error("want an error")
	}
}

func TestCloseReportsAFailedWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.log")
	log, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	log.file.Close() // every later write fails
	log.Info("lost")
	if err := log.Close(); err == nil {
		t.Error("want the failed write's error")
	}
	if err := log.Close(); err == nil {
		t.Error("a second Close must still report it")
	}
}
