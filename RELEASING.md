# Release Process

This document describes how to release new versions of Sift and its adapter modules.

## Multi-Module Repository Structure

Sift uses a multi-module repository structure with independent versioning:

```
github.com/nisimpson/sift                    (main module)
github.com/nisimpson/sift/thru/dynamodb      (DynamoDB adapter)
github.com/nisimpson/sift/thru/sql           (SQL adapter)
github.com/nisimpson/sift/thru/exprlang      (Expr-lang adapter)
github.com/nisimpson/sift/thru/jsonapi       (JSON:API marshaler)
```

Each module can be versioned independently, allowing:
- Breaking changes in one adapter without affecting others
- Feature releases for specific adapters
- Bug fixes for individual modules

## Versioning Strategy

### Semantic Versioning

All modules follow [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** version: Incompatible API changes
- **MINOR** version: Backwards-compatible functionality additions
- **PATCH** version: Backwards-compatible bug fixes

### When to Version Together

Release all modules together when:
- Core library (main module) has breaking changes that affect adapters
- Major feature additions that span multiple modules
- Initial releases (v1.0.0, v2.0.0, etc.)

### When to Version Independently

Release modules independently when:
- Bug fix in a specific adapter
- New feature in one adapter only
- Adapter-specific improvements or optimizations

## Release Process

### 1. Pre-Release Checklist

**CRITICAL: Update Submodule Dependencies**

Before releasing, you MUST update the submodule go.mod files to reference the actual version instead of using `replace` directives:

```bash
# For each submodule (thru/dynamodb, thru/sql, thru/exprlang, thru/jsonapi):
# 1. Remove the replace directive
# 2. Update the require version to match the release version

# Example for v1.0.0 release:
# In thru/sql/go.mod, change:
#   require github.com/nisimpson/sift v0.0.0
#   replace github.com/nisimpson/sift => ../..
# To:
#   require github.com/nisimpson/sift v1.0.0
```

**Note:** The `replace` directives are used during development to reference the local main module. They must be removed before publishing or users won't be able to install the modules.

**Standard Pre-Release Checks:**

```bash
# Ensure working directory is clean
git status

# Run all checks
make check

# Verify packages are ready
make publish-check
```

### 2. Update Documentation

- Update CHANGELOG.md with release notes
- Update README.md if API changed
- Update adapter-specific READMEs if needed

### 3. Create Tags

#### Option A: Release All Modules (Synchronized Release)

Use this for major releases or when changes affect multiple modules:

```bash
# Create tags for all modules with the same version
make tag-release VERSION=1.0.0

# This creates:
#   v1.0.0                    (main module)
#   thru/dynamodb/v1.0.0      (DynamoDB adapter)
#   thru/sql/v1.0.0           (SQL adapter)
#   thru/exprlang/v1.0.0      (Expr-lang adapter)
#   thru/jsonapi/v1.0.0       (JSON:API marshaler)
```

#### Option B: Release Main Module Only

```bash
make tag-release VERSION=1.0.0 MODULE=.
# Creates: v1.0.0
```

#### Option C: Release Specific Adapter

```bash
# Release SQL adapter only
make tag-release VERSION=1.0.1 MODULE=thru/sql
# Creates: thru/sql/v1.0.1

# Release DynamoDB adapter only
make tag-release VERSION=1.2.0 MODULE=thru/dynamodb
# Creates: thru/dynamodb/v1.2.0
```

### 4. Push Tags

```bash
# Push all tags
git push origin --tags

# Or push specific tag
git push origin v1.0.0
git push origin thru/sql/v1.0.1
```

### 5. Verify Release

After pushing tags, verify on pkg.go.dev:

- Main module: https://pkg.go.dev/github.com/nisimpson/sift@v1.0.0
- SQL adapter: https://pkg.go.dev/github.com/nisimpson/sift/thru/sql@v1.0.1
- DynamoDB adapter: https://pkg.go.dev/github.com/nisimpson/sift/thru/dynamodb@v1.2.0

Note: It may take a few minutes for pkg.go.dev to index new versions.

## Example Release Scenarios

### Scenario 1: Initial Release

All modules start at v1.0.0:

```bash
make tag-release VERSION=1.0.0
git push origin --tags
```

### Scenario 2: Bug Fix in SQL Adapter

Only the SQL adapter needs a patch release:

```bash
# Fix bug in thru/sql/adapter.go
git commit -m "fix(sql): correct LIMIT clause generation"

# Release only SQL adapter
make tag-release VERSION=1.0.1 MODULE=thru/sql
git push origin thru/sql/v1.0.1
```

Users can update:
```bash
go get github.com/nisimpson/sift/thru/sql@v1.0.1
```

### Scenario 3: New Feature in Core Library

Core library gets a minor version bump, adapters may or may not need updates:

```bash
# Add new feature to core
git commit -m "feat: add new filter operation"

# If adapters need updates to support the feature
make tag-release VERSION=1.1.0

# If adapters work without changes, release main module only
make tag-release VERSION=1.1.0 MODULE=.
```

### Scenario 4: Breaking Change in Core Library

Major version bump for core and all adapters:

```bash
# Make breaking change
git commit -m "feat!: refactor API with breaking changes"

# Release all modules with new major version
make tag-release VERSION=2.0.0
git push origin --tags
```

### Scenario 5: Independent Adapter Versions

After initial release, adapters diverge:

```
Main module:        v1.0.0
DynamoDB adapter:   v1.0.0 → v1.1.0 (new feature)
SQL adapter:        v1.0.0 → v1.0.1 (bug fix) → v1.0.2 (bug fix)
Exprlang adapter:   v1.0.0 (no changes)
```

```bash
# Release DynamoDB feature
make tag-release VERSION=1.1.0 MODULE=thru/dynamodb
git push origin thru/dynamodb/v1.1.0

# Release SQL bug fixes
make tag-release VERSION=1.0.1 MODULE=thru/sql
git push origin thru/sql/v1.0.1

# Later, another SQL bug fix
make tag-release VERSION=1.0.2 MODULE=thru/sql
git push origin thru/sql/v1.0.2
```

## Managing Tags

### List All Tags

```bash
make list-tags
```

### Delete a Tag

```bash
# Delete local tag
make delete-tag TAG=v1.0.0

# Delete remote tag
git push origin :refs/tags/v1.0.0
```

### View Tag Details

```bash
# Show tag message and commit
git show v1.0.0

# List all tags with messages
git tag -n
```

## User Perspective

### Installing Specific Versions

Users can install specific versions of each module:

```bash
# Install main module
go get github.com/nisimpson/sift@v1.0.0

# Install specific adapter versions
go get github.com/nisimpson/sift/thru/sql@v1.0.1
go get github.com/nisimpson/sift/thru/dynamodb@v1.2.0

# Install latest versions
go get github.com/nisimpson/sift@latest
go get github.com/nisimpson/sift/thru/sql@latest
```

### Checking Installed Versions

```bash
# View all dependencies
go list -m all | grep sift

# Output example:
# github.com/nisimpson/sift v1.0.0
# github.com/nisimpson/sift/thru/sql v1.0.1
# github.com/nisimpson/sift/thru/dynamodb v1.2.0
```

## Best Practices

1. **Always run `make publish-check` before releasing**
   - Ensures tests pass
   - Verifies go.mod files are tidy
   - Checks working directory is clean

2. **Use descriptive commit messages**
   - Follow conventional commits format
   - Makes it easier to generate changelogs

3. **Document breaking changes**
   - Update CHANGELOG.md
   - Add migration guide if needed
   - Use `!` in commit message: `feat!: breaking change`

4. **Test before tagging**
   - Run full test suite: `make test`
   - Run linters: `make lint`
   - Test in a real project if possible

5. **Coordinate major releases**
   - When core library has breaking changes, update all adapters
   - Release all modules together with same major version
   - Provide migration guide

6. **Keep adapters compatible**
   - Adapters should work with a range of core versions when possible
   - Use go.mod `require` directives appropriately
   - Document minimum required core version

## Troubleshooting

### Tag Already Exists

```bash
# Delete local tag
git tag -d v1.0.0

# Delete remote tag
git push origin :refs/tags/v1.0.0

# Recreate tag
make tag-release VERSION=1.0.0
```

### Wrong Version Tagged

```bash
# Delete incorrect tag locally and remotely
make delete-tag TAG=v1.0.0
git push origin :refs/tags/v1.0.0

# Create correct tag
make tag-release VERSION=1.0.1
git push origin v1.0.1
```

### Module Not Appearing on pkg.go.dev

1. Ensure tag is pushed to GitHub
2. Wait a few minutes for indexing
3. Manually request indexing: Visit `https://pkg.go.dev/github.com/nisimpson/sift@v1.0.0`
4. Check that go.mod file exists in the module directory

## References

- [Go Modules Reference](https://go.dev/ref/mod)
- [Multi-Module Repositories](https://go.dev/wiki/Modules#faqs--multi-module-repositories)
- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
