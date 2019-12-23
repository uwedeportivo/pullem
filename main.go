package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var cleanBranches = flag.Bool("prune", false, "if set prunes orphaned local branches")

func pathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

func defaultBranch(path string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = path

	outBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	output := strings.TrimSpace(string(outBytes))

	return output, nil
}

func isOnDefault(path string, branch string) (bool, error) {
	cmd := exec.Command("git", "symbolic-ref", "--quiet", "HEAD")
	cmd.Dir = path

	outBytes, err := cmd.Output()
	if err != nil {
		return false, err
	}
	output := strings.TrimSpace(string(outBytes))
	return output == "refs/heads/" + branch, nil
}

func isClean(path string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = path

	outBytes, err := cmd.Output()
	if err != nil {
		return false, err
	}
	output := strings.TrimSpace(string(outBytes))
	return output == "", nil
}

func pull(path string, branch string) error {
	cmd := exec.Command("git", "pull", "origin", branch, "--ff-only")
	cmd.Dir = path

	return cmd.Run()
}

func localRefs(path string, branch string) ([]string, error) {
	cmd := exec.Command("git", "for-each-ref", "--format", "%(refname) %(upstream)", "refs/heads")
	cmd.Dir = path

	outBytes, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	output := strings.TrimSpace(string(outBytes))
	lines := strings.Split(output, "\n")

	var lrfs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == ""  || strings.HasPrefix(line, "refs/heads/master") ||
			strings.HasPrefix(line, "refs/heads/" + branch) {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) == 2 {
			continue
		}

		if !strings.HasPrefix(parts[0], "refs/heads/") {
			continue
		}

		lrfs = append(lrfs, parts[0][len("refs/heads/"):])
	}

	return lrfs, nil
}

func deleteBranch(path string, localRef string) error {
	ok := askForConfirmation(fmt.Sprintf("\tDo you really want to delete branch %s", localRef))
	if !ok {
		return nil
	}

	cmd := exec.Command("git", "branch", "-D", localRef)
	cmd.Dir = path

	return cmd.Run()
}

func processDir(root string, path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		fmt.Printf("❌  %s failed processing %v\n", rel, err)
		return filepath.SkipDir
	}

	if !info.IsDir() {
		return nil
	}

	isGitDir, err := pathExists(filepath.Join(path, ".git"))
	if err != nil {
		fmt.Printf("❌  %s failed processing %v\n", rel, err)
		return filepath.SkipDir
	}

	if !isGitDir {
		return nil
	}

	branch, err := defaultBranch(path)
	if err != nil {
		fmt.Printf("❌  %s failed processing %v\n", rel, err)
		return filepath.SkipDir
	}

	onDefault, err := isOnDefault(path, branch)
	if err != nil {
		fmt.Printf("❌  %s failed processing %v\n", rel, err)
		return filepath.SkipDir
	}
	if !onDefault {
		fmt.Printf("❌  %s not on default branch\n", rel)
		return filepath.SkipDir
	}

	clean, err := isClean(path)
	if err != nil {
		fmt.Printf("❌  %s failed processing %v\n", rel, err)
		return filepath.SkipDir
	}
	if !clean {
		fmt.Printf("❌  %s not clean\n", rel)
		return filepath.SkipDir
	}

	err = pull(path, branch)
	if err != nil {
		fmt.Printf("❌  %s fast forwarding not possible\n", rel)
		return filepath.SkipDir
	}

	fmt.Printf("✅  %s updated\n", rel)

	if *cleanBranches {
		lrfs, err := localRefs(path, branch)
		if err != nil {
			fmt.Printf("\t❌ failed pruning orphaned branches %v\n", err)
			return filepath.SkipDir
		}

		for _, lrf := range lrfs {
			err = deleteBranch(path, lrf)
			if err != nil {
				fmt.Printf("\t❌ failed pruning orphaned branch %s %v\n", lrf, err)
			} else {
				fmt.Printf("\t✅ pruned orphaned branch %s\n", lrf)
			}
		}
	}
	return filepath.SkipDir
}


func main() {
	flag.Parse()

	if len(flag.Args()) > 1 {
		fmt.Println(`
Usage:
    pullem             
         recursively updates git repos starting from current working dir)
    pullem some_path
         recursively updates git repos starting from specified path`)
		os.Exit(0)
	}

	root := "."
	if len(flag.Args()) == 1 {
		root = flag.Arg(0)
	}

	root, err := filepath.Abs(root)
	if err != nil {
		log.Fatal(err)
	}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		return processDir(root, path, info, err)
	})
	if err != nil {
		log.Fatal(err)
	}
}
