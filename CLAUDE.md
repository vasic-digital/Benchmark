# CLAUDE.md - Benchmark Module

## Overview

`digital.vasic.benchmark` is a Go module for LLM benchmark evaluation: running standardized
benchmarks (MMLU, HumanEval, GSM8K, SWE-Bench, MBPP, LMSYS, HellaSwag, MATH, custom), comparing
providers, and tracking performance metrics.

**Module**: `digital.vasic.benchmark` (Go 1.24+)

## Build & Test

```bash
go build ./...
go test ./... -count=1 -race
```

## Package Structure

| Package | Purpose |
|---------|---------|
| `benchmark` | Core: benchmark runner, types, integration adapters, metrics |
