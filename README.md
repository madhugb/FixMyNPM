# fixmynpm

A command-line tool for diagnosing and fixing `.npmrc` configuration issues across your projects.

`fixmynpm` checks your global `~/.npmrc`, scans your repos for project-level `.npmrc` files, audits them against security best practices, and applies fixes — including incident-response tooling to hunt for compromised packages across your `node_modules`.

---

## Install

```sh
go install github.com/madhugb/fixmynpm@latest
```

Or build from source:

```sh
git clone https://github.com/madhugb/fixmynpm.git
cd fixmynpm
go build -o fixmynpm .
```

---

## Commands

### `doctor`

Check your global `~/.npmrc` for security and configuration issues.

```sh
fixmynpm doctor
```

**Example output:**

```
Running doctor on ~/.npmrc ...
Found 2 issue(s):

  [error]  rule=registry-insecure-http              key=registry line 1
           registry uses insecure http — upgrade to https to prevent MITM attacks
           Fix: registry=https://registry.npmjs.org/

  [error]  rule=strict-ssl-disabled                 key=strict-ssl line 2
           strict-ssl=false disables TLS certificate verification — severe security risk
           Fix: strict-ssl=true
```

---

### `scan`

Find `.npmrc` files or search for installed packages across a directory tree.

```sh
# Find all .npmrc files under home directory
fixmynpm scan

# Find all .npmrc files under a specific path
fixmynpm scan --root /Users/me/projects

# Incident response: include node_modules, flag bundled .npmrc files
fixmynpm scan --incident --root /Users/me/projects

# Find a specific package across all node_modules
fixmynpm scan --package lodash

# Glob matching — find all lodash-* packages
fixmynpm scan --package "lodash*"

# Find all @babel scoped packages
fixmynpm scan --package "@babel/*"

# Find a package in a specific vulnerable version range
fixmynpm scan --package lodash --version ">=4.17.0 <4.17.21"

# Only show results where the package has a bundled .npmrc (supply chain red flag)
fixmynpm scan --package "ua-parser-js" --npmrc
```

**Flags:**

| Flag | Description |
|---|---|
| `--root <dir>` | Root directory to scan (default: `~`) |
| `--package <glob>` | Package name or glob to search in `node_modules` |
| `--version <range>` | Semver constraint to filter matches (e.g. `">=1.0.0 <2.0.0"`) |
| `--npmrc` | Only show packages that have a bundled `.npmrc` |
| `--incident` | Incident mode: include `node_modules` in `.npmrc` walk, flag suspicious files |

**Typical incident response workflow:**

```sh
# Step 1 — do I have the compromised package?
fixmynpm scan --package ua-parser-js --version ">=0.7.29 <=0.7.30"

# Step 2 — does it have a bundled .npmrc (registry hijack)?
fixmynpm scan --package ua-parser-js --npmrc

# Step 3 — sweep everything for hidden .npmrc files
fixmynpm scan --incident --root ~/projects
```

---

### `audit`

Scan a directory tree for all `.npmrc` files and report every issue found.

```sh
fixmynpm audit
fixmynpm audit --root /Users/me/projects
```

**Flags:**

| Flag | Description |
|---|---|
| `--root <dir>` | Root directory to scan (default: `~`) |

---

### `fixer`

Apply recommended fixes to `.npmrc` files. Supports dry-run mode.

```sh
# Preview what would be changed
fixmynpm fixer --dry-run

# Apply all fixable issues
fixmynpm fixer

# Fix files under a specific path
fixmynpm fixer --root /Users/me/projects
```

**Flags:**

| Flag | Description |
|---|---|
| `--root <dir>` | Root directory to scan (default: `~`) |
| `--dry-run` | Preview fixes without writing any files |

---

## Security Rules

Rules are sourced from the [npm Security Best Practices](https://github.com/lirantal/npm-security-best-practices) guide.

| Rule | Severity | Description |
|---|---|---|
| `registry-invalid-url` | error | Registry value is not a valid URL |
| `registry-insecure-http` | error | Registry uses `http://` instead of `https://` — MITM risk |
| `strict-ssl-disabled` | error | `strict-ssl=false` disables TLS verification |
| `auth-token-empty` | error | Auth token key exists but has no value |
| `auth-token-in-project-npmrc` | error | Auth token found in a project-level `.npmrc` — at risk of being committed |
| `ignore-scripts-disabled` | warning | `ignore-scripts` not set to `true` — post-install scripts can run arbitrary code |
| `allow-git-unrestricted` | warning | `allow-git` not set to `none` — git deps bypass registry security |
| `scope-missing-registry` | warning | A scoped package has config but no `@scope:registry` mapping — dependency confusion risk |
| `min-release-age-missing` | info | `min-release-age` not set — no cooldown before installing newly published packages |
| `save-exact-disabled` | info | `save-exact=false` — floating semver allows unexpected upgrades |

Rules with a suggested fix can be auto-applied by `fixmynpm fixer`. Rules marked `error` with no fix (e.g. `auth-token-in-project-npmrc`) require manual action.

---

## Contributing

Pull requests are welcome. To add a new rule, add a `check*` function in `internal/doctor/doctor.go` and call it from `Check()`. Each rule returns an `[]Issue` with a `Rule` ID, `Severity`, `Message`, and optional `Fix` string.

---

## License

MIT License

Copyright (c) 2026 Madhu G B

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
