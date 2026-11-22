// repostat checks status of git repos inside a folder and prints a table.
//
// Usage: repostat [--fetch] [-r] [folder]
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

func git(repo string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

func findRepos(root string, recursive bool) []string {
	var repos []string
	if recursive {
		filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if exists(filepath.Join(p, ".git")) {
				repos = append(repos, p)
				return filepath.SkipDir
			}
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		})
	} else {
		entries, _ := os.ReadDir(root)
		for _, e := range entries {
			p := filepath.Join(root, e.Name())
			if fi, err := os.Stat(p); err == nil && fi.IsDir() && exists(filepath.Join(p, ".git")) {
				repos = append(repos, p)
			}
		}
	}
	sort.Strings(repos)
	return repos
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(s, "\n"))
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func inspect(repo, root string, fetch bool) []string {
	if fetch {
		exec.Command("git", "-C", repo, "fetch", "--quiet").Run()
	}
	branch := git(repo, "rev-parse", "--abbrev-ref", "HEAD")
	if branch == "" {
		branch = "?"
	}
	if branch == "HEAD" {
		branch = "(detached " + git(repo, "rev-parse", "--short", "HEAD") + ")"
	}

	status := git(repo, "status", "--porcelain")
	total, untracked := 0, 0
	if status != "" {
		for l := range strings.SplitSeq(status, "\n") {
			total++
			if strings.HasPrefix(l, "??") {
				untracked++
			}
		}
	}
	changed := total - untracked

	sync := "no upstream"
	if git(repo, "rev-parse", "--abbrev-ref", "@{u}") != "" {
		f := strings.Fields(git(repo, "rev-list", "--left-right", "--count", "HEAD...@{u}"))
		f = append(f, "?", "?")
		sync = fmt.Sprintf("+%s/-%s", f[0], f[1])
	}

	stash := lineCount(git(repo, "stash", "list"))
	date, subj, _ := strings.Cut(git(repo, "log", "-1", "--format=%cd|%s", "--date=short"), "|")
	if date == "" {
		date = "-"
	}
	subj = truncate(subj, 40)
	if subj == "" {
		subj = "-"
	}
	state := "clean"
	if total > 0 {
		state = "dirty"
	}
	rel, err := filepath.Rel(root, repo)
	if err != nil {
		rel = repo
	}
	return []string{rel, branch, state, fmt.Sprint(changed), fmt.Sprint(untracked), sync, fmt.Sprint(stash), date, subj}
}

func pad(s string, w int) string {
	n := len([]rune(s))
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

func main() {
	fetch := flag.Bool("fetch", false, "run `git fetch` first so ahead/behind is up to date")
	rec := flag.Bool("r", false, "scan recursively (stops descending at a repo root)")
	flag.BoolVar(rec, "recursive", false, "same as -r")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: repostat [--fetch] [-r] [folder]")
		flag.PrintDefaults()
	}
	flag.Parse()

	folder := "."
	if flag.NArg() > 0 {
		folder = flag.Arg(0)
	}
	root, err := filepath.Abs(folder)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	repos := findRepos(root, *rec)
	if len(repos) == 0 {
		fmt.Printf("No git repos found in %s\n", root)
		return
	}

	rows := make([][]string, len(repos))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i, r := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			rows[i] = inspect(r, root, *fetch)
		}()
	}
	wg.Wait()

	header := []string{"Repo", "Branch", "State", "Chg", "Untr", "Ahead/Behind", "Stash", "Last commit", "Subject"}
	widths := make([]int, len(header))
	for _, r := range append([][]string{header}, rows...) {
		for i, c := range r {
			if n := len([]rune(c)); n > widths[i] {
				widths[i] = n
			}
		}
	}
	line := func(r []string) {
		cells := make([]string, len(r))
		for i, c := range r {
			cells[i] = pad(c, widths[i])
		}
		fmt.Println(strings.Join(cells, " | "))
	}
	line(header)
	seps := make([]string, len(widths))
	for i, w := range widths {
		seps[i] = strings.Repeat("-", w)
	}
	fmt.Println(strings.Join(seps, "-+-"))
	dirty := 0
	for _, r := range rows {
		line(r)
		if r[2] == "dirty" {
			dirty++
		}
	}
	fmt.Printf("\n%d repos, %d dirty\n", len(rows), dirty)
}
