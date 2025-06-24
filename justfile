# Justfile for ViewNet (Cross-platform)
# Run with: just <recipe-name>

# Set shell for Windows
set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

# Variables for cross-platform compatibility
exec_ext := if os() == "windows" { ".exe" } else { "" }
null_redirect := if os() == "windows" { ">$null" } else { ">/dev/null" }

# Default recipe - shows available commands
default:
    just --list

# Build the application
build:
    @echo "🔨 Building viewnet..."
    go build -o viewnet{{exec_ext}} .
    @echo "✅ Build complete"

# Build for different platforms (Windows)
[windows]
build-all:
    @echo "🔨 Building for Linux and Windows..."
    -powershell "if (-not (Test-Path 'dist')) { mkdir dist }"
    powershell "`$env:GOOS='windows'; `$env:GOARCH='amd64'; go build -o 'dist/viewnet-windows-amd64.exe' ."
    powershell "`$env:GOOS='linux'; `$env:GOARCH='amd64'; go build -o 'dist/viewnet-linux-amd64' ."
    @echo "✅ Multi-platform build complete"

# Build for different platforms (Unix)
[unix]
build-all:
    @echo "🔨 Building for Linux and Windows..."
    mkdir -p dist
    GOOS=windows GOARCH=amd64 go build -o dist/viewnet-windows-amd64.exe .
    GOOS=linux GOARCH=amd64 go build -o dist/viewnet-linux-amd64 .
    @echo "✅ Multi-platform build complete"

# Clean build artifacts (Windows)
[windows]
clean:
    @echo "🧹 Cleaning build artifacts..."
    -powershell "if (Test-Path 'viewnet.exe') { Remove-Item 'viewnet.exe' -Force }"
    -powershell "if (Test-Path 'dist') { Remove-Item 'dist' -Recurse -Force }"
    -powershell "Get-ChildItem -Filter '*.csv' -ErrorAction SilentlyContinue | Remove-Item -Force"
    -powershell "Remove-Item 'coverage.out', 'coverage.html', 'cpu.prof', 'mem.prof' -Force -ErrorAction SilentlyContinue"
    @echo "✅ Clean complete"

# Clean build artifacts (Unix)
[unix]
clean:
    @echo "🧹 Cleaning build artifacts..."
    -rm -f viewnet
    -rm -rf dist
    -rm -f *.csv test_*.csv bench_test_*.csv
    -rm -f coverage.out coverage.html cpu.prof mem.prof
    @echo "✅ Clean complete"

# Install dependencies
deps:
    @echo "📦 Installing dependencies..."
    go mod download
    go mod tidy
    @echo "✅ Dependencies installed"

# Run all tests
test:
    @echo "🧪 Running all tests..."
    go test -v .
    @echo "✅ Tests complete"

# Run tests with coverage
test-cover:
    @echo "🔍 Running tests with coverage..."
    go test -cover -coverprofile=coverage.out .
    go tool cover -html=coverage.out -o coverage.html
    @echo "✅ Coverage report generated: coverage.html"

# Run only unit tests
test-unit:
    @echo "📋 Running unit tests..."
    go test -v -run="^Test[^I]" .
    @echo "✅ Unit tests complete"

# Run only integration tests  
test-integration:
    @echo "🚀 Running integration tests..."
    go test -v -run="TestIntegration" .
    @echo "✅ Integration tests complete"

# Run benchmarks
bench:
    @echo "📊 Running benchmarks..."
    go test -run=^$$ -bench=. -benchmem -benchtime=3s
    @echo "✅ Benchmarks complete"

# Run performance tests
perf:
    @echo "⚡ Running performance analysis..."
    go test -run=^$$ -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof
    @echo "✅ Performance profiles generated"

# Run tests in short mode (faster)
test-short:
    @echo "⚡ Running tests in short mode..."
    go test -short .
    @echo "✅ Short tests complete"

# Run race condition detection
test-race:
    @echo "🏃 Running race condition tests..."
    go test -race .
    @echo "✅ Race tests complete"

# Format code
fmt:
    @echo "🎨 Formatting code..."
    go fmt ./...
    @echo "✅ Code formatted"

# Lint code
lint:
    @echo "🔍 Linting code..."
    go vet ./...
    @echo "✅ Linting complete"

