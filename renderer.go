package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	defaultRenderWidth   = 80
	wideLayoutWidth      = 64
	compactLayoutWidth   = 38
	defaultFetchInterval = 3 * time.Minute
)

type renderRow struct {
	Path     string
	Branch   string
	Worktree string
	Remote   string
}

func renderSnapshot(statuses []repositorySnapshot, width int, selected string, now time.Time, fetchInterval time.Duration, palette renderPalette) string {
	if width <= 0 {
		width = defaultRenderWidth
	}
	ordered := append([]repositorySnapshot(nil), statuses...)
	sortSnapshots(ordered)
	rows := make([]renderRow, 0, len(ordered))
	clean, dirty, ahead, behind, failures := 0, 0, 0, 0, 0
	for _, status := range ordered {
		branch := status.Branch
		if status.Error != "" {
			failures++
			branch = palette.missing
		} else if status.dirty() {
			dirty++
		} else {
			clean++
		}
		if status.Ahead > 0 {
			ahead++
		}
		if status.Behind > 0 {
			behind++
		}
		rows = append(rows, renderRow{
			Path:     safeCell(status.Path),
			Branch:   safeCell(branch),
			Worktree: worktreeText(status.repoStatus),
			Remote:   remoteText(status, now, fetchInterval, palette),
		})
	}

	prefixWidth := 0
	if selected != "" {
		prefixWidth = displayWidth(palette.selection)
	}
	contentWidth := max(1, width-prefixWidth)
	lines := renderTable(rows, ordered, contentWidth, width, selected, palette)
	for index := range lines {
		lines[index] = fitLine(lines[index], width)
	}
	lines = append(lines, "", fitLine(renderSummary(width, len(statuses), clean, dirty, ahead, behind, failures), width))
	if selected != "" {
		for _, status := range ordered {
			if status.Path == selected {
				lines = append(lines, "")
				lines = append(lines, renderDetails(status, width, now, fetchInterval, palette)...)
				break
			}
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderTable(rows []renderRow, statuses []repositorySnapshot, width, layoutWidth int, selected string, palette renderPalette) []string {
	var lines []string
	switch {
	case layoutWidth >= wideLayoutWidth:
		const branchWidth, worktreeWidth, remoteWidth = 16, 12, 14
		pathWidth := max(1, width-branchWidth-worktreeWidth-remoteWidth-6)
		lines = append(lines,
			wideLine("REPOSITORY", "BRANCH", "WORKTREE", "REMOTE", pathWidth, branchWidth, worktreeWidth, remoteWidth),
			wideLine(strings.Repeat(palette.rule, pathWidth), strings.Repeat(palette.rule, branchWidth), strings.Repeat(palette.rule, worktreeWidth), strings.Repeat(palette.rule, remoteWidth), pathWidth, branchWidth, worktreeWidth, remoteWidth),
		)
		for index, row := range rows {
			line := wideLine(row.Path, row.Branch, row.Worktree, row.Remote, pathWidth, branchWidth, worktreeWidth, remoteWidth)
			lines = append(lines, selectionPrefix(statuses[index].Path, selected, palette)+line)
		}
	case layoutWidth >= compactLayoutWidth:
		const worktreeWidth, remoteWidth = 12, 14
		pathWidth := max(1, width-worktreeWidth-remoteWidth-4)
		lines = append(lines,
			compactLine("REPOSITORY", "WORKTREE", "REMOTE", pathWidth, worktreeWidth, remoteWidth),
			compactLine(strings.Repeat(palette.rule, pathWidth), strings.Repeat(palette.rule, worktreeWidth), strings.Repeat(palette.rule, remoteWidth), pathWidth, worktreeWidth, remoteWidth),
		)
		for index, row := range rows {
			line := compactLine(row.Path, row.Worktree, row.Remote, pathWidth, worktreeWidth, remoteWidth)
			lines = append(lines, selectionPrefix(statuses[index].Path, selected, palette)+line)
		}
	default:
		stateWidth := min(20, max(5, width*2/3))
		pathWidth := max(1, width-stateWidth-2)
		stateWidth = max(1, width-pathWidth-2)
		lines = append(lines,
			narrowLine(truncateEnd("REPOSITORY", pathWidth), "STATE", pathWidth, stateWidth),
			narrowLine(strings.Repeat(palette.rule, pathWidth), strings.Repeat(palette.rule, stateWidth), pathWidth, stateWidth),
		)
		for index, row := range rows {
			line := narrowLine(row.Path, strings.TrimSpace(row.Worktree+" "+row.Remote), pathWidth, stateWidth)
			lines = append(lines, selectionPrefix(statuses[index].Path, selected, palette)+line)
		}
	}
	if selected != "" {
		indent := strings.Repeat(" ", displayWidth(palette.selection))
		lines[0] = indent + lines[0]
		lines[1] = indent + lines[1]
	}
	return lines
}

func wideLine(path, branch, worktree, remote string, pathWidth, branchWidth, worktreeWidth, remoteWidth int) string {
	return padRight(middleTruncate(path, pathWidth), pathWidth) + "  " +
		padRight(truncateEnd(branch, branchWidth), branchWidth) + "  " +
		padRight(truncateEnd(worktree, worktreeWidth), worktreeWidth) + "  " +
		padRight(truncateEnd(remote, remoteWidth), remoteWidth)
}

func compactLine(path, worktree, remote string, pathWidth, worktreeWidth, remoteWidth int) string {
	return padRight(middleTruncate(path, pathWidth), pathWidth) + "  " +
		padRight(truncateEnd(worktree, worktreeWidth), worktreeWidth) + "  " +
		padRight(truncateEnd(remote, remoteWidth), remoteWidth)
}

func narrowLine(path, state string, pathWidth, stateWidth int) string {
	return padRight(middleTruncate(path, pathWidth), pathWidth) + "  " + padRight(truncateEnd(state, stateWidth), stateWidth)
}

func selectionPrefix(path, selected string, palette renderPalette) string {
	if selected == "" {
		return ""
	}
	if path == selected {
		return palette.selection
	}
	return strings.Repeat(" ", utf8.RuneCountInString(palette.selection))
}

func renderSummary(width, repositories, clean, dirty, ahead, behind, failures int) string {
	switch {
	case width >= wideLayoutWidth:
		return fmt.Sprintf("%d %s (%d clean, %d dirty, %d ahead, %d behind, %d %s)",
			repositories, plural(repositories, "repo"), clean, dirty, ahead, behind, failures, plural(failures, "error"))
	case width >= compactLayoutWidth:
		return fmt.Sprintf("%d %s (%d clean, %d dirty, %d %s)",
			repositories, plural(repositories, "repo"), clean, dirty, failures, plural(failures, "error"))
	default:
		return fmt.Sprintf("%d %s (%d dirty, %d %s)",
			repositories, plural(repositories, "repo"), dirty, failures, plural(failures, "error"))
	}
}

func remoteText(snapshot repositorySnapshot, now time.Time, fetchInterval time.Duration, palette renderPalette) string {
	state := freshnessState(snapshot, now, fetchInterval)
	switch state {
	case "error", "no remote", "no upstream":
		return state
	}
	distance := remoteDistance(snapshot.repoStatus, palette)
	if distance == "" {
		if state == "current" {
			return palette.current + state
		}
		return state
	}
	return distance + " " + state
}

func freshnessState(snapshot repositorySnapshot, now time.Time, fetchInterval time.Duration) string {
	switch {
	case snapshot.Error != "":
		return "error"
	case !snapshot.HasRemote && !snapshot.HasUpstream:
		return "no remote"
	case !snapshot.HasUpstream:
		return "no upstream"
	case snapshot.Fetching:
		return "fetching"
	case snapshot.Fetch.Error != "" && snapshot.Fetch.LastSuccess.IsZero():
		return "failed"
	case snapshot.Fetch.Error != "":
		return "stale"
	case snapshot.Fetch.LastSuccess.IsZero():
		return "cached"
	case fetchInterval > 0 && now.Sub(snapshot.Fetch.LastSuccess) > fetchInterval:
		return "stale"
	default:
		return "current"
	}
}

func remoteDistance(status repoStatus, palette renderPalette) string {
	var parts []string
	if status.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("%s%d", palette.ahead, status.Ahead))
	}
	if status.Behind > 0 {
		parts = append(parts, fmt.Sprintf("%s%d", palette.behind, status.Behind))
	}
	return strings.Join(parts, " ")
}

func renderDetails(snapshot repositorySnapshot, width int, now time.Time, fetchInterval time.Duration, palette renderPalette) []string {
	lines := wrappedLine("details: "+safeCell(snapshot.Path), width)
	if snapshot.Error != "" {
		lines = append(lines, wrappedLine("local error: "+safeCell(oneLine(snapshot.Error)), width)...)
	} else {
		lines = append(lines, wrappedLine(fmt.Sprintf("local: branch %s; worktree %s", safeCell(snapshot.Branch), worktreeText(snapshot.repoStatus)), width)...)
	}

	remote := "remote: " + remoteText(snapshot, now, fetchInterval, palette)
	if snapshot.HasUpstream && fetchInterval > 0 {
		remote += "; freshness interval " + fetchInterval.String()
	}
	lines = append(lines, wrappedLine(remote, width)...)

	switch {
	case snapshot.Fetching:
		lines = append(lines, wrappedLine("fetch: in progress", width)...)
	case snapshot.Fetch.LastAttempt.IsZero():
		lines = append(lines, wrappedLine("fetch: never attempted", width)...)
	case snapshot.Fetch.Error != "":
		lines = append(lines, wrappedLine(fmt.Sprintf("fetch: failed; attempted %s; duration %s",
			formatTime(snapshot.Fetch.LastAttempt), snapshot.Fetch.Duration.Round(time.Millisecond)), width)...)
	default:
		lines = append(lines, wrappedLine(fmt.Sprintf("fetch: succeeded %s; duration %s",
			formatTime(snapshot.Fetch.LastSuccess), snapshot.Fetch.Duration.Round(time.Millisecond)), width)...)
	}
	if !snapshot.Fetch.LastSuccess.IsZero() && (snapshot.Fetching || snapshot.Fetch.Error != "") {
		lines = append(lines, wrappedLine("last success: "+formatTime(snapshot.Fetch.LastSuccess), width)...)
	}
	if snapshot.Fetch.Error != "" {
		lines = append(lines, wrappedLine("fetch error: "+safeCell(oneLine(snapshot.Fetch.Error)), width)...)
	}
	return lines
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func wrappedLine(value string, width int) []string {
	width = max(1, width)
	if value == "" {
		return []string{""}
	}
	var lines []string
	for value != "" {
		part := takePrefixCells(value, width)
		consumed := len(part)
		if consumed == 0 {
			_, consumed = utf8.DecodeRuneInString(value)
			part = "?"
		}
		lines = append(lines, part)
		value = value[consumed:]
	}
	return lines
}

func middleTruncate(value string, width int) string {
	if displayWidth(value) <= width {
		return value
	}
	if width <= 3 {
		return takeSuffixCells(value, width)
	}
	available := width - 3
	left := available / 3
	right := available - left
	return takePrefixCells(value, left) + "..." + takeSuffixCells(value, right)
}

func truncateEnd(value string, width int) string {
	if displayWidth(value) <= width {
		return value
	}
	if width <= 3 {
		return takePrefixCells(value, width)
	}
	return takePrefixCells(value, width-3) + "..."
}

func padRight(value string, width int) string {
	return value + strings.Repeat(" ", max(0, width-displayWidth(value)))
}

func fitLine(value string, width int) string {
	return truncateEnd(strings.TrimRight(value, " "), width)
}

func displayWidth(value string) int {
	width := 0
	for _, character := range value {
		width += runeWidth(character)
	}
	return width
}

func runeWidth(character rune) int {
	switch {
	case unicode.IsControl(character),
		unicode.Is(unicode.Mn, character),
		unicode.Is(unicode.Me, character),
		unicode.Is(unicode.Cf, character):
		return 0
	case isWideRune(character):
		return 2
	default:
		return 1
	}
}

// ponytail: this compact wcwidth approximation covers common repository names;
// use a maintained grapheme-width package if real terminals expose edge cases.
func isWideRune(character rune) bool {
	return character >= 0x1100 && (character <= 0x115f ||
		character == 0x2329 || character == 0x232a ||
		character >= 0x2e80 && character <= 0xa4cf && character != 0x303f ||
		character >= 0xac00 && character <= 0xd7a3 ||
		character >= 0xf900 && character <= 0xfaff ||
		character >= 0xfe10 && character <= 0xfe19 ||
		character >= 0xfe30 && character <= 0xfe6f ||
		character >= 0xff00 && character <= 0xff60 ||
		character >= 0xffe0 && character <= 0xffe6 ||
		character >= 0x1f300 && character <= 0x1faff ||
		character >= 0x20000 && character <= 0x3fffd)
}

func takePrefixCells(value string, width int) string {
	if width <= 0 {
		return ""
	}
	cells, end := 0, 0
	for index, character := range value {
		characterWidth := runeWidth(character)
		if cells+characterWidth > width {
			break
		}
		cells += characterWidth
		_, size := utf8.DecodeRuneInString(value[index:])
		end = index + size
	}
	return value[:end]
}

func takeSuffixCells(value string, width int) string {
	if width <= 0 {
		return ""
	}
	characters := []rune(value)
	cells, start := 0, len(characters)
	for index := len(characters) - 1; index >= 0; index-- {
		characterWidth := runeWidth(characters[index])
		if cells+characterWidth > width {
			break
		}
		cells += characterWidth
		start = index
	}
	for start < len(characters) && runeWidth(characters[start]) == 0 {
		start++
	}
	return string(characters[start:])
}
