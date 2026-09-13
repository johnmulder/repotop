package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/rivo/uniseg"
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
	return renderSnapshotSized(statuses, width, 0, selected, now, fetchInterval, palette)
}

func renderSnapshotSized(statuses []repositorySnapshot, width, height int, selected string, now time.Time, fetchInterval time.Duration, palette renderPalette) string {
	return renderStyledSnapshotSized(statuses, width, height, selected, now, fetchInterval, palette, terminalStyle{})
}

func renderStyledSnapshotSized(statuses []repositorySnapshot, width, height int, selected string, now time.Time, fetchInterval time.Duration, palette renderPalette, style terminalStyle) string {
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

	selectedIndex := -1
	if selected != "" {
		for index, status := range ordered {
			if status.Path == selected {
				selectedIndex = index
				break
			}
		}
	}
	start, end := 0, len(ordered)
	if height > 0 && len(ordered) > 0 {
		start, end = selectedWindow(len(ordered), selectedIndex, max(1, height-4))
	}

	prefixWidth := 0
	if selected != "" {
		prefixWidth = displayWidth(palette.selection)
	}
	contentWidth := max(1, width-prefixWidth)
	lines := renderTable(rows[start:end], ordered[start:end], contentWidth, width, selected, palette)
	for index := range lines {
		lines[index] = fitLine(lines[index], width)
	}
	for index, status := range ordered[start:end] {
		lines[index+2] = style.apply(status, now, fetchInterval, lines[index+2])
	}
	lines = append(lines, "", fitLine(renderSummary(width, len(statuses), clean, dirty, ahead, behind, failures), width))
	if height > 0 && len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n") + "\n"
}

func selectedWindow(length, selected, limit int) (int, int) {
	if limit >= length {
		return 0, length
	}
	if selected < 0 {
		return 0, limit
	}
	start := max(0, selected-limit/2)
	if start+limit > length {
		start = length - limit
	}
	return start, start + limit
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
	return strings.Repeat(" ", displayWidth(palette.selection))
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
	return uniseg.StringWidth(value)
}

func takePrefixCells(value string, width int) string {
	if width <= 0 {
		return ""
	}
	cells, end, state := 0, 0, -1
	rest := value
	for rest != "" {
		cluster, next, clusterWidth, nextState := uniseg.FirstGraphemeClusterInString(rest, state)
		if cells+clusterWidth > width {
			break
		}
		cells += clusterWidth
		end += len(cluster)
		rest, state = next, nextState
	}
	return value[:end]
}

func takeSuffixCells(value string, width int) string {
	if width <= 0 {
		return ""
	}
	cells := displayWidth(value)
	if cells <= width {
		return value
	}
	state := -1
	for value != "" && cells > width {
		_, rest, clusterWidth, nextState := uniseg.FirstGraphemeClusterInString(value, state)
		value, state = rest, nextState
		cells -= clusterWidth
	}
	for value != "" {
		_, rest, clusterWidth, nextState := uniseg.FirstGraphemeClusterInString(value, state)
		if clusterWidth > 0 {
			break
		}
		value, state = rest, nextState
	}
	return value
}
