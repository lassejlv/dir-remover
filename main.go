package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

const version = "0.1.7"

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
	log.SetFlags(0)

	allFlag := flag.Bool("all", false, "Skip individual confirmations and ask to delete all files at once")
	helpFlag := flag.Bool("help", false, "Show help information")
	hFlag := flag.Bool("h", false, "Show help information (alias for -help)")
	versionFlag := flag.Bool("version", false, "Show version information")
	vFlag := flag.Bool("v", false, "Show version information (alias for -version)")

	flag.Parse()

	if *helpFlag || *hFlag {
		fmt.Printf("\n%s\n\n  %s\n  %s %s\n  %s %s\n\n",
			bold("USAGE:"),
			"dir-remover [path]",
			"--all       ", "Skip individual confirmations and ask to delete all items at once",
			"--help, -h  ", "Show this help information",
			"--version, -v", "Show version information")
		os.Exit(0)
	}

	if *versionFlag || *vFlag {
		fmt.Printf("dir-remover v%s\n", version)
		os.Exit(0)
	}

	targetPath := "."
	args := flag.Args()
	if len(args) > 0 {
		targetPath = args[0]
	}

	var err error
	targetPath, err = filepath.Abs(targetPath)
	if err != nil {
		log.Fatalf("%s Failed to get absolute path for '%s': %v", red("✗ Error:"), targetPath, err)
	}

	fileInfo, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("%s Path does not exist: %s", red("✗ Error:"), cyan(targetPath))
		}
		log.Fatalf("%s Failed to access path: %s (%v)", red("✗ Error:"), cyan(targetPath), err)
	}

	if !fileInfo.IsDir() {
		handleSingleFile(targetPath, fileInfo)
		return
	}

	fmt.Printf("\n%s %s\n\n", bold("DIRECTORY:"), cyan(targetPath))

	if !confirm(fmt.Sprintf("Scan and potentially delete items within %s?", cyan(targetPath))) {
		fmt.Println(yellow("Operation aborted by user."))
		os.Exit(0)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Reading directory %s...", cyan(filepath.Base(targetPath)))
	s.Color("cyan")
	s.Start()

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		s.Stop()
		log.Fatalf("%s Failed to read directory: %v", red("✗ Error:"), err)
	}

	fileEntries := getFileEntries(entries, targetPath)
	s.Stop()

	if len(fileEntries) == 0 {
		fmt.Println(yellow("No files or subdirectories found to delete in %s", cyan(targetPath)))
		os.Exit(0)
	}

	var toDelete []fileEntry
	reader := bufio.NewReader(os.Stdin)

	if *allFlag {
		fmt.Println(bold("\nItems found:"))
		showFileTable(fileEntries, false)
		if confirmWithReader(reader, fmt.Sprintf("Delete all %s items shown above?", bold(fmt.Sprintf("%d", len(fileEntries))))) {
			for i := range fileEntries {
				fileEntries[i].toBeDeleted = true
			}
			toDelete = fileEntries
		}
	} else {
		toDelete = selectFilesInteractively(fileEntries, reader)
	}

	if len(toDelete) > 0 {
		deleteSelectedEntries(toDelete, reader)
	} else {
		fmt.Println(yellow("No items selected for deletion."))
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
		info.ModTime().Format("Jan 02, 2006 15:04"),
	})
	table.Render()
	fmt.Println()

	if confirm(fmt.Sprintf("Do you want to delete the file %s?", cyan(filepath.Base(path)))) {
		err := os.Remove(path)
		if err != nil {
			fmt.Printf("%s Failed to delete file: %v\n", red("✗ Failed:"), err)
		} else {
			fmt.Printf("%s Deleted file: %s\n", green("✓ Success:"), cyan(path))
		}
	} else {
		fmt.Println(yellow("Operation aborted by user."))
	}
}

