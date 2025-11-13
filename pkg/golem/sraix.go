package golem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SRAIXConfig represents configuration for external SRAIX services
type SRAIXConfig struct {
	// Service name identifier
	Name string `json:"name"`
	// Base URL for the service
	BaseURL string `json:"base_url"`
	// URL template with placeholders like {input}, {apikey}, {lat}, {lon}
	// If set, this overrides BaseURL and standard query param handling
	URLTemplate string `json:"url_template"`
	// Query parameter name for GET requests (default: "input")
	QueryParam string `json:"query_param"`
	// HTTP method (GET, POST, etc.)
	Method string `json:"method"`
	// Headers to include in requests
	Headers map[string]string `json:"headers"`
	// Request timeout in seconds
	Timeout int `json:"timeout"`
	// Response format (json, xml, text)
	ResponseFormat string `json:"response_format"`
	// JSON path to extract response (for JSON responses)
	ResponsePath string `json:"response_path"`
	// Fallback response when service is unavailable
	FallbackResponse string `json:"fallback_response"`
	// Whether to include wildcards in the request
	IncludeWildcards bool `json:"include_wildcards"`
}

// SRAIXManager manages external service configurations and HTTP client
type SRAIXManager struct {
	configs map[string]*SRAIXConfig
	client  *http.Client
	logger  *log.Logger
	verbose bool
}

// NewSRAIXManager creates a new SRAIX manager
func NewSRAIXManager(logger *log.Logger, verbose bool) *SRAIXManager {
	return &SRAIXManager{
		configs: make(map[string]*SRAIXConfig),
		client: &http.Client{
			Timeout: 30 * time.Second, // Default timeout
		},
		logger:  logger,
		verbose: verbose,
	}
}

// AddConfig adds a new SRAIX service configuration
func (sm *SRAIXManager) AddConfig(config *SRAIXConfig) error {
	if config.Name == "" {
		return fmt.Errorf("SRAIX config name cannot be empty")
	}
	if config.BaseURL == "" && config.URLTemplate == "" {
		return fmt.Errorf("SRAIX config base URL or URL template is required")
	}
	if config.Method == "" {
		config.Method = "POST" // Default to POST
	}
	if config.Timeout == 0 {
		config.Timeout = 30 // Default 30 seconds
	}
	if config.ResponseFormat == "" {
		config.ResponseFormat = "text" // Default to text
	}
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}

	sm.configs[config.Name] = config
	if sm.verbose {
		url := config.BaseURL
		if url == "" {
			url = config.URLTemplate
		}
		sm.logger.Printf("Added SRAIX config: %s -> %s", config.Name, url)
	}
	return nil
}

// GetConfig retrieves a SRAIX service configuration
func (sm *SRAIXManager) GetConfig(name string) (*SRAIXConfig, bool) {
	config, exists := sm.configs[name]
	return config, exists
}

// ListConfigs returns all configured SRAIX services
func (sm *SRAIXManager) ListConfigs() map[string]*SRAIXConfig {
	return sm.configs
}

