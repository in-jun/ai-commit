package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	genai "github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

const Version = "1.1.0"

type Config struct {
	ApiKey       string     `yaml:"api_key"`
	MaxDiffSize  int        `yaml:"max_diff_size"`
	HistoryDepth int        `yaml:"history_depth"`
	Templates    []Template `yaml:"templates"`
	ColorEnabled bool       `yaml:"color_enabled"`
}

type Template struct {
	Prefix      string `yaml:"prefix"`
	Description string `yaml:"description"`
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
)

func colorize(text string, color string, config *Config) string {
	if !config.ColorEnabled {
		return text
	}
	return color + text + colorReset
}

func printWithSpinner(message string) func() {
	if !isTerminal() {
		fmt.Print(message)
		return func() { fmt.Println("Done!") }
	}

	done := make(chan bool)
	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-done:
				fmt.Printf("\r%s Done!\n", message)
				return
			default:
				fmt.Printf("\r%s %s", message, frames[i])
				i = (i + 1) % len(frames)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return func() {
		done <- true
	}
}

func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

type GitOperation struct {
	ctx    context.Context
	config *Config
}

func NewGitOperation(ctx context.Context, config *Config) *GitOperation {
	return &GitOperation{ctx: ctx, config: config}
}

func (g *GitOperation) getGitDiff(maxDiffSize int) (string, error) {
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		return "", fmt.Errorf("not a git repository")
	}

	statusCmd := exec.Command("git", "diff", "--cached", "--name-only")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to check staged changes: %v", err)
	}

	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		return "", fmt.Errorf("no staged changes")
	}

	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %v", err)
	}

	if len(output) > maxDiffSize {
		return "", fmt.Errorf("diff size exceeds maximum allowed size of %d bytes", maxDiffSize)
	}

	return string(output), nil
}

func (g *GitOperation) getRecentCommits(count int) ([]string, error) {
	cmd := exec.Command("git", "log", fmt.Sprintf("-%d", count), "--pretty=format:%s")
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "does not have any commits yet") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get recent commits: %v", err)
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	return strings.Split(string(output), "\n"), nil
}

func (g *GitOperation) commitChanges(message string) error {
	stopSpinner := printWithSpinner("Committing changes...")
	defer stopSpinner()

	cmd := exec.Command("git", "commit", "-m", message)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit: %s", string(output))
	}
	return nil
}

type ConfigManager struct {
	config *Config
	path   string
}

func NewConfigManager() (*ConfigManager, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	config := &Config{
		MaxDiffSize:  10000,
		HistoryDepth: 5,
		ColorEnabled: true,
		Templates: []Template{
			{Prefix: "feat", Description: "Add new feature"},
			{Prefix: "fix", Description: "Fix a bug"},
			{Prefix: "chore", Description: "Maintenance tasks"},
			{Prefix: "docs", Description: "Documentation changes"},
			{Prefix: "style", Description: "Code style changes"},
			{Prefix: "refactor", Description: "Code refactoring"},
			{Prefix: "perf", Description: "Performance improvements"},
			{Prefix: "test", Description: "Add or modify tests"},
			{Prefix: "build", Description: "Build system changes"},
			{Prefix: "ci", Description: "CI configuration changes"},
		},
	}

	if envKey := os.Getenv("API_KEY"); envKey != "" {
		config.ApiKey = envKey
	}

	return &ConfigManager{
		config: config,
		path:   path,
	}, nil
}

