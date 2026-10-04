package main

import (
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantArgs []string
		check    func(*options) bool
	}{
		{"command only", []string{"watch"}, []string{"watch"}, nil},
		{"flags before command", []string{"--no-notify", "--watch", "/w", "watch"}, []string{"watch"},
			func(o *options) bool { return o.overrides.NoNotify && o.overrides.WatchDir == "/w" }},
		{"flags after command", []string{"watch", "--dry-run", "--ingest", "api"}, []string{"watch"},
			func(o *options) bool { return o.overrides.DryRun && o.overrides.Ingest == "api" }},
		{"equals syntax", []string{"--settle-delay=5s", "watch"}, []string{"watch"},
			func(o *options) bool { return o.overrides.SettleDelay == "5s" }},
		{"subcommand", []string{"--config", "/c.toml", "service", "install"}, []string{"service", "install"},
			func(o *options) bool { return o.configPath == "/c.toml" }},
		{"long version flag", []string{"--version"}, nil,
			func(o *options) bool { return o.showVersion }},
		{"short version flag", []string{"-v"}, nil,
			func(o *options) bool { return o.showVersion }},
		{"default config path", []string{"watch"}, []string{"watch"},
			func(o *options) bool { return strings.HasSuffix(o.configPath, "paperflow/config.toml") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, args, err := parseArgs(tt.args)
			if err != nil {
				t.Fatalf("parseArgs: %v", err)
			}
			if strings.Join(args, " ") != strings.Join(tt.wantArgs, " ") {
				t.Errorf("args = %q, want %q", args, tt.wantArgs)
			}
			if tt.check != nil && !tt.check(opts) {
				t.Errorf("unexpected options: %+v", opts)
			}
		})
	}
}

func TestParseArgsHelp(t *testing.T) {
	for _, arg := range []string{"--help", "-h"} {
		if _, _, err := parseArgs([]string{arg}); !errors.Is(err, flag.ErrHelp) {
			t.Errorf("parseArgs(%s) error = %v, want flag.ErrHelp", arg, err)
		}
	}
}

func TestParseArgsRejectsUnknownFlag(t *testing.T) {
	if _, _, err := parseArgs([]string{"--wacth", "/w", "watch"}); err == nil {
		t.Error("parseArgs should reject unknown flags")
	}
}

func TestPrintUsageListsFlags(t *testing.T) {
	opts, _, err := parseArgs(nil)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	printUsage(&b, opts.flags)

	for _, want := range []string{"--watch <dir>", "--paperless-token-file <file>", "--dry-run", "service install"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("usage should mention %q:\n%s", want, b.String())
		}
	}
	if strings.Contains(b.String(), "  --v ") {
		t.Error("usage should not list the -v alias")
	}
}

func TestTomlList(t *testing.T) {
	if got := tomlList([]string{"pdf", "~$*"}); got != `["pdf", "~$*"]` {
		t.Errorf("tomlList = %s", got)
	}
}
