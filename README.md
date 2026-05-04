# sensors

Build instructions for the `sns` binary using the local `dev` build tool.

## Prerequisites

- Go installed
- Run commands from the repository root

## 1) Build the `dev` tool first

Use `Makefile` to compile `dev` for your current host platform:

```bash
make build
```

This creates `./dev`.

## 2) Build `sns` with `./dev`

Your most common build:

```bash
./dev build --os linux --arch amd64 --version 0.2.0
```

Output binary is written to `dist/sns`.

## Cross-compilation examples

Build Linux ARM64:

```bash
./dev build --os linux --arch arm64 --version 0.2.0
```

If you run on Linux amd64 and need to cross-compile to ARM64 in the native path, you can also use:

```bash
./dev build --os linux --arch amd64 --cross-os linux --cross-arch arm64 --version 0.2.0
```

Build macOS ARM64:

```bash
./dev build --os darwin --arch arm64 --version 0.2.0
```