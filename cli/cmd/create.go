package cmd

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v58/github"
	"github.com/hairyhenderson/go-which"
	"github.com/spf13/cobra"
)

const (
	GH_OWNER = "redwoodjs"
	GH_REPO  = "rw-cli"
)

var (
	// Used for flags.
	yesFlag           bool
	overwriteFlag     bool
	typescriptFlag    bool
	gitInitFlag       bool
	commitMessageFlag string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new redwood project",
	Args:  cobra.MaximumNArgs(1),
	Run:   runCreate,
}

func init() {
	rootCmd.AddCommand(createCmd)

	// TODO(jgmw): PFlags doesn't support multi character short flags
	createCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Skip prompts and use defaults")
	createCmd.Flags().BoolVar(&overwriteFlag, "overwrite", false, "Create even if target directory isn't empty")
	createCmd.Flags().BoolVar(&typescriptFlag, "typescript", true, "Generate a TypeScript project")
	createCmd.Flags().BoolVar(&gitInitFlag, "git-init", true, "Initialize a git repository")
	createCmd.Flags().StringVarP(&commitMessageFlag, "commit-message", "m", "initial commit", "Commit message for the initial commit")

	// TODO(jgmw): Add flag for yarn install based on yarn version?
}

func runCreate(cmd *cobra.Command, args []string) {
	printInto()

	// Check node
	err := checkNode()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Check yarn
	err = checkYarn()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Target directory
	tDir := "./redwood-app"
	if len(args) == 0 {
		// TODO: Prompt for target directory
	} else {
		tDir = args[0]
	}
	vTDir, err := validateTargetDirectory(tDir, overwriteFlag)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("Target directory: %s\n", vTDir)

	// TS or JS
	useTS := typescriptFlag
	if !cmd.Flags().Changed("typescript") && !yesFlag {
		// TODO(jgmw): Prompt for TS or JS
	}
	fmt.Printf("Use TypeScript: %t\n", useTS)

	// Get the latest release
	client := github.NewClient(nil)
	if os.Getenv("GH_TOKEN") != "" {
		client = client.WithAuthToken(os.Getenv("GH_TOKEN"))
	}
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), GH_OWNER, GH_REPO)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	rTag := release.GetTagName()
	fmt.Printf("%v\n", rTag)

	templateAssetName := "CRWA_JS.zip"
	if useTS {
		templateAssetName = "CRWA_TS.zip"
	}

	templateAssetId := int64(0)
	for _, asset := range release.Assets {
		if asset.GetName() == templateAssetName {
			templateAssetId = asset.GetID()
		}
	}

	zipName := rTag + "-js.zip"
	if useTS {
		zipName = rTag + "-ts.zip"
	}

	// Check if we already have the template cached
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cachePath := filepath.Join(homeDir, ".redwood", "templates", zipName)
	cachedTemplate := false
	if _, err := os.Stat(cachePath); err == nil {
		cachedTemplate = true
	} else if os.IsNotExist(err) {
		// Template is not cached
	} else {
		// Some other error
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Cached template: %t\n", cachedTemplate)

	// Download template
	if !cachedTemplate {
		rc, _, err := client.Repositories.DownloadReleaseAsset(context.Background(), GH_OWNER, GH_REPO, templateAssetId, http.DefaultClient)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer rc.Close()

		err = os.MkdirAll(filepath.Dir(cachePath), 0755)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		tf, err := os.Create(cachePath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer tf.Close()

		_, err = io.Copy(tf, rc)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// Create app from template
	err = os.MkdirAll(vTDir, 0755)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Unzip template into target directory
	archive, err := zip.OpenReader(cachePath)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	tlFolder := archive.File[0].Name
	for _, f := range archive.File {
		filePath := filepath.Join(vTDir, strings.Replace(f.Name, tlFolder, "", 1))

		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}

	// Must rename the gitignore template file
	err = os.Rename(filepath.Join(vTDir, "gitignore.template"), filepath.Join(vTDir, ".gitignore"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Git
	useGit := gitInitFlag
	if !cmd.Flags().Changed("git-init") && !yesFlag {
		// TODO(jgmw): Prompt for git
	}
	fmt.Printf("Use Git: %t\n", useGit)

	if useGit {
		err = setupGit(cmd, vTDir)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// TODO(jgmw): Yarn install - maybe
	// TODO(jgmw): Generate types

	printEpilogue()
}

func checkNode() error {
	nodes := which.All("node")
	if len(nodes) == 0 {
		return fmt.Errorf("node not found")
	}
	fmt.Println("Node found:")
	for _, node := range nodes {
		fmt.Printf("  %s\n", node)
	}

	// TODO(jgmw): Check node installation source

	// Check node version
	nodeVer, err := exec.Command("node", "-v").Output()
	if err != nil {
		return err
	}
	fmt.Printf("Node version: %s\n", nodeVer)

	// TODO(jgmw): Check node version against minimum version

	return nil
}

func checkYarn() error {
	yarns := which.All("yarn")
	if len(yarns) == 0 {
		fmt.Println("Yarn not found")
		os.Exit(1)
	}
	fmt.Println("Yarn found:")
	for _, yarn := range yarns {
		fmt.Printf("  %s\n", yarn)
	}

	// TODO(jgmw): Check yarn installation source

	// TODO(jgmw): Execute yarn version check
	yarnVer, err := exec.Command("yarn", "-v").Output()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("Yarn version: %s\n", yarnVer)

	// TODO(jgmw): Check yarn version against minimum version

	return nil
}

func setupGit(cmd *cobra.Command, vTDir string) error {
	commitMsg := commitMessageFlag
	if !cmd.Flags().Changed("commit-message") && !yesFlag {
		// TODO(jgmw): Prompt for commit message
	}

	// Perform git init
	r, err := git.PlainInit(vTDir, false)
	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			fmt.Println("Git repository already exists, skipping git init")
			return nil
		}
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	_, err = w.Add(".")
	if err != nil {
		return err
	}

	// Perform initial commit
	_, err = w.Commit(commitMsg, &git.CommitOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("Commit message: %s\n", commitMsg)
	return nil
}

func printInto() {
	// TODO(jgmw): Use terminal width to determine how many dashes to print
	yellow := color.New(color.FgYellow)
	yellow.Println(strings.Repeat("-", 66))
	fmt.Printf("%16s🌲⚡️ %s ⚡️🌲\n", "", ("Welcome to RedwoodJS!"))
	yellow.Println(strings.Repeat("-", 66))
}

func printEpilogue() {
	// TODO(jgmw): Use terminal width to determine how many dashes to print
	green := color.New(color.FgGreen)

	fmt.Println()
	green.Println("Thanks for trying out Redwood!")
	fmt.Println()
	fmt.Println(" ⚡️ Get up and running fast with this Quick Start guide: https://redwoodjs.com/quick-start")
	fmt.Println()
	fmt.Println("Fire it up! 🚀")
	fmt.Println()
	green.Println("  cd <your-app-name>")
	green.Println("  yarn install")
	green.Println("  yarn rw dev")
	fmt.Println()
}

func validateTargetDirectory(tDir string, overwrite bool) (string, error) {
	absTDir, err := filepath.Abs(tDir)
	if err != nil {
		return "", err
	}

	// Get path stats
	fi, err := os.Stat(absTDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Target directory doesn't exist
			return absTDir, nil
		}
		return "", err
	}

	// Check if target directory is a directory
	if !fi.IsDir() {
		return "", fmt.Errorf("target directory is not a directory")
	}

	// Check if target directory is empty
	if overwrite {
		return absTDir, nil
	}

	isEmpty, err := IsEmpty(absTDir)
	if err != nil {
		return "", err
	}
	if !isEmpty {
		return "", fmt.Errorf("target directory is not empty")
	}

	return absTDir, nil
}

// See: https://stackoverflow.com/a/30708914
func IsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}
