package doctor

import (
	"encoding/json"
	"regexp"
	"strings"

	"efctl/pkg/container"
)

const versionFields = `{"Client":{{json .Client.Version}},"Server":{{if .Server}}{{json .Server.Version}}{{else}}""{{end}}}`
const dockerInfoFields = `{"OS":{{json .OSType}},"Architecture":{{json .Architecture}},"OperatingSystem":{{json .OperatingSystem}}}`
const podmanInfoFields = `{"Version":{{json .Version.Version}},"OS":{{json .Host.OS}},"Architecture":{{json .Host.Arch}},"Remote":{{json .Host.ServiceIsRemote}}}`
const containerFields = `{"State":{{json .State.Status}},"Running":{{json .State.Running}},"ExitCode":{{json .State.ExitCode}},"Reference":{{json .Config.Image}},"ID":{{json .Image}}}`
const imageFields = `{"ID":{{json .Id}},"OS":{{json .Os}},"Architecture":{{json .Architecture}},"Digests":{{json .RepoDigests}},"Volumes":{{len .Config.Volumes}}}`
const podmanImageFields = `{"ID":{{json .ID}},"OS":{{json .Os}},"Architecture":{{json .Architecture}},"Digests":{{json .RepoDigests}},"Volumes":{{len .Config.Volumes}}}`

type imageMetadata struct {
	ID, OS, Architecture string
	Digests              []string
	Volumes              *int
}

func decodeDiagnostic(r CommandResult, target any) string {
	if r.Reason != "" {
		return r.Reason
	}
	if len(r.Stdout) > captureLimit {
		return "output exceeded capture limit"
	}
	if json.Unmarshal(r.Stdout, target) != nil {
		return "malformed selected metadata"
	}
	return ""
}

func gatherRuntimeServer(p probes, engine string) RuntimeServerInfo {
	source := "runtime " + engine
	info := newRuntimeServerInfo(source)
	identity := p.evidence(source, engine, "--version")
	podman := engine == "podman" || strings.Contains(strings.ToLower(identity.Value), "podman")
	if podman {
		source += " (Podman-compatible engine)"
	}
	applyClientIdentity(&info, source, engine, podman, identity)
	applyStructuredVersions(p, &info, source, engine)
	applyServerInfo(p, &info, source, engine, podman)
	return info
}

func newRuntimeServerInfo(source string) RuntimeServerInfo {
	return RuntimeServerInfo{
		Client:       unavailable(source, "client version unavailable"),
		Server:       unavailable(source, "server version unavailable"),
		OS:           unavailable(source, "server OS unavailable"),
		Architecture: unavailable(source, "server architecture unavailable"),
		Boundary:     unavailable(source, "remote/VM boundary not established"),
	}
}

func applyClientIdentity(info *RuntimeServerInfo, source, engine string, podman bool, identity Evidence) {
	if identity.Reason != "" {
		return
	}
	versionEngine := engine
	if podman {
		versionEngine = "podman"
	}
	info.Client = fieldEvidence(source+" client --version", parseContainerVersion(versionEngine, identity.Value))
}

func applyStructuredVersions(p probes, info *RuntimeServerInfo, source, engine string) {
	var versions struct{ Client, Server string }
	versionResult := p.command(engine, "version", "--format", versionFields)
	reason := decodeDiagnostic(versionResult, &versions)
	if reason == "" {
		info.Client = fieldEvidence(source+" client", versions.Client)
		info.Server = fieldEvidence(source+" server", versions.Server)
		return
	}
	// Docker can return selected client data with a nonzero status when the
	// server is unreachable. Do not discard that independently available data.
	applyPartialClient(info, source, versionResult, reason)
	info.Server = unavailable(source+" server", reason)
}

func applyPartialClient(info *RuntimeServerInfo, source string, versionResult CommandResult, reason string) {
	var versions struct{ Client, Server string }
	if !strings.Contains(versionResult.Reason, "limit") && json.Unmarshal(versionResult.Stdout, &versions) == nil && versions.Client != "" {
		info.Client = available(source+" client", versions.Client)
	}
	if info.Client.Value == "" {
		info.Client = unavailable(source+" client", reason)
	}
}

type serverMetadata struct {
	Version, OS, Architecture, OperatingSystem string
	Remote                                     bool
}

func applyServerInfo(p probes, info *RuntimeServerInfo, source, engine string, podman bool) {
	format := dockerInfoFields
	if podman {
		format = podmanInfoFields
	}
	var metadata serverMetadata
	reason := decodeDiagnostic(p.command(engine, "info", "--format", format), &metadata)
	if reason != "" {
		info.OS = unavailable(source+" server", reason)
		info.Architecture = info.OS
		return
	}
	info.OS = fieldEvidence(source+" server", metadata.OS)
	info.Architecture = fieldEvidence(source+" server", metadata.Architecture)
	if podman && metadata.Version != "" {
		info.Server = available(source+" engine info", metadata.Version)
	}
	info.Boundary = serverBoundary(source, metadata, info.Boundary)
}

func serverBoundary(source string, metadata serverMetadata, current Evidence) Evidence {
	if metadata.Remote {
		return available(source, "engine reports remote service")
	}
	if strings.Contains(strings.ToLower(metadata.OperatingSystem), "docker desktop") {
		return available(source, "Docker Desktop server; VM boundary possible")
	}
	return current
}