// ProcessSRAIX processes a SRAIX tag by making an external HTTP request
func (sm *SRAIXManager) ProcessSRAIX(serviceName, input string, wildcards map[string]string) (string, error) {
	config, exists := sm.GetConfig(serviceName)
	if !exists {
		return "", fmt.Errorf("SRAIX service '%s' not configured", serviceName)
	}

	// Parse form-urlencoded input to extract parameters for URL substitution
	// This handles cases like "list_id=1&content=buy milk" where list_id is needed in the URL
	if strings.Contains(input, "=") && strings.Contains(input, "&") {
		pairs := strings.Split(input, "&")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				// Only add if not already present (passed wildcards take precedence)
				if _, exists := wildcards[key]; !exists {
					wildcards[key] = value
				}
			}
		}
	}

	// Prepare the request
	var url string
	var body io.Reader
	var contentType string

	// Check if URL template is configured
	if config.URLTemplate != "" {
		// Use URL template with placeholder substitution
		url = sm.substituteURLTemplate(config.URLTemplate, input, wildcards, config.Headers)
	} else {
		url = config.BaseURL
	}

	// Build request body based on method and configuration
	if config.Method == "GET" {
		// Skip query param appending if URL template was used
		if config.URLTemplate == "" {
			// For GET requests, append input as query parameter
			paramName := config.QueryParam
			if paramName == "" {
				paramName = "input" // Default parameter name
			}
			if strings.Contains(url, "?") {
				url += "&" + paramName + "=" + strings.ReplaceAll(input, " ", "+")
			} else {
				url += "?" + paramName + "=" + strings.ReplaceAll(input, " ", "+")
			}
		}
	} else {
		// Check if Content-Type is already configured
		configuredContentType := config.Headers["Content-Type"]

		// For form-urlencoded requests, use input directly as body
		if configuredContentType == "application/x-www-form-urlencoded" {
			// Input is already in form-urlencoded format (e.g., "username=X&password=Y")
			body = bytes.NewBufferString(input)
			contentType = "application/x-www-form-urlencoded"
		} else if configuredContentType == "application/json" {
			// For JSON Content-Type, check if input is already valid JSON
			trimmedInput := strings.TrimSpace(input)
			if (strings.HasPrefix(trimmedInput, "{") && strings.HasSuffix(trimmedInput, "}")) ||
				(strings.HasPrefix(trimmedInput, "[") && strings.HasSuffix(trimmedInput, "]")) {
				// Input appears to be JSON already, use it directly
				// Validate it's parseable JSON
				var testJSON interface{}
				if json.Unmarshal([]byte(trimmedInput), &testJSON) == nil {
					// Valid JSON, use as-is
					body = bytes.NewBufferString(trimmedInput)
					contentType = "application/json"
				} else {
					// Not valid JSON, wrap it
					requestData := map[string]interface{}{
						"input": input,
					}
					jsonData, err := json.Marshal(requestData)
					if err != nil {
						return "", fmt.Errorf("failed to marshal request data: %v", err)
					}
					body = bytes.NewBuffer(jsonData)
					contentType = "application/json"
				}
			} else {
				// Not JSON format, wrap in {"input": ...}
				requestData := map[string]interface{}{
					"input": input,
				}

				// Include wildcards if configured
				if config.IncludeWildcards && len(wildcards) > 0 {
					requestData["wildcards"] = wildcards
				}

				// Include additional SRAIX parameters
				if botid, exists := wildcards["botid"]; exists && botid != "" {
					requestData["botid"] = botid
				}
				if host, exists := wildcards["host"]; exists && host != "" {
					requestData["host"] = host
				}
				if hint, exists := wildcards["hint"]; exists && hint != "" {
					requestData["hint"] = hint
				}

				jsonData, err := json.Marshal(requestData)
				if err != nil {
					return "", fmt.Errorf("failed to marshal request data: %v", err)
				}
				body = bytes.NewBuffer(jsonData)
				contentType = "application/json"
			}
		} else {
			// For other POST/PUT requests (default), create JSON body
			requestData := map[string]interface{}{
				"input": input,
			}

			// Include wildcards if configured
			if config.IncludeWildcards && len(wildcards) > 0 {
				requestData["wildcards"] = wildcards
			}

			// Include additional SRAIX parameters
			if botid, exists := wildcards["botid"]; exists && botid != "" {
				requestData["botid"] = botid
			}
			if host, exists := wildcards["host"]; exists && host != "" {
				requestData["host"] = host
			}
			if hint, exists := wildcards["hint"]; exists && hint != "" {
				requestData["hint"] = hint
			}

			jsonData, err := json.Marshal(requestData)
			if err != nil {
				return "", fmt.Errorf("failed to marshal request data: %v", err)
			}
			body = bytes.NewBuffer(jsonData)
			contentType = "application/json"
		}
	}

	// Create HTTP request
	req, err := http.NewRequest(config.Method, url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers - configured headers take precedence
	// Substitute placeholders in header values (e.g., {access_token}, {user_id})
	for key, value := range config.Headers {
		// Substitute placeholders in header value
		substitutedValue := sm.substituteURLTemplate(value, input, wildcards, config.Headers)
		req.Header.Set(key, substitutedValue)
	}
	// Only set Content-Type if not already configured
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	// Make the request
	if sm.verbose {
		sm.logger.Printf("=== SRAIX Request to %s ===", serviceName)
		sm.logger.Printf("Method: %s", config.Method)
		sm.logger.Printf("URL: %s", url)
		sm.logger.Printf("Headers:")
		for key, values := range req.Header {
			for _, value := range values {
				sm.logger.Printf("  %s: %s", key, value)
			}
		}
		if body != nil {
			// Read body for logging
			bodyBytes, _ := io.ReadAll(body)
			sm.logger.Printf("Body: %s", string(bodyBytes))
			// Recreate body since we consumed it
			body = bytes.NewBuffer(bodyBytes)
			req.Body = io.NopCloser(body)
		}
		sm.logger.Printf("=========================")
	}

	resp, err := sm.client.Do(req)
	if err != nil {
		if sm.verbose {
			sm.logger.Printf("SRAIX request failed: %v", err)
		}
		// Return fallback response if configured
		if config.FallbackResponse != "" {
			return config.FallbackResponse, nil
		}
		return "", fmt.Errorf("SRAIX request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Log response details
	if sm.verbose {
		sm.logger.Printf("=== SRAIX Response from %s ===", serviceName)
		sm.logger.Printf("Status: %d %s", resp.StatusCode, resp.Status)
		sm.logger.Printf("Response Headers:")
		for key, values := range resp.Header {
			for _, value := range values {
				sm.logger.Printf("  %s: %s", key, value)
			}
		}
		sm.logger.Printf("Response Body: %s", string(responseBody))
		sm.logger.Printf("============================")
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		// Return fallback response if configured
		if config.FallbackResponse != "" {
			return config.FallbackResponse, nil
		}
		return "", fmt.Errorf("SRAIX request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Process response based on format
	response := string(responseBody)
	switch config.ResponseFormat {
	case "json":
		if config.ResponsePath != "" {
			// Extract specific field from JSON response
			var jsonData interface{}
			if err := json.Unmarshal(responseBody, &jsonData); err != nil {
				return "", fmt.Errorf("failed to parse JSON response: %v", err)
			}
			// JSON path extraction (supports dot notation and array indices like "0.lat")
			response = sm.extractJSONPath(jsonData, config.ResponsePath)
		}
	case "xml":
		// For XML, we'll return the raw response for now
		// Could be enhanced to parse XML and extract specific elements
	case "text":
		// Return raw text response
	default:
		// Default to text
	}

	if sm.verbose {
		sm.logger.Printf("SRAIX response from %s: %s", serviceName, response)
	}

	return strings.TrimSpace(response), nil
}

// substituteURLTemplate replaces placeholders in URL template with actual values
// Supported placeholders:
//   {input} - the SRAIX input text
//   {apikey} - API key from headers (Authorization header)
//   {lat}, {lon} - latitude/longitude from wildcards or parsed from hint
//   {location} - location name from wildcards
//   {WILDCARD_NAME} - any wildcard value in uppercase
//   ${ENV_VAR} - environment variable (e.g., ${PIRATE_WEATHER_API_KEY})
func (sm *SRAIXManager) substituteURLTemplate(template, input string, wildcards map[string]string, headers map[string]string) string {
	result := template

	// First, substitute environment variables ${ENV_VAR}
	// Match ${VARNAME} pattern and replace with os.Getenv(VARNAME)
	envVarPattern := regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)
	result = envVarPattern.ReplaceAllStringFunc(result, func(match string) string {
		// Extract variable name from ${VARNAME}
		varName := match[2 : len(match)-1] // Remove ${ and }
		envValue := os.Getenv(varName)
		if envValue == "" && sm.verbose {
			sm.logger.Printf("Warning: Environment variable %s is not set", varName)
		}
		return envValue
	})

	// URL-encode the input for safe inclusion in URLs
	encodedInput := strings.ReplaceAll(input, " ", "+")

	// Substitute {input}
	result = strings.ReplaceAll(result, "{input}", encodedInput)

	// Substitute {apikey} from Authorization header
	if apikey, exists := headers["Authorization"]; exists {
		result = strings.ReplaceAll(result, "{apikey}", apikey)
	}

	// Parse coordinates from hint if present (format: "lat,lon")
	if hint, exists := wildcards["hint"]; exists && hint != "" {
		parts := strings.Split(hint, ",")
		if len(parts) == 2 {
			// Add parsed lat/lon to wildcards for substitution
			wildcards["lat"] = strings.TrimSpace(parts[0])
			wildcards["lon"] = strings.TrimSpace(parts[1])
		}
	}

	// Substitute common wildcard placeholders
	commonWildcards := []string{"lat", "lon", "location", "hint", "botid", "host", "user_id", "list_id", "item_id", "access_token"}
	for _, key := range commonWildcards {
		placeholder := "{" + key + "}"
		if value, exists := wildcards[key]; exists {
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}

	// Substitute any uppercase wildcard placeholders {WILDCARD_NAME}
	for key, value := range wildcards {
		placeholder := "{" + strings.ToUpper(key) + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Also substitute lowercase placeholders for any remaining wildcards
	// This handles cases like {username}, {password}, etc.
	for key, value := range wildcards {
		placeholder := "{" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// extractJSONPath extracts a value from JSON data using dot notation
func (sm *SRAIXManager) extractJSONPath(data interface{}, path string) string {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, return the value
			switch v := current.(type) {
			case map[string]interface{}:
				if val, ok := v[part]; ok {
					if str, ok := val.(string); ok {
						return str
					}
					return fmt.Sprintf("%v", val)
				}
			case []interface{}:
				// If current is an array and part is a number, index into it
				var index int
				if n, err := fmt.Sscanf(part, "%d", &index); err == nil && n == 1 {
					if index >= 0 && index < len(v) {
						if str, ok := v[index].(string); ok {
							return str
						}
						return fmt.Sprintf("%v", v[index])
					}
				}
			}
			return ""
		}

		// Navigate deeper
		switch v := current.(type) {
		case map[string]interface{}:
			if next, ok := v[part]; ok {
				current = next
			} else {
				return ""
			}
		case []interface{}:
			// If current is an array and part is a number, index into it
			var index int
			if n, err := fmt.Sscanf(part, "%d", &index); err == nil && n == 1 {
				if index >= 0 && index < len(v) {
					current = v[index]
				} else {
					return ""
				}
			} else {
				return ""
			}
		default:
			return ""
		}
	}

	return ""
}

// LoadSRAIXConfigsFromFile loads SRAIX configurations from a JSON file
func (sm *SRAIXManager) LoadSRAIXConfigsFromFile(filename string) error {
	data, err := readFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read SRAIX config file: %v", err)
	}

	var configs []*SRAIXConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to parse SRAIX config file: %v", err)
	}

	for _, config := range configs {
		if err := sm.AddConfig(config); err != nil {
			return fmt.Errorf("failed to add SRAIX config %s: %v", config.Name, err)
		}
	}

	return nil
}

// LoadSRAIXConfigsFromDirectory loads all SRAIX configuration files from a directory
func (sm *SRAIXManager) LoadSRAIXConfigsFromDirectory(dirPath string) error {
	files, err := listFiles(dirPath, ".sraix.json")
	if err != nil {
		return fmt.Errorf("failed to list SRAIX config files: %v", err)
	}

	for _, file := range files {
		if err := sm.LoadSRAIXConfigsFromFile(file); err != nil {
			sm.logger.Printf("Warning: Failed to load SRAIX config file %s: %v", file, err)
		}
	}

	return nil
}

// readFile reads the contents of a file
func readFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// listFiles lists files in a directory with a specific extension
func listFiles(dirPath, extension string) ([]string, error) {
	var files []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), strings.ToLower(extension)) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

// ConfigureFromProperties configures SRAIX services from AIML properties
// Properties should be named like:
//   sraix.servicename.baseurl = https://api.example.com
//   sraix.servicename.apikey = your-api-key
//   sraix.servicename.method = POST
//   sraix.servicename.timeout = 30
//   sraix.servicename.responseformat = json
//   sraix.servicename.responsepath = data.response
//   sraix.servicename.fallback = Service unavailable
//   sraix.servicename.header.Authorization = Bearer TOKEN
//   sraix.servicename.header.Content-Type = application/json
func (sm *SRAIXManager) ConfigureFromProperties(properties map[string]string) error {
	// Group properties by service name
	serviceProps := make(map[string]map[string]string)

	for key, value := range properties {
		// Only process properties starting with "sraix."
		if !strings.HasPrefix(key, "sraix.") {
			continue
		}

		// Parse the property key: sraix.servicename.property
		parts := strings.SplitN(key, ".", 3)
		if len(parts) < 3 {
			sm.logger.Printf("Warning: Invalid SRAIX property format: %s (expected sraix.servicename.property)", key)
			continue
		}

		serviceName := parts[1]
		propertyName := parts[2]

		if serviceName == "" {
			sm.logger.Printf("Warning: Empty service name in SRAIX property: %s", key)
			continue
		}

		// Initialize service properties map if needed
		if serviceProps[serviceName] == nil {
			serviceProps[serviceName] = make(map[string]string)
		}

		serviceProps[serviceName][propertyName] = value
	}

	// Create SRAIX configs from grouped properties
	for serviceName, props := range serviceProps {
		config, err := sm.buildConfigFromProperties(serviceName, props)
		if err != nil {
			sm.logger.Printf("Warning: Failed to build SRAIX config for service '%s': %v", serviceName, err)
			continue
		}

		if err := sm.AddConfig(config); err != nil {
			sm.logger.Printf("Warning: Failed to add SRAIX config for service '%s': %v", serviceName, err)
			continue
		}

		if sm.verbose {
			sm.logger.Printf("Configured SRAIX service from properties: %s -> %s", serviceName, config.BaseURL)
		}
	}

	return nil
}

// buildConfigFromProperties builds a SRAIXConfig from property map
func (sm *SRAIXManager) buildConfigFromProperties(serviceName string, props map[string]string) (*SRAIXConfig, error) {
	config := &SRAIXConfig{
		Name:    serviceName,
		Headers: make(map[string]string),
	}

	// Parse standard properties
	for key, value := range props {
		switch {
		case key == "baseurl":
			config.BaseURL = value
		case key == "urltemplate":
			config.URLTemplate = value
		case key == "queryparam":
			config.QueryParam = value
		case key == "apikey":
			// API key can be set as a header or used differently based on the service
			// By default, add it as Authorization header
			config.Headers["Authorization"] = value
		case key == "method":
			config.Method = strings.ToUpper(value)
		case key == "timeout":
			timeout, err := strconv.Atoi(value)
			if err != nil {
				sm.logger.Printf("Warning: Invalid timeout value for service '%s': %s", serviceName, value)
			} else {
				config.Timeout = timeout
			}
		case key == "responseformat":
			config.ResponseFormat = value
		case key == "responsepath":
			config.ResponsePath = value
		case key == "fallback":
			config.FallbackResponse = value
		case key == "includewildcards":
			include, err := strconv.ParseBool(value)
			if err != nil {
				sm.logger.Printf("Warning: Invalid includewildcards value for service '%s': %s", serviceName, value)
			} else {
				config.IncludeWildcards = include
			}
		case strings.HasPrefix(key, "header."):
			// Extract header name
			headerName := strings.TrimPrefix(key, "header.")
			if headerName != "" {
				config.Headers[headerName] = value
			}
		default:
			sm.logger.Printf("Warning: Unknown SRAIX property for service '%s': %s", serviceName, key)
		}
	}

	// Validate required fields
	if config.BaseURL == "" && config.URLTemplate == "" {
		return nil, fmt.Errorf("baseurl or urltemplate is required for SRAIX service '%s'", serviceName)
	}

	// Set defaults if not specified
	if config.Method == "" {
		config.Method = "POST"
	}
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.ResponseFormat == "" {
		config.ResponseFormat = "text"
	}

	return config, nil
}
