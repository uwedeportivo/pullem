package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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

//git symbolic-ref --short HEAD

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
