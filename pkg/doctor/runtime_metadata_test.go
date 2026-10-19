package doctor

import (
	"context"
	"strings"
	"testing"
)

func TestRuntimeMetadata(t *testing.T) {
	for _, engine := range []string{"docker", "podman", "wrapper"} {
		p := probes{ctx: context.Background(), run: func(_ context.Context, name string, args ...string) CommandResult {
			switch args[0] {
			case "--version":
				if engine != "docker" {
					return CommandResult{Stdout: []byte("podman version 6.1")}
				}
				return CommandResult{Stdout: []byte("Docker version 29")}
			case "version":
				return CommandResult{Stdout: []byte(`{"Client":"29","Server":"28"}`)}
			case "info":
				if engine != "docker" {
					return CommandResult{Stdout: []byte(`{"Version":"6.1","OS":"linux","Architecture":"arm64","Remote":true}`)}
				}
				return CommandResult{Stdout: []byte(`{"OS":"linux","Architecture":"x86_64","OperatingSystem":"Docker Desktop"}`)}
			}
			t.Fatalf("unexpected %s %v", name, args)
			return CommandResult{}
		}}
		got := gatherRuntimeServer(p, "docker")
		if got.Client.Value != "29" || got.OS.Value != "linux" || got.Architecture.Value == "" {
			t.Fatalf("%s: %+v", engine, got)
		}
		if engine != "docker" && !strings.Contains(got.Boundary.Value, "remote") {
			t.Fatalf("boundary: %+v", got)
		}
	}
	for _, result := range []CommandResult{{Reason: "server unavailable"}, {Stdout: []byte(`{"Client":"29"}`)}, {Stdout: []byte(`malformed`)}} {
		p := probes{ctx: context.Background(), run: func(context.Context, string, ...string) CommandResult { return result }}
		got := gatherRuntimeServer(p, "docker")
		if got.Server.Reason == "" || got.Architecture.Reason == "" {
			t.Fatalf("missing reasons: %+v", got)
		}
	}
}

func TestUnreachableServerRetainsClient(t *testing.T) {
	p := probes{ctx: context.Background(), run: func(_ context.Context, _ string, args ...string) CommandResult {
		if args[0] == "--version" {
			return CommandResult{Stdout: []byte("Docker version 29.0, build abc")}
		}
		if args[0] == "version" {
			return CommandResult{Stdout: []byte(`{"Client":"29.0","Server":""}`), Reason: "exit status 1"}
		}
		return CommandResult{Reason: "server unavailable"}
	}}
	got := gatherRuntimeServer(p, "docker")
	if got.Client.Value != "29.0" || got.Server.Reason == "" || got.Architecture.Reason == "" {
		t.Fatalf("client/server conflated: %+v", got)
	}
}

func TestInspectFailureDoesNotInventAbsentContainer(t *testing.T) {
	p := probes{ctx: context.Background(), run: func(_ context.Context, _ string, args ...string) CommandResult {
		if args[0] == "image" {
			t.Fatal("fallback image inspected without confirming container absence")
		}
		return CommandResult{Reason: "server unavailable"}
	}}
	got, _, safe := gatherImage(p, "docker")
	if safe || got.ID.Reason == "" {
		t.Fatal("inspect failure treated as absence")
	}
}

func TestPodmanImageTemplateCompatibility(t *testing.T) {
	for _, engine := range []string{"podman", "docker"} {
		t.Run(engine, func(t *testing.T) {
			id := strings.Repeat("a", 64)
			p := probes{ctx: context.Background(), run: func(_ context.Context, _ string, args ...string) CommandResult {
				if args[0] == "container" {
					return CommandResult{Stdout: []byte(`{"State":"exited","Running":false,"ID":"` + id + `"}`)}
				}
				if args[3] != podmanImageFields {
					return CommandResult{Reason: "exit status 125"}
				}
				return CommandResult{Stdout: []byte(`{"ID":"` + id + `","OS":"linux","Architecture":"amd64","Volumes":0}`)}
			}}
			got, _, safe := gatherImage(p, engine)
			if !safe || got.ID.Value != id || got.OS.Value != "linux" {
				t.Fatalf("Podman template not supported: %+v", got)
			}
		})
	}
}

func TestImageMetadata(t *testing.T) {
	id := "sha256:" + strings.Repeat("a", 64)
	for _, state := range []string{"running", "exited", "missing", "volumes", "noimage"} {
		assertImageMetadata(t, id, state)
	}
}

type imageMetaProbe struct {
	t        *testing.T
	state    string
	id       string
	selected string
}

func (p *imageMetaProbe) run(_ context.Context, _ string, args ...string) CommandResult {
	switch {
	case args[0] == "container" && args[1] == "inspect":
		return p.inspectContainer()
	case args[0] == "container" && args[1] == "ls":
		return CommandResult{}
	case args[0] == "image":
		return p.inspectImage(args)
	default:
		p.t.Fatalf("unexpected mutation %v", args)
		return CommandResult{}
	}
}

func (p *imageMetaProbe) inspectContainer() CommandResult {
	if p.state == "missing" {
		return CommandResult{Reason: "exit status 1"}
	}
	running := "false"
	if p.state == "running" {
		running = "true"
	}
	return CommandResult{Stdout: []byte(`{"State":"` + p.state + `","Running":` + running + `,"ExitCode":132,"Reference":"old-tag","ID":"` + p.id + `"}`)}
}

func (p *imageMetaProbe) inspectImage(args []string) CommandResult {
	p.selected = args[len(args)-1]
	if p.state == "noimage" {
		return CommandResult{Reason: "image unavailable"}
	}
	volumes := "0"
	if p.state == "volumes" {
		volumes = "1"
	}
	return CommandResult{Stdout: []byte(`{"ID":"` + p.id + `","OS":"linux","Architecture":"amd64","Digests":[],"Volumes":` + volumes + `}`)}
}

func assertImageMetadata(t *testing.T, id, state string) {
	t.Helper()
	probe := &imageMetaProbe{t: t, state: state, id: id}
	got, running, safe := gatherImage(probes{ctx: context.Background(), run: probe.run}, "docker")
	if running != (state == "running") {
		t.Fatal("running state lost")
	}
	assertImageSelection(t, state, id, probe.selected, got)
	assertImageSafety(t, state, id, safe, got)
}

func assertImageSelection(t *testing.T, state, id, selected string, got ImageInfo) {
	t.Helper()
	if state == "missing" {
		if selected != "localhost/efctl-sui-dev" || !strings.Contains(got.ID.Source, "fallback") {
			t.Fatalf("fallback: %+v", got)
		}
		return
	}
	if selected != id {
		t.Fatalf("tag selected over actual image: %s", selected)
	}
}

func assertImageSafety(t *testing.T, state, id string, safe bool, got ImageInfo) {
	t.Helper()
	if state == "noimage" {
		if got.OS.Reason == "" || got.ID.Value != id || safe {
			t.Fatal("missing image treated as safe or recorded identity discarded")
		}
		return
	}
	if got.Digests.Value != "not recorded" {
		t.Fatal("digest invented")
	}
	if state == "volumes" && safe {
		t.Fatal("declared volumes allowed")
	}
}
