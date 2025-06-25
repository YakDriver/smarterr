# smarterr command-line interface reference

The smarterr CLI helps you check and inspect your smarterr Config files. Use it to check for errors and view merged configs. See if your configuration looks as expected before embedding or deploying.

---

## Installation

Install the smarterr CLI using Go:

```sh
go install github.com/YakDriver/smarterr/cmd/smarterr@latest
```

This command places the `smarterr` binary in your `$GOPATH/bin` or `$HOME/go/bin` directory.

---

## Usage

Run the CLI from your project directory or anywhere you want to inspect or check a smarterr Config.

```sh
smarterr <command> [flags]
```

---

## Commands

### Config

Show the effective merged Config for a directory. This command helps you debug layered Config resolution and see what smarterr, the library, will use at a given path.

```sh
smarterr config --base-dir /path/to/project --start-dir /path/to/project/internal/service
```

**Flags:**

- `--base-dir`, `-b`: Directory, perhaps parent directory, where you use `go:embed` in your project (for example, `internal`). If not set, the command looks at current directory and won't merge parent or global configs.
- `--start-dir`, `-d`: Directory where code using smarterr lives (default: current directory). Typically, set this to where an error occurs.
- `--debug`, `-D`: Enable debug output (shows internal merging and raw Config).

**Example:**

```sh
smarterr config -b /path/to/project -d /path/to/project/internal/service
```

---

### Check

Check the merged smarterr Config for parse errors, missing fields, and other issues. This command validates your configuration and reports any problems.

```sh
smarterr check --base-dir /path/to/project --start-dir /path/to/project/internal/service
```

**Flags:**

- `--base-dir`, `-b`: Directory, perhaps parent directory, where you use `go:embed` in your project (for example, `internal`). If not set, the command looks at current directory and won't merge parent or global configs.
- `--start-dir`, `-d`: Directory where code using smarterr lives (default: current directory).
- `--debug`, `-D`: Enable debug output (shows internal diagnostics).
- `--quiet`, `-q`: Output just errors (suppresses merged Config and warnings).
- `--silent`, `-S`: No output, just the exit code (non-zero if errors).

**Example:**

Go to the directory where your code calls the `smarterr` library (for example, where you use `AddError`). If you use `go:embed` two directories up from there, and you want to merge configs the same way the smarterr library does and check them, run this command:

```sh
smarterr check -b ../..
```

---

## Tips

- Always set `--base-dir` to the directory where you use `go:embed` in your application for correct Config layering. This way the CLI will work the same as the smarterr library.
- Use `--debug` to see detailed merging and diagnostics output.
- Run `smarterr check` in CI to catch Config errors before deployment.

---

## See also

- [Config schema](schema.md)
- [Layered configs & merging](layering.md)
- [Diagnostics & fallbacks](diagnostics.md)
