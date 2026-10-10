package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
)

type call struct {
	method string
	args   []any
}

type fakeManager struct {
	calls []call
	err   error
}

func (f *fakeManager) Add(out io.Writer, repo, name string, dryRun bool) error {
	f.calls = append(f.calls, call{"Add", []any{repo, name, dryRun}})
	fmt.Fprintln(out, "fake add")
	return f.err
}

func (f *fakeManager) Remove(out io.Writer, name string, dryRun bool) error {
	f.calls = append(f.calls, call{"Remove", []any{name, dryRun}})
	fmt.Fprintln(out, "fake remove")
	return f.err
}

func (f *fakeManager) Status(out io.Writer) error {
	f.calls = append(f.calls, call{"Status", nil})
	fmt.Fprintln(out, "fake status")
	return f.err
}

// Each command, with and without its flags, calls the matching manager method
// once with the right arguments, and the manager's output reaches stdout.
func TestRunCallsManager(t *testing.T) {
	tests := []struct {
		args    []string
		want    call
		wantOut string
	}{
		{[]string{"add", "/home/a/utils"}, call{"Add", []any{"/home/a/utils", "", false}}, "fake add\n"},
		{[]string{"add", "-name", "tools", "-dry-run", "/home/a/utils"}, call{"Add", []any{"/home/a/utils", "tools", true}}, "fake add\n"},
		{[]string{"remove", "utils"}, call{"Remove", []any{"utils", false}}, "fake remove\n"},
		{[]string{"remove", "-dry-run", "utils"}, call{"Remove", []any{"utils", true}}, "fake remove\n"},
		{[]string{"status"}, call{"Status", nil}, "fake status\n"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			mgr := &fakeManager{}
			var out, errOut bytes.Buffer

			if err := run(tt.args, &out, &errOut, mgr); err != nil {
				t.Fatalf("run: %v", err)
			}
			if !reflect.DeepEqual(mgr.calls, []call{tt.want}) {
				t.Errorf("calls = %v, want %v", mgr.calls, []call{tt.want})
			}
			if out.String() != tt.wantOut {
				t.Errorf("out = %q, want %q", out.String(), tt.wantOut)
			}
		})
	}
}

// Missing, unknown or extra arguments and unknown flags print the usage to
// stderr, return an error, and never reach the manager.
func TestRunBadArgsShowsUsage(t *testing.T) {
	tests := [][]string{
		{},
		{"start"},
		{"add"},
		{"add", "/a", "/b"},
		{"add", "-nope", "/a"},
		{"remove"},
		{"remove", "a", "b"},
		{"status", "extra"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			mgr := &fakeManager{}
			var out, errOut bytes.Buffer

			if err := run(args, &out, &errOut, mgr); err == nil {
				t.Fatal("expected an error")
			}
			if len(mgr.calls) != 0 {
				t.Errorf("manager called: %v", mgr.calls)
			}
			if !strings.Contains(errOut.String(), "usage:") {
				t.Errorf("usage not shown, stderr = %q", errOut.String())
			}
		})
	}
}

// When a manager method fails, run returns that same error instead of hiding it.
func TestRunReturnsManagerError(t *testing.T) {
	boom := errors.New("boom")
	mgr := &fakeManager{err: boom}

	err := run([]string{"status"}, io.Discard, io.Discard, mgr)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}
