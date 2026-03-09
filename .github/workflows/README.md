# GitHub Actions Workflows

This directory contains GitHub Actions workflows for automated testing, linting, and releasing.

## Workflows

### CI (`ci.yml`)

**Triggers**: Push to `main`/`develop`, Pull Requests

**Jobs**:
- **Test**: Runs tests on Go 1.21, 1.22, and 1.23
  - Executes `make test` with race detector
  - Uploads coverage to Codecov (optional)
- **Lint**: Runs golangci-lint with configured rules
- **Build**: Builds all packages to verify compilation
- **Verify**: Checks formatting, dependencies, and runs go vet

**Purpose**: Ensures code quality and compatibility across Go versions

### Release (`release.yml`)

**Triggers**: Push of version tags (`v*` or `thru/*/v*`)

**Jobs**:
- **Release**: Creates GitHub releases automatically
  - Detects module (main or submodule) from tag format
  - Generates changelog from git commits
  - Creates release with installation instructions
  - Links to pkg.go.dev documentation
- **Notify**: Sends success notification

**Tag Formats**:
- Main module: `v1.0.0`
- Submodules: `thru/sql/v1.0.1`, `thru/dynamodb/v1.2.0`

**Purpose**: Automates release creation when tags are pushed

### Nightly (`nightly.yml`)

**Triggers**: Daily at 2 AM UTC, Manual dispatch

**Jobs**:
- **Test**: Runs full test suite with coverage
- **Test Latest Go**: Tests with latest stable Go version
- **Security**: Runs security scanners (gosec, govulncheck)

**Purpose**: Catches issues early and monitors security

### Publish Check (`publish-check.yml`)

**Triggers**: Pull Requests to `main`

**Jobs**:
- **Check**: Validates packages are ready for release
  - Runs `make publish-check`
  - Verifies all go.mod files are tidy
  - Comments on PR with results

**Purpose**: Ensures PRs are release-ready before merging

## Usage

### Running CI on Pull Requests

CI runs automatically on all PRs. Ensure all checks pass before merging:

```bash
# Locally verify before pushing
make check
```

### Creating a Release

#### Option 1: Using Makefile (Recommended)

```bash
# Create tags locally
make tag-release VERSION=1.0.0

# Push tags to trigger release workflow
git push origin --tags
```

#### Option 2: Manual Tag Creation

```bash
# Main module
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Submodule
git tag -a thru/sql/v1.0.1 -m "Release thru/sql v1.0.1"
git push origin thru/sql/v1.0.1
```

The release workflow will:
1. Run tests to verify the release
2. Generate changelog from commits
3. Create GitHub release with:
   - Installation instructions
   - Changelog
   - Links to documentation
4. Notify on success

### Manual Workflow Triggers

Some workflows support manual triggering:

```bash
# Trigger nightly workflow manually
gh workflow run nightly.yml
```

Or via GitHub UI: Actions → Select workflow → Run workflow

## Secrets and Permissions

### Required Secrets

- `GITHUB_TOKEN`: Automatically provided by GitHub Actions
  - Used for creating releases and commenting on PRs
  - No configuration needed

### Optional Secrets

- `CODECOV_TOKEN`: For uploading coverage to Codecov
  - Add in repository settings: Settings → Secrets → Actions
  - Get token from https://codecov.io

### Permissions

Workflows use these permissions:
- `contents: write` - For creating releases
- `pull-requests: write` - For commenting on PRs

## Customization

### Changing Go Versions

Edit `ci.yml` matrix:

```yaml
strategy:
  matrix:
    go-version: ['1.21', '1.22', '1.23']  # Add/remove versions
```

### Adjusting Nightly Schedule

Edit `nightly.yml` cron expression:

```yaml
schedule:
  - cron: '0 2 * * *'  # 2 AM UTC daily
  # Examples:
  # - cron: '0 */6 * * *'  # Every 6 hours
  # - cron: '0 0 * * 0'    # Weekly on Sunday
```

### Customizing Release Notes

Edit `release.yml` changelog generation:

```yaml
- name: Generate changelog
  run: |
    # Customize git log format
    git log --pretty=format:"- %s (%h)" "$PREV_TAG..$TAG"
```

### Adding Slack/Discord Notifications

Add notification step to workflows:

```yaml
- name: Notify Slack
  uses: slackapi/slack-github-action@v1
  with:
    webhook-url: ${{ secrets.SLACK_WEBHOOK }}
    payload: |
      {
        "text": "Release ${{ github.ref }} created!"
      }
```

## Troubleshooting

### CI Failing on Formatting

```bash
# Fix locally
make fmt
git commit -am "chore: format code"
git push
```

### Release Not Created

Check:
1. Tag format is correct (`v1.0.0` or `thru/sql/v1.0.0`)
2. Tag was pushed to GitHub: `git push origin --tags`
3. Workflow logs in Actions tab
4. Repository permissions allow workflow to create releases

### Tests Failing in CI but Passing Locally

Common causes:
- Different Go versions
- Race conditions (CI uses `-race` flag)
- Missing dependencies

Debug:
```bash
# Run with same flags as CI
make test

# Test specific Go version
go test -race ./...
```

### Coverage Upload Failing

This is non-critical (marked `continue-on-error: true`). To fix:
1. Add `CODECOV_TOKEN` secret
2. Or remove codecov step if not needed

## Best Practices

1. **Always run `make check` before pushing**
   - Catches issues locally before CI

2. **Keep workflows fast**
   - Use caching (already configured)
   - Run expensive checks in nightly only

3. **Test releases locally first**
   ```bash
   make publish-check
   make tag-release VERSION=1.0.0
   # Review tags before pushing
   git tag -l
   ```

4. **Monitor workflow runs**
   - Check Actions tab regularly
   - Fix failures promptly

5. **Update dependencies**
   - Review nightly dependency checks
   - Update Go versions as needed

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Go GitHub Actions](https://github.com/actions/setup-go)
- [golangci-lint Action](https://github.com/golangci/golangci-lint-action)
- [Codecov Action](https://github.com/codecov/codecov-action)