func (cm *ConfigManager) Load() error {
	data, err := os.ReadFile(cm.path)
	if err != nil {
		if os.IsNotExist(err) {
			return cm.SaveDefault()
		}
		return fmt.Errorf("failed to read config file: %v", err)
	}

	if err := yaml.Unmarshal(data, cm.config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	return cm.Validate()
}

func (cm *ConfigManager) SaveDefault() error {
	data, err := yaml.Marshal(cm.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(cm.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	fmt.Printf("Created default config at %s\n", cm.path)
	return nil
}

func (cm *ConfigManager) Validate() error {
	if cm.config.ApiKey == "" {
		return fmt.Errorf("API key is required. Set it in config file or API_KEY environment variable")
	}
	if cm.config.MaxDiffSize <= 0 {
		return fmt.Errorf("max_diff_size must be positive")
	}
	if cm.config.HistoryDepth <= 0 {
		return fmt.Errorf("history_depth must be positive")
	}

	prefixRegex := regexp.MustCompile(`^[a-z]+$`)
	for _, t := range cm.config.Templates {
		if t.Prefix == "" || len(t.Prefix) > 10 || !prefixRegex.MatchString(t.Prefix) {
			return fmt.Errorf("invalid template prefix: %s", t.Prefix)
		}
		if t.Description == "" || len(t.Description) > 50 {
			return fmt.Errorf("invalid template description for prefix %s", t.Prefix)
		}
	}
	return nil
}

type CommitMessageGenerator struct {
	client *genai.Client
	config *Config
}

func NewCommitMessageGenerator(ctx context.Context, config *Config) (*CommitMessageGenerator, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(config.ApiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create AI client: %v", err)
	}

	return &CommitMessageGenerator{
		client: client,
		config: config,
	}, nil
}

func (g *CommitMessageGenerator) Generate(ctx context.Context, diff string, history []string) (string, error) {
	stopSpinner := printWithSpinner("Generating commit message...")
	defer stopSpinner()

	model := g.client.GenerativeModel("gemini-1.5-pro")
	model.SetTemperature(0.7)

	templateInfo := make([]string, len(g.config.Templates))
	for i, t := range g.config.Templates {
		templateInfo[i] = fmt.Sprintf("%s: %s", t.Prefix, t.Description)
	}

	prompt := fmt.Sprintf(`You are a highly skilled Git commit message generator. Analyze the given changes and generate a clear, concise, and conventional commit message.

CONTEXT:
Previous commit messages for this repository:
%s

CHANGES TO COMMIT:
%s

COMMIT TYPES:
%s

IMPORTANT LANGUAGE INSTRUCTION:
1. First, analyze the language used in previous commits
2. Generate the commit message ENTIRELY in that exact same language
3. Match the writing style and terminology of previous commits precisely

REQUIREMENTS:
1. Format: Must follow conventional commit format:
   <type>: <description>
   
   <body>
2. Length: 
   - Header: Maximum 72 characters
   - Body: Wrap at 72 characters
3. Type:
   - MUST use one of the commit types provided in the list above
   - Do not include any other type
4. Grammar:
   - Use present tense 
   - Use imperative mood
   - No period at the end
5. Content:
   - Be specific about what changed
   - Explain why in the body
   - Focus on the change's impact
   - Use consistent terminology
6. Body (Required):
   - Separate from header with blank line
   - Explain motivation for change
   - Contrast with previous behavior
   - Use bullet points for multiple items

EXAMPLES (these are in English but your output should match the language of previous commits):
feat: add user authentication to API endpoints

Implement JWT-based authentication to secure API routes.
- Add token validation middleware
- Restrict access to sensitive endpoints
- Improve overall system security

fix: resolve memory leak in background worker

Worker was not properly clearing temporary files after
processing. Now ensures cleanup happens even when tasks fail.

INSTRUCTIONS:
1. Analyze previous commits to determine the language being used
2. Understand the core changes being made
3. Choose appropriate commit type
4. Write header and body IN THE SAME LANGUAGE as previous commits
5. Verify all requirements are met
6. Return ONLY the complete commit message, nothing else

Generate the commit message now:`,
		strings.Join(history, "\n"),
		diff,
		strings.Join(templateInfo, "\n"))

	maxRetries := 3
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Second * time.Duration(1<<uint(attempt)))
		}

		resp, err := model.GenerateContent(ctx, genai.Text(prompt))
		if err != nil {
			lastErr = err
			continue
		}

		if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
			message := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
			if message != "" {
				return strings.TrimSpace(message), nil
			}
		}

		lastErr = fmt.Errorf("empty response received")
	}

	return "", fmt.Errorf("failed after %d retries: %v", maxRetries, lastErr)
}

func (g *CommitMessageGenerator) Close() {
	g.client.Close()
}

type CLI struct {
	config *Config
}

func NewCLI(config *Config) *CLI {
	return &CLI{config: config}
}

func (cli *CLI) getCommitAction(message string) (string, error) {
	for {
		fmt.Println("\n" + colorize("=== Generated Commit Message ===", colorBlue, cli.config))
		fmt.Printf("%s\n\n", colorize(message, colorGreen, cli.config))

		fmt.Println(colorize("What would you like to do?", colorCyan, cli.config))
		fmt.Println(colorize("[Y]es: ", colorGreen, cli.config) + "Commit with this message")
		fmt.Println(colorize("[E]dit: ", colorYellow, cli.config) + "Edit the message")
		fmt.Println(colorize("[N]o: ", colorRed, cli.config) + "Cancel commit")
		fmt.Print(colorize("Choice [Y/e/n]: ", colorCyan, cli.config))

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		input = strings.ToLower(strings.TrimSpace(input))
		switch input {
		case "", "y", "yes":
			return message, nil
		case "e", "edit":
			return cli.editMessage(message)
		case "n", "no":
			return "", nil
		default:
			fmt.Println(colorize("Invalid choice. Please try again.", colorRed, cli.config))
		}
	}
}