func getFileEntries(entries []fs.DirEntry, basePath string) []fileEntry {
	var fileEntries []fileEntry

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			log.Printf("%s Could not get info for '%s': %v\n", yellow("⚠ Warning:"), entry.Name(), err)
			continue
		}

		fullPath := filepath.Join(basePath, entry.Name())

		fileEntries = append(fileEntries, fileEntry{
			name:        fullPath,
			size:        info.Size(),
			isDir:       entry.IsDir(),
			modTime:     info.ModTime(),
			toBeDeleted: false,
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
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func showFileTable(entries []fileEntry, showCheckmarks bool) {
	if len(entries) == 0 {
		fmt.Println(yellow(" (No items to display) "))
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Type", "Name", "Size", "Modified"})
	table.SetBorder(true)
	table.SetRowLine(true)
	table.SetAutoWrapText(false)

	for _, entry := range entries {
		entryType := "File"
		if entry.isDir {
			entryType = "Dir "
		}

		name := filepath.Base(entry.name)
		if showCheckmarks && entry.toBeDeleted {
			name = green(name + " ✓")
		}

		table.Append([]string{
			entryType,
			name,
			formatSize(entry.size),
			entry.modTime.Format("Jan 02, 2006 15:04"),
		})
	}

	fmt.Println()
	table.Render()
	fmt.Println()
}

func selectFilesInteractively(entries []fileEntry, reader *bufio.Reader) []fileEntry {
	var selected []fileEntry

	fmt.Println(bold("\nSelect items to delete (y/N):"))
	showFileTable(entries, false)

	displayEntries := make([]fileEntry, len(entries))
	copy(displayEntries, entries)

	for i := range displayEntries {
		entry := &displayEntries[i]

		name := filepath.Base(entry.name)
		entryType := "file"
		if entry.isDir {
			entryType = "directory"
		}

		if confirmWithReader(reader, fmt.Sprintf("[%d/%d] Delete %s %s (%s)?",
			i+1, len(displayEntries), entryType, cyan(name), formatSize(entry.size))) {

			entry.toBeDeleted = true

			entries[i].toBeDeleted = true
			selected = append(selected, entries[i])

		}
	}

	if len(selected) > 0 {
		fmt.Println(bold("\nItems marked for deletion:"))
		showFileTable(selected, true)
	}

	return selected
}

func deleteSelectedEntries(toDelete []fileEntry, reader *bufio.Reader) {
	if len(toDelete) == 0 {
		fmt.Println(yellow("Nothing selected to delete."))
		return
	}

	if !confirmWithReader(reader, fmt.Sprintf("\nProceed with deleting these %s items?", bold(fmt.Sprintf("%d", len(toDelete))))) {
		fmt.Println(yellow("Operation aborted by user."))
		return
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Deleting items..."
	s.Color("cyan")
	s.Start()

	successful := 0
	failed := 0
	var failedItems []struct {
		Name string
		Err  error
	}

	for _, entry := range toDelete {
		err := os.RemoveAll(entry.name)
		if err != nil {
			failed++
			failedItems = append(failedItems, struct {
				Name string
				Err  error
			}{entry.name, err})
		} else {
			successful++
		}
	}

	s.Stop()
	fmt.Println()

	if successful > 0 {
		fmt.Println(green(fmt.Sprintf("✓ Successfully deleted %d items.", successful)))
	}
	if failed > 0 {
		fmt.Println(red(fmt.Sprintf("✗ Failed to delete %d items:", failed)))
		for _, item := range failedItems {
			fmt.Printf("  - %s: %v\n", cyan(filepath.Base(item.Name)), item.Err)
		}
	}
}

func confirm(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	return confirmWithReader(reader, message)
}

func confirmWithReader(reader *bufio.Reader, message string) bool {
	for {
		fmt.Printf("%s [y/N]: ", message)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("%s Error reading input: %v. Assuming No.", yellow("⚠ Warning:"), err)
			return false
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		}
		if response == "n" || response == "no" || response == "" {
			return false
		}
		fmt.Println(yellow("Invalid input. Please enter 'y' or 'n'."))
	}
}
