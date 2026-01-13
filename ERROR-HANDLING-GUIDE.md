# Error Handling Guide

This document outlines the error handling patterns and best practices used in the drift-analysis-cli project.

## Table of Contents

1. [Overview](#overview)
2. [Current Implementation](#current-implementation)
3. [Best Practices](#best-practices)
4. [Examples](#examples)
5. [References](#references)

## Overview

The drift-analysis-cli follows Go and Cobra best practices for error handling, emphasizing:

- **Explicit error returns** over exceptions
- **Error wrapping** to preserve context and stack traces
- **Proper Cobra integration** with `RunE` and `SilenceUsage`
- **Structured logging** for diagnostics

## Current Implementation

### 1. Cobra Command Integration

All commands use `RunE` instead of `Run` to enable proper error handling:

```go
var gceCmd = &cobra.Command{
    Use:          "gce",
    Short:        "Analyze GCE instances for configuration drift",
    RunE:         runGCEAnalysis,
    SilenceUsage: true, // Don't show usage on runtime errors
}
```

**Key points:**
- `RunE` returns `error` instead of having no return value
- `SilenceUsage: true` prevents usage text from appearing on runtime errors (API failures, network issues, etc.)
- Usage is still shown for flag validation errors on the root command (automatically handled by Cobra)
- **Note:** Flag parsing errors on subcommands with `SilenceUsage: true` will not show usage, but the error message is still clear

### 2. Error Wrapping

All errors are wrapped using `fmt.Errorf` with the `%w` verb (Go 1.13+):

```go
configData, err := os.ReadFile(cfgFile)
if err != nil {
    logger.Error("Failed to read config file", err, map[string]interface{}{
        "file": cfgFile,
    })
    return fmt.Errorf("failed to read config file: %w", err)
}
```

**Benefits:**
- Preserves the original error for inspection with `errors.Is()` and `errors.As()`
- Adds context at each layer of the call stack
- Enables error chain unwrapping with `errors.Unwrap()`

### 3. Structured Logging

Errors are logged with context using the `logger` package:

```go
logger.Error("Failed to create GCE analyzer", err, map[string]interface{}{
    "baseline": baseline.Name,
    "project":  project,
})
```

**Features:**
- Supports both text and JSON logging formats
- Includes timestamps, log levels, and contextual metadata
- Separate from error returns (logs to stderr, errors returned to caller)

### 4. Error Flow

```
┌─────────────────────────────────────────────────────┐
│ User runs command                                   │
└────────────────┬────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────┐
│ PersistentPreRun: Initialize logger                │
└────────────────┬────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────┐
│ RunE: Execute command logic                         │
│   - Validate inputs                                 │
│   - Call analyzer functions                         │
│   - Wrap errors with context                        │
└────────────────┬────────────────────────────────────┘
                 │
                 ▼
           ┌─────┴──────┐
           │            │
        Success       Error
           │            │
           ▼            ▼
    Return nil   Return wrapped error
                        │
                        ▼
           ┌────────────────────────────┐
           │ Execute(): Log and exit(1) │
           └────────────────────────────┘
```

## Best Practices

### DO ✅

1. **Use RunE for all Cobra commands**
   ```go
   var myCmd = &cobra.Command{
       RunE: runMyCommand,
       SilenceUsage: true,
   }
   ```

2. **Wrap errors with context**
   ```go
   if err != nil {
       return fmt.Errorf("failed to process instance %s: %w", instanceName, err)
   }
   ```

3. **Log errors before returning**
   ```go
   if err != nil {
       logger.Error("Operation failed", err, map[string]interface{}{
           "operation": "discover",
           "project":   projectID,
       })
       return fmt.Errorf("failed to discover instances: %w", err)
   }
   ```

4. **Return errors early**
   ```go
   if config == nil {
       return fmt.Errorf("config is nil")
   }
   
   if err := validate(config); err != nil {
       return fmt.Errorf("invalid config: %w", err)
   }
   
   // Continue with main logic
   ```

5. **Use errors.Is() for sentinel errors**
   ```go
   if errors.Is(err, sql.ErrNoRows) {
       // Handle not found case
   }
   ```

6. **Use errors.As() for custom error types**
   ```go
   var apiErr *googleapi.Error
   if errors.As(err, &apiErr) {
       logger.Warn("API error", map[string]interface{}{
           "code":    apiErr.Code,
           "message": apiErr.Message,
       })
   }
   ```

### DON'T ❌

1. **Don't use Run instead of RunE**
   ```go
   // ❌ Bad
   var myCmd = &cobra.Command{
       Run: func(cmd *cobra.Command, args []string) {
           if err := doSomething(); err != nil {
               fmt.Println(err)  // No way to signal error to caller
               os.Exit(1)        // Bypasses PostRun hooks
           }
       },
   }
   ```

2. **Don't use errors.New() when wrapping**
   ```go
   // ❌ Bad - loses original error
   if err != nil {
       return errors.New("failed to connect")
   }
   
   // ✅ Good - preserves error chain
   if err != nil {
       return fmt.Errorf("failed to connect: %w", err)
   }
   ```

3. **Don't call os.Exit() in command handlers**
   ```go
   // ❌ Bad - bypasses error handling and cleanup
   if err != nil {
       fmt.Println(err)
       os.Exit(1)
   }
   
   // ✅ Good - returns error to caller
   if err != nil {
       return fmt.Errorf("operation failed: %w", err)
   }
   ```

4. **Don't ignore errors**
   ```go
   // ❌ Bad
   analyzer.Close()
   
   // ✅ Good
   if err := analyzer.Close(); err != nil {
       logger.Warn("Failed to close analyzer", map[string]interface{}{
           "error": err.Error(),
       })
   }
   ```

5. **Don't log and return the same error multiple times**
   ```go
   // ❌ Bad - logs at every level
   func a() error {
       if err := b(); err != nil {
           logger.Error("b failed", err)
           return err
       }
       return nil
   }
   
   func b() error {
       if err := c(); err != nil {
           logger.Error("c failed", err)
           return err
       }
       return nil
   }
   
   // ✅ Good - log once at appropriate level
   func a() error {
       if err := b(); err != nil {
           logger.Error("Operation failed", err)  // Log at top level
           return fmt.Errorf("operation failed: %w", err)
       }
       return nil
   }
   
   func b() error {
       if err := c(); err != nil {
           return fmt.Errorf("b failed: %w", err)  // Just wrap, don't log
       }
       return nil
   }
   ```

## Examples

### Example 1: Command with Validation

```go
func runAnalysis(cmd *cobra.Command, args []string) error {
    // Validate inputs (error will show usage automatically)
    if baselineName == "" {
        return fmt.Errorf("baseline name is required")
    }
    
    // Read config
    config, err := loadConfig(cfgFile)
    if err != nil {
        logger.Error("Failed to load config", err)
        return fmt.Errorf("failed to load config: %w", err)
    }
    
    // Create analyzer
    analyzer, err := gce.NewAnalyzer(ctx)
    if err != nil {
        logger.Error("Failed to create analyzer", err)
        return fmt.Errorf("failed to create analyzer: %w", err)
    }
    defer analyzer.Close()
    
    // Perform analysis
    report, err := analyzer.AnalyzeDrift(ctx, instances, baseline)
    if err != nil {
        logger.Error("Analysis failed", err, map[string]interface{}{
            "baseline": baseline.Name,
            "instances": len(instances),
        })
        return fmt.Errorf("analysis failed: %w", err)
    }
    
    // Output results
    fmt.Println(report.Format())
    return nil
}
```

### Example 2: Custom Error Types

```go
// Define custom error type
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

// Use in validation
func validateBaseline(b *Baseline) error {
    if b.Name == "" {
        return &ValidationError{
            Field:   "name",
            Message: "baseline name is required",
        }
    }
    
    if b.VMConfig == nil {
        return &ValidationError{
            Field:   "vm_config",
            Message: "vm_config is required",
        }
    }
    
    return nil
}

// Check in caller
func processBaseline(b *Baseline) error {
    if err := validateBaseline(b); err != nil {
        var valErr *ValidationError
        if errors.As(err, &valErr) {
            logger.Warn("Validation failed", map[string]interface{}{
                "field": valErr.Field,
                "issue": valErr.Message,
            })
        }
        return fmt.Errorf("invalid baseline: %w", err)
    }
    
    // Continue processing...
    return nil
}
```

### Example 3: Partial Failures

```go
func analyzeMultipleProjects(ctx context.Context, projects []string) error {
    var errs []error
    
    for _, project := range projects {
        if err := analyzeProject(ctx, project); err != nil {
            logger.Warn("Project analysis failed", map[string]interface{}{
                "project": project,
                "error":   err.Error(),
            })
            errs = append(errs, fmt.Errorf("project %s: %w", project, err))
            continue  // Continue with other projects
        }
    }
    
    if len(errs) > 0 {
        // Use errors.Join (Go 1.20+) to combine multiple errors
        return fmt.Errorf("failed to analyze %d projects: %w", 
            len(errs), errors.Join(errs...))
    }
    
    return nil
}
```

## Error Categories

### Input Validation Errors
- Missing required flags
- Invalid flag values
- Bad configuration format

**Handling:** Return immediately, allow usage to be shown (don't use SilenceUsage for flag parsing)

### Runtime Errors
- API failures (network, authentication, rate limits)
- File I/O errors
- Database connection failures

**Handling:** Use SilenceUsage, log with context, wrap and return

### Partial Failures
- Some resources succeed, others fail
- Batch operations with mixed results

**Handling:** Log warnings, collect errors, return combined error or continue based on policy

## Testing Error Paths

```go
func TestAnalyzer_ErrorHandling(t *testing.T) {
    tests := []struct {
        name    string
        setup   func() *Analyzer
        wantErr bool
        errMsg  string
    }{
        {
            name: "nil baseline returns error",
            setup: func() *Analyzer {
                return &Analyzer{}
            },
            wantErr: true,
            errMsg:  "baseline is nil",
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            analyzer := tt.setup()
            err := analyzer.Validate()
            
            if tt.wantErr {
                if err == nil {
                    t.Errorf("expected error but got nil")
                }
                if !strings.Contains(err.Error(), tt.errMsg) {
                    t.Errorf("error = %v, want message containing %q", err, tt.errMsg)
                }
            } else {
                if err != nil {
                    t.Errorf("unexpected error: %v", err)
                }
            }
        })
    }
}
```

## References

### Official Documentation
- [Go Error Handling](https://go.dev/blog/error-handling-and-go)
- [Go 1.13 Errors](https://go.dev/blog/go1.13-errors)
- [Cobra Documentation](https://github.com/spf13/cobra)

### Best Practices Articles
- [JetBrains: CLI Apps with Cobra - Error Handling](https://www.jetbrains.com/guide/go/tutorials/cli-apps-go-cobra/error_handling/)
- [Datadog: Go Error Handling](https://www.datadoghq.com/blog/go-error-handling/)
- [Cobra Issue #914: Error Handling Best Practices](https://github.com/spf13/cobra/issues/914)

### Key Takeaways

1. **Use `RunE` + `SilenceUsage`** for clean error messages without confusing usage text
2. **Wrap errors with `%w`** to preserve error chains and enable `errors.Is()` and `errors.As()`
3. **Log errors with context** using structured logging for better debugging
4. **Return errors early** to keep functions readable and avoid deep nesting
5. **Don't call `os.Exit()`** in command handlers - return errors instead

## Migration Checklist

If you're adding a new command or refactoring existing code:

- [ ] Command uses `RunE` instead of `Run`
- [ ] Command has `SilenceUsage: true` set
- [ ] All errors are wrapped with `fmt.Errorf(..., %w, err)`
- [ ] Errors are logged with appropriate context before returning
- [ ] No `os.Exit()` calls in command handlers
- [ ] Custom error types implement `Error()` method if needed
- [ ] Error paths are tested in unit tests
- [ ] Documentation updated if error handling changes