func (cli *CLI) editMessage(originalMessage string) (string, error) {
	tmpfile, err := os.CreateTemp("", "commit-msg-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(originalMessage); err != nil {
		return "", fmt.Errorf("failed to write to temporary file: %v", err)
	}
	tmpfile.Close()

	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		cmd := exec.Command("git", "config", "--get", "core.editor")
		if output, err := cmd.Output(); err == nil {
			editor = strings.TrimSpace(string(output))
		}
	}
	if editor == "" {
		for _, ed := range []string{"vim", "nano", "vi"} {
			if _, err := exec.LookPath(ed); err == nil {
				editor = ed
				break
			}
		}
	}
	if editor == "" {
		return "", fmt.Errorf("no text editor found. Please set EDITOR environment variable")
	}

	cmd := exec.Command(editor, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run editor: %v", err)
	}

	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited message: %v", err)
	}

	editedMessage := strings.TrimSpace(string(content))
	if editedMessage == "" {
		return "", fmt.Errorf("commit message cannot be empty")
	}

	return editedMessage, nil
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help":
			printHelp()
			return
		case "-v", "--version":
			fmt.Printf("ai-commit version %s\n", Version)
			return
		case "init":
			if err := initializeConfig(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			return
		}
	}

	if err := run(); err != nil {
		printError(err)
	}
}

func run() error {
	ctx := context.Background()

	configManager, err := NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %v", err)
	}

	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	gitOps := NewGitOperation(ctx, configManager.config)

	diff, err := gitOps.getGitDiff(configManager.config.MaxDiffSize)
	if err != nil {
		return err
	}

	history, err := gitOps.getRecentCommits(configManager.config.HistoryDepth)
	if err != nil {
		history = []string{}
	}

	generator, err := NewCommitMessageGenerator(ctx, configManager.config)
	if err != nil {
		return fmt.Errorf("failed to initialize message generator: %v", err)
	}
	defer generator.Close()

	message, err := generator.Generate(ctx, diff, history)
	if err != nil {
		return fmt.Errorf("failed to generate commit message: %v", err)
	}

	cli := NewCLI(configManager.config)
	finalMessage, err := cli.getCommitAction(message)
	if err != nil {
		return fmt.Errorf("failed to process commit action: %v", err)
	}

	if finalMessage == "" {
		fmt.Println(colorize("Commit cancelled", colorYellow, configManager.config))
		return nil
	}

	if err := gitOps.commitChanges(finalMessage); err != nil {
		return err
	}

	fmt.Println(colorize("Successfully committed changes!", colorGreen, configManager.config))
	return nil
}

func printHelp() {
	helpText := `Usage: ai-commit [command] [options]

Generate AI-powered commit messages for your staged changes.

Commands:
  init        Initialize or update configuration
  help        Show this help message
  version     Show version information

Options:
  -h, --help     Show this help message
  -v, --version  Show version information

Configuration:
  Config file location: ~/.ai-commit/config.yaml
  Environment variables:
    API_KEY    API key for AI service

Examples:
  1. Initialize configuration:
     ai-commit init
  
  2. Stage your changes:
     git add .
  
  3. Generate commit message:
     ai-commit

Tips:
  - Make atomic commits (one logical change per commit)
  - Stage only related changes together
  - Review the generated message before confirming
  - Use the config file to customize templates and settings`

	fmt.Println(helpText)
}

func printError(err error) {
	errMsg := ""
	switch err.Error() {
	case "not a git repository":
		errMsg = colorize("Error: Not a git repository", colorRed, &Config{ColorEnabled: true}) + "\n" +
			"Tip: Initialize a git repository with 'git init'"
	case "no staged changes":
		errMsg = colorize("Error: No staged changes found", colorRed, &Config{ColorEnabled: true}) + "\n" +
			"Tip: Stage your changes with 'git add <files>' first"
	case "API key is required. Set it in config file or API_KEY environment variable":
		errMsg = colorize("Error: API key is missing", colorRed, &Config{ColorEnabled: true}) + "\n" +
			"Tip: Run 'ai-commit init' to set up configuration or set API_KEY environment variable"
	default:
		errMsg = colorize(fmt.Sprintf("Error: %v", err), colorRed, &Config{ColorEnabled: true})
	}
	fmt.Println(errMsg)
}

func initializeConfig() error {
	configManager, err := NewConfigManager()
	if err != nil {
		return err
	}

	fmt.Print("Enter your API key (or press Enter to use API_KEY environment variable): ")
	reader := bufio.NewReader(os.Stdin)
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read API key: %v", err)
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey != "" {
		configManager.config.ApiKey = apiKey
	}

	if err := configManager.SaveDefault(); err != nil {
		return err
	}

	fmt.Println(colorize("Configuration initialized successfully!", colorGreen, configManager.config))
	fmt.Printf("Edit %s to customize settings\n", configManager.path)
	return nil
}

func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}

	configDir := filepath.Join(home, ".ai-commit")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %v", err)
	}

	return filepath.Join(configDir, "config.yaml"), nil
}
