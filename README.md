# ai-commit

![Version](https://img.shields.io/badge/version-1.3.0-blue.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/in-jun/ai-commit)](https://goreportcard.com/report/github.com/in-jun/ai-commit)

AI-powered Git commit message generator using Google Gemini. Analyzes staged changes, understands project context, and creates Conventional Commits-compliant messages.

## Quick Start

```bash
# Install
go install github.com/in-jun/ai-commit@latest

# Configure API key
ai-commit init  # Enter your Google AI Studio API key

# Use
git add .
ai-commit
```

## Requirements

- Go 1.21+
- Git 2.0+
- [Google AI Studio API key](https://aistudio.google.com/app/apikey)

## Installation

### macOS/Linux

```bash
# Install Go (if needed)
brew install go  # macOS
sudo apt install golang-go  # Ubuntu

# Install ai-commit
go install github.com/in-jun/ai-commit@latest

# Add to PATH
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.zshrc  # macOS
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc  # Linux
```

### Windows

```powershell
# Install Go from https://go.dev/dl/
# Install ai-commit
go install github.com/in-jun/ai-commit@latest

# Add to PATH
$env:Path += ";$(go env GOPATH)\bin"
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$(go env GOPATH)\bin", [EnvironmentVariableTarget]::User)
```

## Configuration

Configure via `~/.ai-commit/config.yaml`:

```yaml
api_key: "your-gemini-api-key"
model: "gemini-2.0-flash" # Available: 2.0-flash, 2.0-flash-lite, 2.5-flash, 2.5-flash-lite
max_diff_size: 10000 # Maximum diff size in bytes
history_depth: 5 # Previous commits to analyze
color_enabled: true # Colored output

templates:
  - prefix: "feat"
    description: "New feature"
  - prefix: "fix"
    description: "Bug fix"
  - prefix: "docs"
    description: "Documentation"
  - prefix: "refactor"
    description: "Code refactoring"
  # Add custom types as needed
```

## Usage

### Basic Workflow

```bash
git add <files>
ai-commit

# Review generated message
# Y/Enter: Accept and commit
# E: Edit message
# N: Cancel
```

### Commands

| Command          | Description              |
| ---------------- | ------------------------ |
| `ai-commit`      | Generate commit message  |
| `ai-commit init` | Initialize configuration |
| `ai-commit -v`   | Show version             |
| `ai-commit -h`   | Show help                |

## Model Selection

Choose models based on your needs:

| Model                   | Performance | Accuracy | Daily Limit | Use Case          |
| ----------------------- | ----------- | -------- | ----------- | ----------------- |
| `gemini-2.0-flash`      | ⚡⚡⚡      | ★★★★     | 1,500       | Default, balanced |
| `gemini-2.0-flash-lite` | ⚡⚡⚡⚡    | ★★       | 1,500       | Simple commits    |
| `gemini-2.5-flash`      | ⚡⚡        | ★★★★★    | 1,500       | Latest, best      |
| `gemini-2.5-flash-lite` | ⚡⚡⚡⚡    | ★★★      | 1,500       | Fast & efficient  |

Update model in config:

```yaml
model: "gemini-2.5-flash" # For best accuracy
```

## Features

- **Context-aware**: Analyzes previous commits for consistent style
- **Multi-model support**: Choose from 4 Gemini models
- **Interactive editing**: Review and modify messages before committing
- **Conventional Commits**: Follows industry standards
- **Language detection**: Adapts to project's commit language
- **Customizable templates**: Define project-specific commit types

## Example Output

```
=== Generated Commit Message ===
feat: implement JWT authentication middleware

Add token-based authentication system for API endpoints
- Create token validation middleware
- Implement protected route handling
- Add 401/403 error responses
- Include token refresh mechanism

What would you like to do?
[Y]es: Commit with this message
[E]dit: Edit the message
[N]o: Cancel commit
Choice [Y/e/n]:
```

## Advanced Configuration

### Custom Commit Types

```yaml
templates:
  - prefix: "security"
    description: "Security updates"
  - prefix: "perf"
    description: "Performance improvements"
  - prefix: "deploy"
    description: "Deployment changes"
```

### Editor Setup

```bash
# Set preferred editor
export EDITOR="vim"           # Linux/macOS
$env:EDITOR="code --wait"     # Windows PowerShell
```

## Troubleshooting

### Command Not Found

```bash
# Verify installation
go version
ls $(go env GOPATH)/bin/ai-commit

# Fix PATH
export PATH=$PATH:$(go env GOPATH)/bin
```

### API Key Issues

```bash
# Check configuration
cat ~/.ai-commit/config.yaml

# Reinitialize
ai-commit init

# Use environment variable
export API_KEY="your-key"
```

### Model Errors

- Rate limit exceeded: Switch to `gemini-2.0-flash-lite` for higher limits
- Complex changes failing: Use `gemini-2.5-flash` for better analysis
- Timeout issues: Check network connection and API status

## Best Practices

1. **Atomic commits**: Stage related changes together
2. **Review messages**: Always verify AI-generated content
3. **Model selection**: Use appropriate model for change complexity
4. **Custom templates**: Align with team conventions

---

For issues and updates: [github.com/in-jun/ai-commit](https://github.com/in-jun/ai-commit)
