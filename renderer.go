package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	defaultRenderWidth = 80
	wideLayoutWidth    = 64
	compactLayoutWidth = 38
)

type renderRow struct {
	Path     string
	Branch   string
	Worktree string
	Remote   string
}

func renderSnapshot(statuses []repoStatus, width int, selected string) string {
	if width <= 0 {
		width = defaultRenderWidth
	}
	ordered := append([]repoStatus(nil), statuses...)
	sortStatuses(ordered)
	rows := make([]renderRow, 0, len(ordered))
	clean, dirty, ahead, behind, failures := 0, 0, 0, 0, 0
	for _, status := range ordered {
		branch := status.Branch
		if status.Error != "" {
			failures++
			branch = "-"
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
			Worktree: worktreeText(status),
			Remote:   remoteText(status),
		})
	}

	prefixWidth := 0
	if selected != "" {
		prefixWidth = 2
	}
	contentWidth := max(1, width-prefixWidth)
	lines := renderTable(rows, ordered, contentWidth, width, selected)
	for index := range lines {
		lines[index] = fitLine(lines[index], width)
	}
	lines = append(lines, "", fitLine(renderSummary(width, len(statuses), clean, dirty, ahead, behind, failures), width))
	return strings.Join(lines, "\n") + "\n"
}

func renderTable(rows []renderRow, statuses []repoStatus, width, layoutWidth int, selected string) []string {
	var lines []string
	switch {
	case layoutWidth >= wideLayoutWidth:
		const branchWidth, worktreeWidth, remoteWidth = 16, 12, 11
		pathWidth := max(1, width-branchWidth-worktreeWidth-remoteWidth-6)
		lines = append(lines,
			wideLine("REPOSITORY", "BRANCH", "WORKTREE", "REMOTE", pathWidth, branchWidth, worktreeWidth, remoteWidth),
			wideLine(strings.Repeat("-", pathWidth), strings.Repeat("-", branchWidth), strings.Repeat("-", worktreeWidth), strings.Repeat("-", remoteWidth), pathWidth, branchWidth, worktreeWidth, remoteWidth),
		)
		for index, row := range rows {
			line := wideLine(row.Path, row.Branch, row.Worktree, row.Remote, pathWidth, branchWidth, worktreeWidth, remoteWidth)
			lines = append(lines, selectionPrefix(statuses[index].Path, selected)+line)
		}
	case layoutWidth >= compactLayoutWidth:
		const worktreeWidth, remoteWidth = 12, 11
		pathWidth := max(1, width-worktreeWidth-remoteWidth-4)
		lines = append(lines,
			compactLine("REPOSITORY", "WORKTREE", "REMOTE", pathWidth, worktreeWidth, remoteWidth),
			compactLine(strings.Repeat("-", pathWidth), strings.Repeat("-", worktreeWidth), strings.Repeat("-", remoteWidth), pathWidth, worktreeWidth, remoteWidth),
		)
		for index, row := range rows {
			line := compactLine(row.Path, row.Worktree, row.Remote, pathWidth, worktreeWidth, remoteWidth)
			lines = append(lines, selectionPrefix(statuses[index].Path, selected)+line)
		}
	default:
		stateWidth := min(20, max(5, width*2/3))
		pathWidth := max(1, width-stateWidth-2)
		stateWidth = max(1, width-pathWidth-2)
		lines = append(lines,
			narrowLine(truncateEnd("REPOSITORY", pathWidth), "STATE", pathWidth, stateWidth),
			narrowLine(strings.Repeat("-", pathWidth), strings.Repeat("-", stateWidth), pathWidth, stateWidth),
		)
		for index, row := range rows {
			line := narrowLine(row.Path, strings.TrimSpace(row.Worktree+" "+row.Remote), pathWidth, stateWidth)
			lines = append(lines, selectionPrefix(statuses[index].Path, selected)+line)
		}
	}
	if selected != "" {
		lines[0] = "  " + lines[0]
		lines[1] = "  " + lines[1]
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

func selectionPrefix(path, selected string) string {
	if selected == "" {
		return ""
	}
	if path == selected {
		return "> "
	}
	return "  "
}

func renderSummary(width, repositories, clean, dirty, ahead, behind, failures int) string {
	switch {
	case width >= wideLayoutWidth:
		return fmt.Sprintf("%d %s  %d clean  %d dirty  %d ahead  %d behind  %d %s",
			repositories, plural(repositories, "repo"), clean, dirty, ahead, behind, failures, plural(failures, "error"))
	case width >= compactLayoutWidth:
		return fmt.Sprintf("%d %s  %d clean  %d dirty  %d %s",
			repositories, plural(repositories, "repo"), clean, dirty, failures, plural(failures, "error"))
	default:
		return fmt.Sprintf("%d %s  %d dirty  %d %s",
			repositories, plural(repositories, "repo"), dirty, failures, plural(failures, "error"))
	}
}

func middleTruncate(value string, width int) string {
	characters := []rune(value)
	if len(characters) <= width {
		return value
	}
	if width <= 3 {
		return string(characters[len(characters)-width:])
	}
	available := width - 3
	left := available / 3
	right := available - left
	return string(characters[:left]) + "..." + string(characters[len(characters)-right:])
}

func truncateEnd(value string, width int) string {
	characters := []rune(value)
	if len(characters) <= width {
		return value
	}
	if width <= 3 {
		return string(characters[:width])
	}
	return string(characters[:width-3]) + "..."
}

func padRight(value string, width int) string {
	return value + strings.Repeat(" ", max(0, width-utf8.RuneCountInString(value)))
}

func fitLine(value string, width int) string {
	return truncateEnd(strings.TrimRight(value, " "), width)
}
