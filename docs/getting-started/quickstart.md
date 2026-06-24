# Quickstart

[← Back to the manual index](../README.md)

This walkthrough takes you from a clone to an initialized Daedalus workspace.

## 1. Build the binary

```sh
go build -o daedalus ./cmd/daedalus
```

See [Installation](installation.md) for prerequisites.

## 2. Verify it runs

```sh
./daedalus --version
# daedalus 0.1.0-dev
```

## 3. Initialize a workspace

Move to the repository you want to manage and create the `.daedalus/`
workspace:

```sh
cd /path/to/your-repo
daedalus init
# Created Daedalus workspace at .daedalus from scratch.
# Seeded factory workflow "sdd-default" at .daedalus/workflows/sdd-default.yaml.
```

This creates the canonical `.daedalus/` structure — the backend-agnostic source
of truth for your project's AI scaffolding — and seeds the default SDD workflow,
`sdd-default`, so you start with a ready-to-use pipeline. It is safe to run in an
existing repository: it never touches files outside `.daedalus/`.

## Where to go next

- [Initializing a workspace](../guide/initializing-a-workspace.md) — what `init` creates and its options.
- [Command line](../guide/command-line.md) — the interface, version, and help.
- [Configuration](../guide/configuration.md) — logging and environment variables.
