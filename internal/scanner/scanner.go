// Package scanner detects sensitive information in files before sending to LLM APIs.
package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding represents a detected sensitive item.
type Finding struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Type     string `json:"type"` // "api_key", "password", "token", "private_key", "env_file"
	Content  string `json:"content"` // masked preview
	Severity string `json:"severity"` // "high", "medium", "low"
}

// Patterns for sensitive data detection.
var patterns = []struct {
	Name     string
	Severity string
	Regex    *regexp.Regexp
}{
	// API keys and tokens
	{"API Key / Token", "high", regexp.MustCompile(`(?i)(api[_-]?key|apikey|api_secret|api[_-]?token)\s*[:=]\s*['"]?[a-z0-9_\-\.]{16,}['"]?`)},
	{"Bearer Token", "high", regexp.MustCompile(`(?i)(bearer|auth[_-]?token|access[_-]?token)\s+[a-z0-9_\-\.]{16,}`)},
	{"Authorization Header", "high", regexp.MustCompile(`(?i)authorization:\s*(basic|bearer|digest)\s+`)},
	
	// Passwords
	{"Password", "high", regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*['"]?[^'"\s]{4,}['"]?`)},
	
	// Private keys and certificates
	{"Private Key", "critical", regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`)},
	{"Certificate", "medium", regexp.MustCompile(`-----BEGIN\s+CERTIFICATE-----`)},
	
	// Database and service URLs with credentials
	{"Connection String", "high", regexp.MustCompile(`[a-z]+://[^:]+:[^@]+@`)}, // protocol://user:pass@host
	
	// AWS-style keys
	{"AWS Key", "high", regexp.MustCompile(`(?i)(AKIA[0-9A-Z]{16}|aws[_-]?(access[_-]?)?key[_-]?id)`)},
	
	// Generic secret
	{"Secret", "high", regexp.MustCompile(`(?i)(secret|SECRET)\s*[:=]\s*['"]?[a-z0-9_\-\.]{16,}['"]?`)},
	
	// Session tokens / JWTs
	{"JWT / Session Token", "high", regexp.MustCompile(`eyJ[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}`)},
}

// sensitiveFilenames lists files that are likely to contain secrets.
var sensitiveFilenames = []string{
	".env", ".env.*", "*.pem", "*.key", "*.pkcs8", "secret*", "credentials*",
	".netrc", "*.cred", "secrets.yml", "secrets.yaml", "vault.*",
}

// allowedFiles lists files that are safe to send (no scanning needed).
var allowedExtensions = []string{".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs", ".toml", ".mod", ".sum", ".md", ".json", ".yaml", ".yml", ".css", ".html"}

// ScanDir scans a directory for sensitive files before sending to LLM.
// It returns findings grouped by file.
func ScanDir(dir string, ignoreDirs []string) ([]Finding, error) {
	var findings []Finding
	ignoreSet := make(map[string]bool)
	for _, d := range ignoreDirs {
		ignoreSet[d] = true
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == ".next" || ignoreSet[name] {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if filename matches sensitive patterns
		base := info.Name()
		for _, pat := range sensitiveFilenames {
			matched, _ := filepath.Match(pat, base)
			if matched {
				findings = append(findings, Finding{
					File:     path,
					Line:     0,
					Type:     "Sensitive File",
					Content:  fmt.Sprintf("[%s matches pattern: %s]", base, pat),
					Severity: "high",
				})
				return nil
			}
		}

		// Only scan text files with known extensions
		ext := filepath.Ext(base)
		isAllowed := false
		for _, e := range allowedExtensions {
			if ext == e {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			return nil
		}

		// Read file and scan content line by line
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			for _, pat := range patterns {
				if pat.Regex.MatchString(line) {
					findings = append(findings, Finding{
						File:     path,
						Line:     i + 1,
						Type:     pat.Name,
						Content:  maskLine(line),
						Severity: pat.Severity,
					})
				}
			}
		}
		return nil
	})

	return findings, err
}

// maskLine replaces sensitive values with *** for display.
func maskLine(line string) string {
	// Replace anything that looks like a value after = or :
	re := regexp.MustCompile(`(['"]?)([a-zA-Z0-9_\-\.]{8,})\1`)
	return re.ReplaceAllString(line, "${1}****${1}")
}

// PrintFindings prints findings to stderr and returns true if any critical/high severity found.
func PrintFindings(findings []Finding) bool {
	hasCritical := false
	for _, f := range findings {
		if f.Severity == "critical" || f.Severity == "high" {
			hasCritical = true
		}
	}

	if len(findings) == 0 {
		return false
	}

	fmt.Fprintf(os.Stderr, "\n⚠️  SENSITIVE DATA DETECTED\n")
	fmt.Fprintf(os.Stderr,   "==========================\n\n")

	for _, f := range findings {
		icon := "🟢"
		switch f.Severity {
		case "critical":
			icon = "🔴 CRITICAL"
		case "high":
			icon = "🟠 HIGH"
		case "medium":
			icon = "🟡 MEDIUM"
		default:
			icon = "🔵 LOW"
		}
		fmt.Fprintf(os.Stderr, "%s | %s:%d\n", icon, f.File, f.Line)
		fmt.Fprintf(os.Stderr, "     Type: %s\n", f.Type)
		fmt.Fprintf(os.Stderr, "     Near: %s\n\n", f.Content)
	}

	return hasCritical
}

// ConfirmScan prompts the user to continue despite findings.
// Returns true if user wants to abort (safety first).
func ConfirmScan(findings []Finding) bool {
	hasCritical := false
	for _, f := range findings {
		if f.Severity == "critical" || f.Severity == "high" {
			hasCritical = true
		}
	}

	if !hasCritical {
		return false // no critical issues, continue
	}

	fmt.Fprintf(os.Stderr, "⚠️  High-severity or critical secrets detected.\n")
	fmt.Fprintf(os.Stderr, "   Send this data to the cloud LLM API? [y/N]: ")
	
	var response string
	fmt.Scanln(&response)
	response = strings.TrimSpace(strings.ToLower(response))
	
	return response != "y" && response != "yes"
}
