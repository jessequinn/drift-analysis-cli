# TUI (Terminal User Interface) Feature

The drift-analysis-cli now supports an interactive Terminal User Interface (TUI) mode for viewing drift analysis reports in a more organized and user-friendly way.

## Features

### Tabbed Interface
The TUI provides 6 organized tabs:

1. **Overview** - Summary statistics, compliance rate, and drift counts by severity
2. **Critical** - Only resources with critical severity drifts
3. **High** - Only resources with high severity drifts
4. **Medium** - Only resources with medium severity drifts
5. **Low** - Only resources with low severity drifts
6. **All Drifts** - Complete list of all resources and their drifts

### Navigation
- **Tab / → / l** - Switch to next tab
- **Shift+Tab / ← / h** - Switch to previous tab
- **↑ / k** - Scroll up
- **↓ / j** - Scroll down
- **PgUp / b** - Page up
- **PgDn / f** - Page down
- **u / Ctrl+u** - Half page up
- **d / Ctrl+d** - Half page down
- **q / Esc / Ctrl+c** - Quit

### Visual Design
- Color-coded severity levels (red for critical, orange for high, yellow for medium, gray for low)
- Progress indicator showing scroll position
- Styled headers and resource information
- Clean, organized layout with proper spacing

## Usage

### Cloud SQL Analysis
```bash
# Run with TUI output
drift-analysis-cli gcp sql --config config.yaml --output tui

# Or use short flag
drift-analysis-cli gcp sql -c config.yaml -o tui
```

### GKE Cluster Analysis
```bash
# Run with TUI output
drift-analysis-cli gcp gke --config config.yaml --output tui

# Or use short flag
drift-analysis-cli gcp gke -c config.yaml -o tui
```

## Benefits

1. **Better Organization** - Filter drifts by severity level instantly
2. **Interactive Exploration** - Navigate through results at your own pace
3. **Quick Overview** - See summary statistics at a glance
4. **Focused Analysis** - Jump directly to critical issues
5. **No External Tools** - Everything runs in your terminal

## Comparison with Other Formats

| Format | Use Case |
|--------|----------|
| `tui` | Interactive exploration and analysis |
| `text` | Quick terminal output or piping to files |
| `json` | Programmatic processing and CI/CD integration |
| `yaml` | Configuration-friendly format |

## Example Workflow

1. Run analysis with TUI mode:
   ```bash
   drift-analysis-cli gcp sql -c config.yaml -o tui
   ```

2. Start in Overview tab to see:
   - Total resources analyzed
   - Compliance rate percentage
   - Drift counts by severity

3. Press `tab` to switch to Critical tab
   - Review all critical issues first
   - Note which resources need immediate attention

4. Use arrow keys or vim-style navigation (j/k) to scroll through results

5. Press `q` to exit when done

## Technical Details

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components (viewport)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Terminal styling

The TUI implementation is located in `pkg/tui/` and includes:
- `model.go` - Core TUI model and event handling
- `report.go` - Tab building and content formatting
- `converters.go` - Converting SQL/GKE reports to TUI format