# Run static analysis
analyze:
    @echo "🔬 Running static analysis..."
    go vet ./...
    -golangci-lint run || echo "golangci-lint not installed"
    @echo "✅ Analysis complete"

# Check for security issues
security:
    @echo "🔒 Running security checks..."
    -gosec ./... || echo "gosec not installed (go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)"
    @echo "✅ Security check complete"

# Generate documentation
docs:
    @echo "📚 Generating documentation..."
    go doc -all . > docs.txt
    @echo "✅ Documentation generated: docs.txt"

# Run the application with IP scan
run-ips:
    @echo "🚀 Running IP discovery scan..."
    ./viewnet{{exec_ext}} -ips

# Run the application with port scan
run-ports ports="22,80,443":
    @echo "🚀 Running port scan on ports: {{ports}}..."
    ./viewnet{{exec_ext}} -p {{ports}}

# Run the application in non-interactive mode
run-csv target="192.168.1.0/24" output="scan_results.csv":
    @echo "🚀 Running non-interactive scan..."
    ./viewnet{{exec_ext}} -subnet {{target}} -csv {{output}}

# Development workflow - format, lint, test
dev: fmt lint test
    @echo "✅ Development checks complete"

# Full CI workflow
ci: clean deps fmt lint test-race test-cover bench
    @echo "✅ CI pipeline complete"

# Release workflow
release version: clean deps test-race build-all
    @echo "🚀 Release {{version}} ready"
    @echo "📦 Binaries available in dist/"

# View test coverage in browser (Windows)
[windows]
show-coverage: test-cover
    @echo "🌐 Opening coverage report..."
    start coverage.html

# View test coverage in browser (Unix)
[unix]
show-coverage: test-cover
    @echo "🌐 Opening coverage report..."
    -xdg-open coverage.html 2>/dev/null || open coverage.html 2>/dev/null || echo "No browser opener found"

# View CPU profile (requires go tool pprof)
profile-cpu: perf
    @echo "📊 Opening CPU profile..."
    go tool pprof cpu.prof

# View memory profile
profile-mem: perf
    @echo "📊 Opening memory profile..."
    go tool pprof mem.prof

# Quick smoke test (Windows)
[windows]
smoke: build
    @echo "💨 Running smoke test..."
    @echo "Testing help output..."
    ./viewnet.exe -h >$null
    @echo "Testing CSV mode..."
    ./viewnet.exe -ips -csv smoke_test.csv >$null
    @powershell "if (Test-Path 'smoke_test.csv') { echo '✅ CSV file created' } else { echo '❌ CSV file not created' }"
    @powershell "if (Test-Path 'smoke_test.csv') { Remove-Item 'smoke_test.csv' -Force }"
    @echo "✅ Smoke test complete"

# Quick smoke test (Unix)
[unix]
smoke: build
    @echo "💨 Running smoke test..."
    @echo "Testing help output..."
    ./viewnet -h >/dev/null
    @echo "Testing CSV mode..."
    ./viewnet -ips -csv smoke_test.csv >/dev/null
    @if [ -f smoke_test.csv ]; then echo "✅ CSV file created"; else echo "❌ CSV file not created"; fi
    @rm -f smoke_test.csv
    @echo "✅ Smoke test complete"

# Install development tools
install-tools:
    @echo "🛠️ Installing development tools..."
    go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    @echo "✅ Development tools installed"

# Show project statistics (Windows)
[windows]
stats:
    @echo "📊 Project Statistics"
    @echo "===================="
    @powershell "Write-Host 'Go files:' (Get-ChildItem '*.go').Count"
    @powershell "Write-Host 'Test files:' (Get-ChildItem '*_test.go').Count"
    @powershell "Write-Host 'Total Go files:' (Get-ChildItem '*.go' -Recurse).Count"

# Show project statistics (Unix)
[unix]
stats:
    @echo "📊 Project Statistics"
    @echo "===================="
    @echo "Go files: $(find . -name '*.go' ! -name '*_test.go' | wc -l)"
    @echo "Test files: $(find . -name '*_test.go' | wc -l)"
    @echo "Lines of code: $(find . -name '*.go' ! -name '*_test.go' -exec cat {} \\; | wc -l)"
    @echo "Lines of tests: $(find . -name '*_test.go' -exec cat {} \\; | wc -l)"
    @echo "Total lines: $(find . -name '*.go' -exec cat {} \\; | wc -l)"
