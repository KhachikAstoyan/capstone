package driver

import "fmt"

// Generate returns a (header, footer) pair for the given language.
// The combined source sent to the executor is:
//
//	header + "\n\n" + userCode + "\n\n" + footer
//
// For problems without a FunctionSpec the caller skips this and uses
// the user's source as-is (raw stdin/stdout mode).
func Generate(spec FunctionSpec, lang string) (header, footer string, err error) {
	switch lang {
	case "python", "python3":
		return generatePython(spec)
	case "javascript":
		return generateJavaScript(spec)
	case "go":
		return generateGo(spec)
	case "java":
		return generateJava(spec)
	default:
		return "", "", fmt.Errorf("unsupported language for function-call driver: %q", lang)
	}
}

// Wrap returns the fully-assembled source ready to compile/run.
// Handles language-specific quirks (e.g. Go imports must be at the top).
func Wrap(spec FunctionSpec, lang, userSource string) (string, error) {
	header, footer, err := Generate(spec, lang)
	if err != nil {
		return "", err
	}
	switch lang {
	case "go":
		userImports, userBody := extractGoImports(userSource)
		merged := injectGoImports(header, userImports)
		return merged + "\n\n" + userBody + "\n\n" + footer, nil
	default:
		return header + "\n\n" + userSource + "\n\n" + footer, nil
	}
}
