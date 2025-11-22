---
layout: default
title: Commands
nav_order: 2
has_children: true
permalink: /commands
---

# Commands

gvm provides a simple command-line interface for managing multiple Go versions. All commands follow a consistent pattern and provide helpful error messages.

## Command Overview

| Command | Description | Example |
|---------|-------------|---------|
| [install](install) | Install a specific Go version | `gvm install go1.19` |
| [list](list) | List installed and available versions | `gvm list --all` |
| [use](use) | Switch to a specific version | `gvm use go1.19` |
| [uninstall](uninstall) | Remove an installed version | `gvm uninstall go1.18` |
| [update](update) | Update gvm to the latest version | `gvm update` |

## Common Usage Patterns

### Installing and Using a New Version
```bash
# Install the latest stable version
gvm install --latest

# Or install a specific version
gvm install go1.19

# Set it as the default
gvm use go1.19

# Verify it's active
go version
```

### Managing Multiple Versions
```bash
# List all installed versions
gvm list

# See what's available to install
gvm list --all

# Switch between versions
gvm use go1.20
gvm use go1.19

# Clean up old versions
gvm uninstall go1.18
```

## Global Flags

All commands support the following global flags:

| Flag | Description |
|------|-------------|
| `--help` | Show help for any command |
| `--version` | Show version information |

## Error Handling

All commands provide clear, actionable error messages for common issues:
- Network connectivity problems
- Invalid version formats
- Missing dependencies
- Permission issues

## Getting Help

For help with any command, use:
```bash
gvm [command] --help
```

For example:
```bash
gvm install --help
gvm list --help
