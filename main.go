package main

import (
"context"
"flag"
"fmt"
"log"
"os"

"github.com/yourusername/drift-analysis-cli/pkg/csql"
"github.com/yourusername/drift-analysis-cli/pkg/gke"
"gopkg.in/yaml.v3"
)

// UnifiedConfig represents the unified YAML configuration for both SQL and GKE
type UnifiedConfig struct {
Projects      []string          `yaml:"projects"`
SQLBaselines  []csql.SQLBaseline  `yaml:"sql_baselines,omitempty"`
GKEBaselines  []gke.GKEBaseline   `yaml:"gke_baselines,omitempty"`

// Legacy support
Baselines    []csql.SQLBaseline  `yaml:"baselines,omitempty"`
}

func main() {
if len(os.Args) < 2 {
printUsage()
os.Exit(1)
}

command := os.Args[1]

switch command {
case "sql":
runSQLCommand(os.Args[2:])
case "gke":
runGKECommand(os.Args[2:])
case "help", "-h", "--help":
printUsage()
default:
fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
printUsage()
os.Exit(1)
}
}

func runSQLCommand(args []string) {
fs := flag.NewFlagSet("sql", flag.ExitOnError)

projects := fs.String("projects", "", "Comma-separated list of GCP project IDs")
config := fs.String("config", "", "Path to unified YAML config file")
output := fs.String("output", "", "Output file path (default: stdout)")
format := fs.String("format", "text", "Output format: text, json, yaml")
filterRole := fs.String("filter-role", "", "Filter by database-role label")
generateConfig := fs.Bool("generate-config", false, "Generate baseline config from current state")

fs.Parse(args)

ctx := context.Background()

// Load unified config if provided
var projectList []string
var baselines []csql.SQLBaseline

if *config != "" {
unifiedConfig, err := loadUnifiedConfig(*config)
if err != nil {
log.Fatalf("Failed to load config: %v", err)
}
projectList = unifiedConfig.Projects

// Use sql_baselines if present, otherwise fall back to baselines (legacy)
if len(unifiedConfig.SQLBaselines) > 0 {
baselines = unifiedConfig.SQLBaselines
} else if len(unifiedConfig.Baselines) > 0 {
baselines = unifiedConfig.Baselines
}
}

cmd := &csql.Command{
Projects:       *projects,
ProjectList:    projectList,
Baselines:      baselines,
OutputFile:     *output,
Format:         *format,
FilterRole:     *filterRole,
GenerateConfig: *generateConfig,
}

if err := cmd.Execute(ctx); err != nil {
log.Fatalf("Error: %v", err)
}
}

func runGKECommand(args []string) {
fs := flag.NewFlagSet("gke", flag.ExitOnError)

projects := fs.String("projects", "", "Comma-separated list of GCP project IDs")
config := fs.String("config", "", "Path to unified YAML config file")
output := fs.String("output", "", "Output file path (default: stdout)")
format := fs.String("format", "text", "Output format: text, json, yaml")
filterRole := fs.String("filter-role", "", "Filter by cluster-role label")
generateConfig := fs.Bool("generate-config", false, "Generate baseline config from current state")

fs.Parse(args)

ctx := context.Background()

// Load unified config if provided
var projectList []string
var baselines []gke.GKEBaseline

if *config != "" {
unifiedConfig, err := loadUnifiedConfig(*config)
if err != nil {
log.Fatalf("Failed to load config: %v", err)
}
projectList = unifiedConfig.Projects
baselines = unifiedConfig.GKEBaselines
}

cmd := &gke.Command{
Projects:       *projects,
ProjectList:    projectList,
Baselines:      baselines,
OutputFile:     *output,
Format:         *format,
FilterRole:     *filterRole,
GenerateConfig: *generateConfig,
}

if err := cmd.Execute(ctx); err != nil {
log.Fatalf("Error: %v", err)
}
}

func loadUnifiedConfig(path string) (*UnifiedConfig, error) {
data, err := os.ReadFile(path)
if err != nil {
return nil, err
}

var config UnifiedConfig
if err := yaml.Unmarshal(data, &config); err != nil {
return nil, err
}

return &config, nil
}

func printUsage() {
fmt.Println(`GCP Drift Analysis CLI

Usage:
  drift-analysis-cli <command> [flags]

Commands:
  sql     Analyze Cloud SQL instances for configuration drift
  gke     Analyze GKE clusters for configuration drift
  help    Show this help message

SQL Command Flags:
  -projects string
        Comma-separated list of GCP project IDs
  -config string
        Path to unified YAML config file (containing sql_baselines)
  -output string
        Output file path (default: stdout)
  -format string
        Output format: text, json, yaml (default: text)
  -filter-role string
        Filter instances by database-role label
  -generate-config
        Generate baseline config from current state

GKE Command Flags:
  -projects string
        Comma-separated list of GCP project IDs
  -config string
        Path to unified YAML config file (containing gke_baselines)
  -output string
        Output file path (default: stdout)
  -format string
        Output format: text, json, yaml (default: text)
  -filter-role string
        Filter clusters by cluster-role label
  -generate-config
        Generate baseline config from current state

Examples:
  # Analyze Cloud SQL with unified config
  drift-analysis-cli sql -config config.yaml

  # Analyze GKE with unified config
  drift-analysis-cli gke -config config.yaml

  # Analyze specific SQL role
  drift-analysis-cli sql -projects "project1,project2" -filter-role vault

  # Generate SQL baseline
  drift-analysis-cli sql -projects "project1" -generate-config -output sql-baseline.yaml

  # Generate GKE baseline
  drift-analysis-cli gke -projects "project1" -generate-config -output gke-baseline.yaml

  # Export analysis as JSON
  drift-analysis-cli sql -config config.yaml -format json -output report.json

Unified Config Format (config.yaml):
  projects:
    - project-1
    - project-2
  
  sql_baselines:
    - name: "application"
      filter_labels:
        database-role: "application"
      config:
        # SQL configuration
  
  gke_baselines:
    - name: "production"
      filter_labels:
        cluster-role: "production"
      cluster_config:
        # GKE cluster configuration
      nodepool_config:
        # Node pool configuration
`)
}