var immutableImage = regexp.MustCompile(`^(sha256:)?[a-fA-F0-9]{64}$`)

func unavailableImage(source, reason string) ImageInfo {
	value := unavailable(source, reason)
	return ImageInfo{State: value, ExitCode: value, Reference: value, ID: value, OS: value, Architecture: value, Digests: value}
}

type managedInspect struct {
	State, Reference, ID string
	Running              bool
	ExitCode             *int
}

type imageFallback struct {
	source string
	target string
	info   ImageInfo
}

func gatherImage(p probes, engine string) (ImageInfo, bool, bool) {
	source := "managed container actual image"
	metadata, reason := inspectManagedContainer(p, engine)
	if reason != "" {
		return imageForAbsentContainer(p, engine, source, metadata, reason)
	}
	info, target, stop := applyContainerIdentity(source, metadata)
	if stop {
		return info, metadata.Running, false
	}
	return finishImageInspect(p, engine, source, target, info, metadata)
}

func inspectManagedContainer(p probes, engine string) (managedInspect, string) {
	var metadata managedInspect
	reason := decodeDiagnostic(p.command(engine, "container", "inspect", "--format", containerFields, container.ContainerSuiPlayground), &metadata)
	return metadata, reason
}

func imageForAbsentContainer(p probes, engine, source string, metadata managedInspect, reason string) (ImageInfo, bool, bool) {
	// Confirm absence independently; failed inspect alone can mean an unreachable server.
	fallback, ok := absentContainerFallback(p, engine, reason)
	if !ok {
		return unavailableImage(source, reason), false, false
	}
	return finishImageInspect(p, engine, fallback.source, fallback.target, fallback.info, metadata)
}

func finishImageInspect(p probes, engine, source, target string, info ImageInfo, metadata managedInspect) (ImageInfo, bool, bool) {
	image, reason := inspectSelectedImage(p, engine, target)
	if reason != "" {
		return applyImageInspectFailure(info, source, reason), metadata.Running, false
	}
	info, safe := applyImageMetadata(info, source, metadata.ID, image)
	return info, metadata.Running, safe
}

func absentContainerFallback(p probes, engine, reason string) (imageFallback, bool) {
	listing := p.command(engine, "container", "ls", "--all", "--filter", "name=^"+container.ContainerSuiPlayground+"$", "--format", "{{.Names}}")
	if listing.Reason != "" || strings.TrimSpace(string(listing.Stdout)) != "" {
		return imageFallback{}, false
	}
	source := "configured local image fallback (not proven startup image)"
	info := unavailableImage(source, "managed container absent")
	target := container.ImageSuiDev
	info.Reference = available(source, target)
	return imageFallback{source: source, target: target, info: info}, true
}

func applyContainerIdentity(source string, metadata managedInspect) (ImageInfo, string, bool) {
	info := ImageInfo{
		State:     fieldEvidence("managed container recorded state", metadata.State),
		ExitCode:  unavailable("managed container recorded exit code (not Sui process status)", "not recorded"),
		Reference: fieldEvidence(source, metadata.Reference),
	}
	if metadata.ExitCode != nil {
		info.ExitCode = available("managed container recorded exit code (not Sui process status)", formatExitCode(*metadata.ExitCode))
	}
	if !immutableImage.MatchString(metadata.ID) {
		info.ID = unavailable(source, "immutable image ID not reported")
		return info, metadata.ID, true
	}
	info.ID = available(source, metadata.ID)
	return info, metadata.ID, false
}

func inspectSelectedImage(p probes, engine, target string) (imageMetadata, string) {
	format := imageFields
	if engine == "podman" {
		format = podmanImageFields
	}
	var image imageMetadata
	reason := decodeDiagnostic(p.command(engine, "image", "inspect", "--format", format, target), &image)
	if reason == "" || engine != "docker" || p.ctx.Err() != nil {
		return image, reason
	}
	// Podman wrappers accept Docker commands but expose an ID struct field;
	// their template alias for .Id does not apply inside the json function.
	image = imageMetadata{}
	reason = decodeDiagnostic(p.command(engine, "image", "inspect", "--format", podmanImageFields, target), &image)
	return image, reason
}

func applyImageInspectFailure(info ImageInfo, source, reason string) ImageInfo {
	failure := unavailable(source, reason)
	if info.ID.Value == "" {
		info.ID = failure
	}
	info.OS = failure
	info.Architecture = failure
	info.Digests = failure
	return info
}

func applyImageMetadata(info ImageInfo, source, containerID string, image imageMetadata) (ImageInfo, bool) {
	if !immutableImage.MatchString(image.ID) || (containerID != "" && image.ID != containerID) {
		info.ID = unavailable(source, "image identity unavailable or inconsistent")
		return info, false
	}
	info.ID = available(source, image.ID)
	info.OS = fieldEvidence(source, image.OS)
	info.Architecture = fieldEvidence(source, image.Architecture)
	info.Digests = recordedDigests(source, image.Digests)
	return info, image.Volumes != nil && *image.Volumes == 0
}

func recordedDigests(source string, digests []string) Evidence {
	if len(digests) == 0 {
		return available(source, "not recorded")
	}
	return available(source, strings.Join(digests, ", "))
}
