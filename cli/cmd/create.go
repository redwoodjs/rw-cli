package cmd

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	bighornFlag       bool
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
	createCmd.Flags().BoolVar(&bighornFlag, "bighorn", false, "Commit message for the initial commit")

	// TODO(jgmw): Add flag for yarn install based on yarn version?
}

func runCreate(cmd *cobra.Command, args []string) {
	slog.Debug("create command", slog.String("positional", args[0]))
	printInto()

	// Check node
	err := checkNode()
	if err != nil {
		slog.Error("node check failed", slog.String("error", err.Error()))
		fmt.Println(err)
		os.Exit(1)
	}
	slog.Debug("node check passed")

	// Check yarn
	err = checkYarn()
	if err != nil {
		slog.Error("yarn check failed", slog.String("error", err.Error()))
		fmt.Println(err)
		os.Exit(1)
	}
	slog.Debug("yarn check passed")

	// Target directory
	tDir := "./redwood-app"
	if len(args) == 0 {
		// TODO: Prompt for target directory
	} else {
		tDir = args[0]
	}
	vTDir, err := validateTargetDirectory(tDir, overwriteFlag)
	if err != nil {
		slog.Error("target directory validation failed", slog.String("error", err.Error()))
		fmt.Println(err)
		os.Exit(1)
	}
	slog.Debug("target directory validation passed", slog.String("target", vTDir))
	fmt.Printf("Target directory: %s\n", vTDir)

	// TS or JS
	useTS := typescriptFlag
	if !cmd.Flags().Changed("typescript") && !yesFlag {
		// TODO(jgmw): Prompt for TS or JS
		slog.Debug("typescript flag unset, prompting")
	}
	slog.Debug("typescript flag", slog.Bool("typescript", useTS))
	fmt.Printf("Use TypeScript: %t\n", useTS)

	// Get the latest release
	client := github.NewClient(nil)
	token := os.Getenv("RW_GITHUB_TOKEN")
	if token != "" {
		slog.Debug("RW_GITHUB_TOKEN is set", slog.String("token", token[len(token)-4:]))
		client = client.WithAuthToken(token)
	}
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), GH_OWNER, GH_REPO)
	if err != nil {
		slog.Error("failed to get latest release", slog.String("error", err.Error()))
		fmt.Println(err)
		os.Exit(1)
	}
	rTag := release.GetTagName()
	slog.Debug("latest release", slog.String("tag", rTag))
	fmt.Printf("Latest version: %s\n", rTag)

	epochName := "arapaho"
	if bighornFlag {
		epochName = "bighorn"
	}
	slog.Debug("epoch choice", slog.String("name", epochName))

	templateAssetName := epochName + "_js.zip"
	if useTS {
		templateAssetName = epochName + "_ts.zip"
	}
	slog.Debug("template asset name", slog.String("name", templateAssetName))

	templateAssetId := int64(0)
	for _, asset := range release.Assets {
		if asset.GetName() == templateAssetName {
			templateAssetId = asset.GetID()
		}
	}

	zipName := rTag + "_" + templateAssetName
	slog.Debug("zip name", slog.String("name", zipName))

	// Check if we already have the template cached
	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("failed to get user home directory", slog.String("error", err.Error()))
		fmt.Println(err)
		os.Exit(1)
	}
	cachePath := filepath.Join(homeDir, ".rw", "templates", zipName)
	cachedTemplate := false
	if _, err := os.Stat(cachePath); err == nil {
		cachedTemplate = true
	} else if os.IsNotExist(err) {
		// Template is not cached
	} else {
		// Some other error
		slog.Error("failed to check if template is cached", slog.String("error", err.Error()))
		fmt.Println(err)
		os.Exit(1)
	}
	slog.Debug("template cache path", slog.String("path", cachePath))
	slog.Debug("template cache status", slog.Bool("status", cachedTemplate))
	fmt.Printf("Cached template: %t\n", cachedTemplate)

	// Download template
	if !cachedTemplate {
		sTime := time.Now()

		// TODO(jgmw): Consider downloading via a HTTP endpoint but could be less robust?
		rc, _, err := client.Repositories.DownloadReleaseAsset(context.Background(), GH_OWNER, GH_REPO, templateAssetId, http.DefaultClient)
		if err != nil {
			slog.Error("failed to download template", slog.String("error", err.Error()))
			fmt.Println(err)
			os.Exit(1)
		}
		defer rc.Close()

		err = os.MkdirAll(filepath.Dir(cachePath), 0755)
		if err != nil {
			slog.Error("failed to create template cache directory", slog.String("error", err.Error()))
			fmt.Println(err)
			os.Exit(1)
		}
		tf, err := os.Create(cachePath)
		if err != nil {
			slog.Error("failed to create template cache file", slog.String("error", err.Error()))
			fmt.Println(err)
			os.Exit(1)
		}
		defer tf.Close()

		_, err = io.Copy(tf, rc)
		if err != nil {
			slog.Error("failed to write template cache file", slog.String("error", err.Error()))
			fmt.Println(err)
			os.Exit(1)
		}
		slog.Debug("template downloaded and saved", slog.Duration("duration", time.Since(sTime)))
	}

	// Create app from template
	err = os.MkdirAll(vTDir, 0755)
	if err != nil {
		slog.Error("failed to create target directory", slog.String("error", err.Error()))
		fmt.Println(err)
		os.Exit(1)
	}
	slog.Debug("target directory created", slog.String("path", vTDir))

	// Unzip template into target directory
	archive, err := zip.OpenReader(cachePath)
	if err != nil {
		slog.Error("failed to open template zip", slog.String("error", err.Error()))
		fmt.Println(err)
		os.Exit(1)
	}
	defer archive.Close()
	slog.Debug("template zip opened", slog.String("path", cachePath))

	for _, f := range archive.File {
		filePath := filepath.Join(vTDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			slog.Error("failed to create directory for file in template", slog.String("error", err.Error()))
			fmt.Println(err)
			os.Exit(1)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			slog.Error("failed to create file in template", slog.String("error", err.Error()))
			fmt.Println(err)
			os.Exit(1)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			slog.Error("failed to open file in template", slog.String("error", err.Error()))
			fmt.Println(err)
			os.Exit(1)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			slog.Error("failed to write file in template", slog.String("error", err.Error()))
			fmt.Println(err)
			os.Exit(1)
		}

		dstFile.Close()
		fileInArchive.Close()

		slog.Debug("template entry written", slog.String("path", filePath))
	}

	// Must rename the gitignore template file if it exists
	gitignoreTemplatePath := filepath.Join(vTDir, "gitignore.template")
	if _, err := os.Stat(gitignoreTemplatePath); err == nil {
		err = os.Rename(filepath.Join(vTDir, "gitignore.template"), filepath.Join(vTDir, ".gitignore"))
		if err != nil {
			slog.Error("failed to rename gitignore template", slog.String("error", err.Error()))
			fmt.Println(err)
			os.Exit(1)
		}
		slog.Debug("gitignore template renamed")
	} else if !errors.Is(err, os.ErrNotExist) {
		slog.Error("failed to check if gitignore template exists", slog.String("error", err.Error()))
		fmt.Println(err)
		os.Exit(1)
	}

	// Git
	useGit := gitInitFlag
	if !cmd.Flags().Changed("git-init") && !yesFlag {
		slog.Debug("git flag unset, prompting")
		// TODO(jgmw): Prompt for git
	}
	slog.Debug("git flag", slog.Bool("git", useGit))
	fmt.Printf("Use Git: %t\n", useGit)

	if useGit {
		err = setupGit(cmd, vTDir)
		if err != nil {
			slog.Error("failed to setup git", slog.String("error", err.Error()))
			fmt.Println(err)
			os.Exit(1)
		}
		slog.Debug("git setup complete")
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
		return fmt.Errorf("yarn not found")
	}
	fmt.Println("Yarn found:")
	for _, yarn := range yarns {
		fmt.Printf("  %s\n", yarn)
	}

	// TODO(jgmw): Check yarn installation source

	// TODO(jgmw): Execute yarn version check
	yarnVer, err := exec.Command("yarn", "-v").Output()
	if err != nil {
		return err
	}
	fmt.Printf("Yarn version: %s\n", yarnVer)

	// TODO(jgmw): Check yarn version against minimum version

	return nil
}

func setupGit(cmd *cobra.Command, vTDir string) error {
	commitMsg := commitMessageFlag
	if !cmd.Flags().Changed("commit-message") && !yesFlag {
		slog.Debug("commit message flag unset, prompting")
		// TODO(jgmw): Prompt for commit message
	}
	slog.Debug("commit message flag", slog.String("message", commitMsg))

	// Perform git init
	r, err := git.PlainInit(vTDir, false)
	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			slog.Warn("git repository already exists, skipping git init")
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
	slog.Debug("initial commit complete", slog.String("message", commitMsg))
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
