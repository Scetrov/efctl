package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	crashStartSlack  = 2 * time.Second
	crashFinishGrace = 30 * time.Second
	crashWindowCap   = 6 * time.Hour
	maxUnscopedKept  = 20
)

var (
	containerIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	pidArgPattern      = regexp.MustCompile(`^COREDUMP_PID=[0-9]{1,7}$`)
	timestampPattern   = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$`)
)

var journalFieldList = []string{
	"COREDUMP_TIMESTAMP",
	"COREDUMP_PID",
	"COREDUMP_SIGNAL",
	"COREDUMP_SIGNAL_NAME",
	"COREDUMP_EXE",
	"COREDUMP_COMM",
	"COREDUMP_CGROUP",
	"COREDUMP_CONTAINER_CMDLINE",
	"COREDUMP_FILENAME",
	"COREDUMP_HOSTNAME",
}

type crashSpan struct{ Since, Until time.Time }

func buildWindow(hints CrashHints, now time.Time) (crashSpan, bool) {
	if missingCrashTime(hints.StartedAt) {
		return crashSpan{}, false
	}
	until := hints.FinishedAt.Add(crashFinishGrace)
	if hints.Running || missingCrashTime(hints.FinishedAt) {
		until = now.UTC()
	}
	since := hints.StartedAt.Add(-crashStartSlack)
	if until.Before(since) {
		until = since
	}
	if until.Sub(since) > crashWindowCap {
		since = until.Add(-crashWindowCap)
	}
	return crashSpan{Since: since.UTC(), Until: until.UTC()}, true
}

func missingCrashTime(t time.Time) bool {
	return t.IsZero() || t.Year() < 1970
}

func loadJournal(ctx context.Context, opts CrashOptions, window crashSpan) ([]parsedCrash, string) {
	cmd, err := journalCommand(window)
	if err != nil {
		return nil, "crash window failed validation"
	}
	res := opts.run(ctx, cmd)
	if res.Reason != "" {
		return nil, crashFailureReason(res)
	}
	records, err := parseJournalExport(res.Stdout)
	if err != nil {
		return nil, "malformed crash journal"
	}
	return records, ""
}

func journalCommand(window crashSpan) (CrashCommand, error) {
	since, err := validatedTimestamp(window.Since)
	if err != nil {
		return CrashCommand{}, err
	}
	until, err := validatedTimestamp(window.Until)
	if err != nil {
		return CrashCommand{}, err
	}
	fields := strings.Join(journalFieldList, ",")
	return crashMetaCommand("journalctl",
		"--no-pager",
		"--output=json",
		"--output-fields="+fields,
		"MESSAGE_ID="+crashMessageID,
		"--since="+since,
		"--until="+until,
	), nil
}

func indexCommand(window crashSpan) (CrashCommand, error) {
	since, err := validatedTimestamp(window.Since)
	if err != nil {
		return CrashCommand{}, err
	}
	until, err := validatedTimestamp(window.Until)
	if err != nil {
		return CrashCommand{}, err
	}
	return crashMetaCommand("coredumpctl", "--no-pager", "--json=short", "--since="+since, "--until="+until, "list"), nil
}

type parsedCrash struct {
	record  CrashRecord
	related bool
}

func parseJournalExport(data []byte) ([]parsedCrash, error) {
	objects, err := decodeJSONObjects(data)
	if err != nil {
		return nil, err
	}
	out := make([]parsedCrash, 0, len(objects))
	for _, obj := range objects {
		parsed, ok := copyJournalObject(obj)
		if ok {
			out = append(out, parsed)
		}
	}
	return out, nil
}

func copyJournalObject(obj map[string]json.RawMessage) (parsedCrash, bool) {
	rec := CrashRecord{
		Timestamp:        parseCrashTime(jsonScalar(obj["COREDUMP_TIMESTAMP"])),
		PID:              parsePID(jsonScalar(obj["COREDUMP_PID"])),
		Signal:           parsePID(jsonScalar(obj["COREDUMP_SIGNAL"])),
		SignalName:       SanitizeDiagnostic(jsonScalar(obj["COREDUMP_SIGNAL_NAME"])),
		Executable:       SanitizeDiagnostic(jsonScalar(obj["COREDUMP_EXE"])),
		Cgroup:           SanitizeDiagnostic(jsonScalar(obj["COREDUMP_CGROUP"])),
		ContainerCmdline: SanitizeDiagnostic(jsonScalar(obj["COREDUMP_CONTAINER_CMDLINE"])),
		CorePresent:      corePresentFromFilename(obj["COREDUMP_FILENAME"]),
		CoreKnown:        hasJSONKey(obj, "COREDUMP_FILENAME"),
	}
	if rec.PID <= 0 || missingCrashTime(rec.Timestamp) {
		return parsedCrash{}, false
	}
	comm := SanitizeDiagnostic(jsonScalar(obj["COREDUMP_COMM"]))
	return parsedCrash{record: rec, related: relatedCrash(rec, comm)}, true
}

func relatedCrash(rec CrashRecord, comm string) bool {
	if rec.ContainerCmdline != "" && rec.ContainerCmdline != "[redacted sensitive diagnostic]" {
		return true
	}
	if strings.EqualFold(filepath.Base(rec.Executable), "sui") {
		return true
	}
	return strings.EqualFold(comm, "sui")
}

func corePresentFromFilename(raw json.RawMessage) bool {
	value := strings.TrimSpace(jsonScalar(raw))
	return value != "" && value != "null"
}

type indexEntry struct {
	PID         int
	CorePresent bool
	CoreKnown   bool
}

func parseCoredumpIndex(data []byte) ([]indexEntry, error) {
	objects, err := decodeJSONObjects(data)
	if err != nil {
		return nil, err
	}
	out := make([]indexEntry, 0, len(objects))
	for _, obj := range objects {
		if entry, ok := copyIndexObject(obj); ok {
			out = append(out, entry)
		}
	}
	return out, nil
}

func copyIndexObject(obj map[string]json.RawMessage) (indexEntry, bool) {
	pid := firstPID(obj, "pid", "PID", "COREDUMP_PID")
	if pid <= 0 {
		return indexEntry{}, false
	}
	present, known := indexCorePresence(obj)
	return indexEntry{PID: pid, CorePresent: present, CoreKnown: known}, true
}

func indexCorePresence(obj map[string]json.RawMessage) (bool, bool) {
	if raw, ok := firstRaw(obj, "corefile", "COREFILE"); ok {
		state := strings.ToLower(jsonScalar(raw))
		return state == "present" || state == "true", true
	}
	if _, ok := firstRaw(obj, "COREDUMP_FILENAME", "coredump_filename"); ok {
		return corePresentFromFilename(obj["COREDUMP_FILENAME"]), true
	}
	return false, false
}

func classifyCrashes(records []parsedCrash, containerID string, window crashSpan) (attributed []CrashRecord, unscoped []CrashRecord) {
	id := strings.ToLower(containerID)
	for _, parsed := range records {
		rec := parsed.record
		if !window.contains(rec.Timestamp) {
			continue
		}
		if cgroupMatches(rec.Cgroup, id) {
			attributed = append(attributed, rec)
			continue
		}
		if parsed.related {
			unscoped = append(unscoped, rec)
		}
	}
	return attributed, unscoped
}

func (w crashSpan) contains(t time.Time) bool {
	if missingCrashTime(t) {
		return false
	}
	t = t.UTC()
	return !t.Before(w.Since) && !t.After(w.Until)
}

func cgroupMatches(cgroup, id string) bool {
	if !containerIDPattern.MatchString(id) {
		return false
	}
	return strings.Contains(strings.ToLower(cgroup), id)
}

func newestCrash(records []CrashRecord) CrashRecord {
	selected := records[0]
	for _, rec := range records[1:] {
		if rec.Timestamp.After(selected.Timestamp) || (rec.Timestamp.Equal(selected.Timestamp) && rec.PID > selected.PID) {
			selected = rec
		}
	}
	return selected
}

func applyCorePresence(ctx context.Context, opts CrashOptions, window crashSpan, selected CrashRecord) (CrashRecord, string) {
	cmd, err := indexCommand(window)
	if err != nil {
		selected.CoreKnown = false
		return selected, "core presence unavailable"
	}
	res := opts.run(ctx, cmd)
	if res.Reason != "" {
		selected.CoreKnown = false
		return selected, "core presence unavailable: " + crashFailureReason(res)
	}
	entries, err := parseCoredumpIndex(res.Stdout)
	if err != nil {
		selected.CoreKnown = false
		return selected, "core presence unavailable: malformed crash index"
	}
	selected.CoreKnown = false
	selected.CorePresent = false
	for _, entry := range entries {
		if entry.PID != selected.PID || !entry.CoreKnown {
			continue
		}
		selected.CoreKnown = true
		if entry.CorePresent {
			selected.CorePresent = true
		}
	}
	return selected, ""
}

func decodeJSONObjects(data []byte) ([]map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}
	if trimmed[0] == '[' {
		var rows []map[string]json.RawMessage
		if json.Unmarshal(trimmed, &rows) != nil {
			return nil, errMalformedCrash
		}
		return rows, nil
	}
	lines := bytes.Split(trimmed, []byte{'\n'})
	rows := make([]map[string]json.RawMessage, 0, len(lines))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var obj map[string]json.RawMessage
		if json.Unmarshal(line, &obj) != nil {
			return nil, errMalformedCrash
		}
		rows = append(rows, obj)
	}
	return rows, nil
}

var errMalformedCrash = errString("malformed crash record")

type errString string

func (e errString) Error() string { return string(e) }

func jsonScalar(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var number json.Number
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if dec.Decode(&number) == nil {
		return number.String()
	}
	return ""
}

func parsePID(value string) int {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") || len(value) > 7 {
		return 0
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func firstPID(obj map[string]json.RawMessage, keys ...string) int {
	raw, ok := firstRaw(obj, keys...)
	if !ok {
		return 0
	}
	return parsePID(jsonScalar(raw))
}

func firstRaw(obj map[string]json.RawMessage, keys ...string) (json.RawMessage, bool) {
	for _, key := range keys {
		if raw, ok := obj[key]; ok {
			return raw, true
		}
	}
	return nil, false
}

func hasJSONKey(obj map[string]json.RawMessage, key string) bool {
	_, ok := obj[key]
	return ok
}

func parseCrashTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC()
		}
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}
	}
	switch {
	case n > 1e17:
		return time.Unix(0, n).UTC()
	case n > 1e14:
		return time.UnixMicro(n).UTC()
	case n > 1e11:
		return time.UnixMilli(n).UTC()
	default:
		return time.Unix(n, 0).UTC()
	}
}

func validatedTimestamp(t time.Time) (string, error) {
	formatted := t.UTC().Format(time.RFC3339)
	if !timestampPattern.MatchString(formatted) {
		return "", errMalformedCrash
	}
	return formatted, nil
}

func validatedPIDArg(pid int) (string, error) {
	if pid <= 0 {
		return "", errMalformedCrash
	}
	arg := "COREDUMP_PID=" + strconv.Itoa(pid)
	if !pidArgPattern.MatchString(arg) {
		return "", errMalformedCrash
	}
	return arg, nil
}
