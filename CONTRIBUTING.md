# Contributing to ShellSage

First off, thank you for considering contributing to ShellSage! It's people like you that make ShellSage such a great tool.

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the issue list as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

* **Use a clear and descriptive title**
* **Describe the exact steps which reproduce the problem**
* **Provide specific examples to demonstrate the steps**
* **Describe the behavior you observed after following the steps**
* **Explain which behavior you expected to see instead and why**
* **Include screenshots and animated GIFs if possible**
* **Include your environment details** (OS, Go version, etc.)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

* **Use a clear and descriptive title**
* **Provide a step-by-step description of the suggested enhancement**
* **Provide specific examples to demonstrate the steps**
* **Describe the current behavior and expected behavior**
* **Explain why this enhancement would be useful**

### Pull Requests

* Fill in the required template
* Follow the Go styleguides
* Include appropriate test cases
* End all files with a newline
* Avoid platform-specific code

## Styleguides

### Git Commit Messages

* Use the present tense ("Add feature" not "Added feature")
* Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
* Limit the first line to 72 characters or less
* Reference issues and pull requests liberally after the first line

### Go Styleguide

* Follow [Effective Go](https://golang.org/doc/effective_go)
* Use `gofmt` to format your code
* Run `go vet ./...` before committing
* Write comments for exported functions
* Keep functions small and focused

### Documentation Styleguide

* Use [GitHub Flavored Markdown](https://guides.github.com/features/mastering-markdown/)
* Reference function names and constants with backticks
* Include code examples where appropriate

## Local Development

1. Fork the repo
2. Clone your fork: `git clone https://github.com/mahmud-r-farhan/ShellSage`
3. Create a new branch: `git checkout -b feature/your-feature`
4. Make your changes
5. Run tests: `go test ./...`
6. Commit: `git commit -am 'Add some feature'`
7. Push: `git push origin feature/your-feature`
8. Create a Pull Request

## Running Tests

```bash
go test ./...
go test -v ./...
go test -cover ./...
```

## Building

```bash
go build -o shellsage
go build -o shellsage.exe  # Windows
```

## Additional Notes

### Issue and Pull Request Labels

* `bug` - Something isn't working
* `enhancement` - New feature or request
* `documentation` - Improvements or additions to documentation
* `good first issue` - Good for newcomers
* `help wanted` - Extra attention is needed

## Thank You!

Your contributions to ShellSage are greatly appreciated. Thank you for your interest in improving this project!
