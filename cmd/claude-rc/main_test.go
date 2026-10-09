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

type fakeServices struct {
	calls []call
	err   error
}

func (f *fakeServices) Add(out io.Writer, repo, name string, dryRun bool) error {
	f.calls = append(f.calls, call{"Add", []any{repo, name, dryRun}})
	fmt.Fprintln(out, "fake add")
	return f.err
}

func (f *fakeServices) Remove(out io.Writer, name string, dryRun bool) error {
	f.calls = append(f.calls, call{"Remove", []any{name, dryRun}})
	fmt.Fprintln(out, "fake remove")
	return f.err
}

func (f *fakeServices) Status(out io.Writer) error {
	f.calls = append(f.calls, call{"Status", nil})
	fmt.Fprintln(out, "fake status")
	return f.err
}

// Each command, with and without its flags, calls the matching service method
// once with the right arguments, and the service's output reaches stdout.
func TestRunCallsServices(t *testing.T) {
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
			svc := &fakeServices{}
			var out, errOut bytes.Buffer

			if err := run(tt.args, &out, &errOut, svc); err != nil {
				t.Fatalf("run: %v", err)
			}
			if !reflect.DeepEqual(svc.calls, []call{tt.want}) {
				t.Errorf("calls = %v, want %v", svc.calls, []call{tt.want})
			}
			if out.String() != tt.wantOut {
				t.Errorf("out = %q, want %q", out.String(), tt.wantOut)
			}
		})
	}
}

// Missing, unknown or extra arguments and unknown flags print the usage to
// stderr, return an error, and never reach the services.
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
			svc := &fakeServices{}
			var out, errOut bytes.Buffer

			if err := run(args, &out, &errOut, svc); err == nil {
				t.Fatal("expected an error")
			}
			if len(svc.calls) != 0 {
				t.Errorf("services called: %v", svc.calls)
			}
			if !strings.Contains(errOut.String(), "usage:") {
				t.Errorf("usage not shown, stderr = %q", errOut.String())
			}
		})
	}
}

// When a service method fails, run returns that same error instead of hiding it.
func TestRunReturnsServiceError(t *testing.T) {
	boom := errors.New("boom")
	svc := &fakeServices{err: boom}

	err := run([]string{"status"}, io.Discard, io.Discard, svc)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}
