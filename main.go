package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

const version = "0.1.5"

var (
	cyan   = color.New(color.FgCyan).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
)

type fileEntry struct {
	name        string
	size        int64
	isDir       bool
	modTime     time.Time
	toBeDeleted bool
}

func main() {
	allFlag := flag.Bool("all", false, "Skip individual confirmations and ask to delete all files at once")
	helpFlag := flag.Bool("help", false, "Show help information")
	hFlag := flag.Bool("h", false, "Show help information")
	versionFlag := flag.Bool("version", false, "Show version information")
	vFlag := flag.Bool("v", false, "Show version information")

	flag.Parse()

	if *helpFlag || *hFlag {
		fmt.Printf("\n%s\n\n  %s\n  %s\n\n",
			bold("USAGE:"),
			"dir-remover [path]",
			"--all    Skip individual confirmations and ask to delete all files at once")
		os.Exit(0)
	}

	if *versionFlag || *vFlag {
		fmt.Printf("dir-remover %s\n", version)
		os.Exit(0)
	}

	devPath := "."
	args := flag.Args()
	if len(args) > 0 {
		devPath = args[0]
	}

	if devPath == "." {
		var err error
		devPath, err = os.Getwd()
		if err != nil {
			fmt.Println(red("✗ Error:"), "Failed to get current directory")
			os.Exit(1)
		}
	}

	fileInfo, err := os.Stat(devPath)
	if err != nil {
		fmt.Println(red("✗ Error:"), "Failed to access path:", devPath)
		os.Exit(1)
	}

	if !fileInfo.IsDir() {
		handleSingleFile(devPath, fileInfo)
		return
	}

	fmt.Printf("\n%s %s\n\n", bold("DIRECTORY:"), cyan(devPath))

	if !confirm(fmt.Sprintf("Are you sure you want to proceed with %s?", cyan(devPath))) {
		fmt.Println(yellow("Operation aborted by user"))
		os.Exit(0)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Reading directory..."
	s.Start()

	entries, err := os.ReadDir(devPath)
	if err != nil {
		s.Stop()
		fmt.Println(red("✗ Error:"), "Failed to read directory:", err)
		os.Exit(1)
	}

	fileEntries := getFileEntries(entries, devPath)
	s.Stop()

	if len(fileEntries) == 0 {
		fmt.Println(yellow("No files or directories found"))
		os.Exit(0)
	}

	var toDelete []fileEntry
	if *allFlag {
		showFileTable(fileEntries, true)
		if confirm(fmt.Sprintf("Delete %s items shown above?", bold(fmt.Sprintf("%d", len(fileEntries))))) {
			toDelete = fileEntries
		}
	} else {
		toDelete = selectFilesInteractively(fileEntries)
	}

	if len(toDelete) > 0 {
		deleteSelectedEntries(toDelete, devPath)
	} else {
		fmt.Println(yellow("Nothing to delete"))
	}
}

func handleSingleFile(path string, info os.FileInfo) {
	fmt.Printf("\n%s %s\n\n", bold("FILE:"), cyan(path))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Size", "Modified"})
	table.SetBorder(true)
	table.SetRowLine(true)

	table.Append([]string{
		filepath.Base(path),
		formatSize(info.Size()),
		info.ModTime().Format("Jan 02, 2006 15:04:05"),
	})
	table.Render()
	fmt.Println()

	if confirm(fmt.Sprintf("Do you want to delete %s?", cyan(path))) {
		err := os.Remove(path)
		if err != nil {
			fmt.Println(red("✗ Failed:"), "Could not delete file:", err)
		} else {
			fmt.Println(green("✓ Success:"), "Deleted file:", path)
		}
	} else {
		fmt.Println(yellow("Operation aborted by user"))
	}
}

func getFileEntries(entries []fs.DirEntry, basePath string) []fileEntry {
	var fileEntries []fileEntry

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(basePath, entry.Name())

		fileEntries = append(fileEntries, fileEntry{
			name:    fullPath,
			size:    info.Size(),
			isDir:   entry.IsDir(),
			modTime: info.ModTime(),
		})
	}

	return fileEntries
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func showFileTable(entries []fileEntry, showAll bool) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Type", "Name", "Size", "Modified"})
	table.SetBorder(true)

	limit := len(entries)
	if !showAll && limit > 10 {
		limit = 10
	}

	for i := 0; i < limit; i++ {
		entry := entries[i]
		entryType := "File"
		if entry.isDir {
			entryType = "Dir"
		}

		name := filepath.Base(entry.name)
		if entry.toBeDeleted {
			name = green(name + " ✓")
		}

		table.Append([]string{
			entryType,
			name,
			formatSize(entry.size),
			entry.modTime.Format("Jan 02, 2006 15:04"),
		})
	}

	if !showAll && len(entries) > 10 {
		table.SetFooter([]string{"", fmt.Sprintf("... and %d more", len(entries)-10), "", ""})
	}

	fmt.Println()
	table.Render()
	fmt.Println()
}

func selectFilesInteractively(entries []fileEntry) []fileEntry {
	var selected []fileEntry

	fmt.Println(bold("\nSelect files/directories to delete:"))
	showFileTable(entries, true)

	for i, entry := range entries {
		name := filepath.Base(entry.name)
		entryType := "file"
		if entry.isDir {
			entryType = "directory"
		}

		if confirm(fmt.Sprintf("[%d/%d] Delete %s %s?",
			i+1, len(entries), entryType, cyan(name))) {
			entry.toBeDeleted = true
			selected = append(selected, entry)
		}
	}

	return selected
}

func deleteSelectedEntries(entries []fileEntry, basePath string) {
	if len(entries) == 0 {
		return
	}

	fmt.Println(bold("\nSummary of items to delete:"))
	showFileTable(entries, true)

	if !confirm(fmt.Sprintf("\nAre you sure you want to delete these %d items?", len(entries))) {
		fmt.Println(yellow("Operation aborted by user"))
		return
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Deleting files..."
	s.Start()

	successful := 0
	failed := 0

	for _, entry := range entries {
		err := os.RemoveAll(entry.name)
		if err != nil {
			failed++
		} else {
			successful++
		}
	}

	s.Stop()

	fmt.Println()
	fmt.Println(green(fmt.Sprintf("✓ Successfully deleted %d items", successful)))
	if failed > 0 {
		fmt.Println(red(fmt.Sprintf("✗ Failed to delete %d items", failed)))
	}
}

func confirm(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", message)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
