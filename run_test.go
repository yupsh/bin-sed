package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestRun(t *testing.T) {
	cases := []struct {
		files      map[string]string
		name       string
		version    string
		stdin      string
		wantOut    string
		wantErrSub string
		args       []string
		wantCode   int
	}{
		{
			name:    "substitute first occurrence from stdin",
			args:    []string{"sed", "s/a/x/"},
			stdin:   "aaa\nbanana\n",
			wantOut: "xaa\nbxnana\n",
		},
		{
			name:    "global substitution",
			args:    []string{"sed", "s/a/x/g"},
			stdin:   "banana\n",
			wantOut: "bxnxnx\n",
		},
		{
			name:    "single file source",
			args:    []string{"sed", "s/world/universe/", "/in.txt"},
			files:   map[string]string{"/in.txt": "hello world\n"},
			wantOut: "hello universe\n",
		},
		{
			name: "multiple file sources",
			args: []string{"sed", "s/foo/bar/", "/a.txt", "/b.txt"},
			files: map[string]string{
				"/a.txt": "foo one\n",
				"/b.txt": "foo two\n",
			},
			wantOut: "bar one\nbar two\n",
		},
		{
			name:    "version flag reports injected version",
			version: "1.2.3",
			args:    []string{"sed", "--version"},
			wantOut: "sed version 1.2.3\n",
		},
		{
			name:       "missing script errors",
			args:       []string{"sed"},
			wantCode:   1,
			wantErrSub: "sed: missing SCRIPT operand",
		},
		{
			name:       "unknown flag errors",
			args:       []string{"sed", "--nope", "s/a/b/"},
			wantCode:   1,
			wantErrSub: "sed:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			for path, content := range tc.files {
				if err := afero.WriteFile(fs, path, []byte(content), 0o644); err != nil {
					t.Fatalf("write fixture %s: %v", path, err)
				}
			}

			var out, errOut bytes.Buffer
			code := run(tc.version, tc.args, strings.NewReader(tc.stdin), &out, &errOut, fs)

			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d (stderr=%q)", code, tc.wantCode, errOut.String())
			}
			if tc.wantErrSub == "" && out.String() != tc.wantOut {
				t.Fatalf("stdout = %q, want %q", out.String(), tc.wantOut)
			}
			if tc.wantErrSub != "" && !strings.Contains(errOut.String(), tc.wantErrSub) {
				t.Fatalf("stderr = %q, want substring %q", errOut.String(), tc.wantErrSub)
			}
		})
	}
}

func Test_main(t *testing.T) {
	origExit, origRun := osExit, runCLI
	t.Cleanup(func() { osExit, runCLI = origExit, origRun })

	gotCode := -1
	osExit = func(code int) { gotCode = code }
	runCLI = func(string, []string, io.Reader, io.Writer, io.Writer, afero.Fs) int { return 7 }

	main()

	if gotCode != 7 {
		t.Fatalf("main propagated exit code %d, want 7", gotCode)
	}
}
