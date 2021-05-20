#!/usr/bin/env bash

echo
echo "Running Lint (Linux specific)"
GOOS=linux golangci-lint run || exit 1
echo "Running Lint (Windows specific)"
GOOS=windows golangci-lint run || exit 1
echo "Running Lint (FreeBSD specific)"
GOOS=freebsd golangci-lint run || exit 1

