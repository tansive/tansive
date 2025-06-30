package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/joho/godotenv"
)

type TemplateContext struct {
	ENV map[string]string
}

// PreprocessYAML replaces {{ .ENV.VAR }} placeholders with values from env or .env file.
func PreprocessYAML(inputRaw []byte) ([]byte, error) {
	input := string(inputRaw)
	// Load .env file from the current working directory if it exists
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	envPath := filepath.Join(cwd, ".env")
	_ = godotenv.Load(envPath) // no error if .env doesn't exist

	envMap := map[string]string{}
	for _, e := range os.Environ() {
		parts := bytes.SplitN([]byte(e), []byte("="), 2)
		if len(parts) == 2 {
			envMap[string(parts[0])] = string(parts[1])
		}
	}

	ctx := TemplateContext{ENV: envMap}

	// Parse and execute the template
	tmpl, err := template.New("yaml").Option("missingkey=error").Parse(input)
	if err != nil {
		return nil, err
	}

	var output bytes.Buffer

	missingKeyRegex := regexp.MustCompile(`map has no entry for key "(.*?)"`)

	if err := tmpl.Execute(&output, ctx); err != nil {
		matches := missingKeyRegex.FindStringSubmatch(err.Error())
		if len(matches) == 2 {
			missingKey := matches[1]
			return nil, fmt.Errorf("missing environment variable: %s (set it in your shell or .env file)", missingKey)
		}
		return nil, fmt.Errorf("template error: %w", err)
	}

	return output.Bytes(), nil
}
