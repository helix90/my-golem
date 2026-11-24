package golem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SRAIXConfig represents configuration for external SRAIX services
type SRAIXConfig struct {
	// Service name identifier
	Name string `json:"name"`
	// Base URL for the service
	BaseURL string `json:"base_url"`
	// URL template with placeholders like {input}, {apikey}, {list_id}
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
	// Params maps placeholders to the predicates that fill them, e.g.
	// {"access_token": "list_access_token"} fills {access_token} with what
	// <get name="list_access_token"/> returns
	Params map[string]string `json:"params"`
	// Set maps predicates to JSON paths: after a successful JSON response,
	// a <sraix> tag sets each predicate to the value at its path, e.g.
	// {"geocode_lon": "0.lon"}
	Set map[string]string `json:"set"`
	// Defaults gives placeholders a value when nothing else does. Values
	// can use ${ENV_VAR} and ${ENV_VAR:-fallback}.
	Defaults map[string]string `json:"defaults"`
	// Username and Password, if set, are sent as basic auth. They can use
	// ${ENV_VAR}, so that they need not be written in configuration files.
	Username string `json:"username"`
	Password string `json:"password"`
	// Handler names a built-in protocol that makes the service's requests
	// instead of a single HTTP request: "caldav" (see caldav.go)
	Handler string `json:"handler"`
	// Operation is what the handler does, e.g. "today" for caldav
	Operation string `json:"operation"`
	// Reauth is an input the bot answers when the service replies 401
	// Unauthorized, such as one that logs in again and sets a fresh token.
	// A <sraix> tag then reads the service's Params again and, if one
	// changed, retries the request once.
	Reauth string `json:"reauth"`
	// BodyTemplate, when set, is the request body, with placeholders
	// filled as in the URL template and escaped for the Content-Type:
	// URL-encoded for application/x-www-form-urlencoded, JSON string
	// escaped for JSON (so a template writes "{name}" in quotes), as they
	// are otherwise. Use it rather than building a body in AIML, where a
	// value with & or " would break it.
	BodyTemplate string `json:"body"`
}

// sraixStatusError is a request that the service answered with an HTTP
// error status
type sraixStatusError struct {
	status int
	detail string
}

func (e *sraixStatusError) Error() string {
	return fmt.Sprintf("SRAIX request failed with status %d: %s", e.status, e.detail)
}

// isUnauthorized reports whether a SRAIX request failed with 401 Unauthorized
func isUnauthorized(err error) bool {
	var statusErr *sraixStatusError
	return errors.As(err, &statusErr) && statusErr.status == http.StatusUnauthorized
}

// applyDefaults fills in the method, timeout, response format and headers
// when they are not set
func (config *SRAIXConfig) applyDefaults() {
	config.Method = strings.ToUpper(config.Method)
	if config.Method == "" {
		config.Method = http.MethodPost
	}
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.ResponseFormat == "" {
		config.ResponseFormat = "text"
	}
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
}

// target returns the URL template, or the base URL when there is none
func (config *SRAIXConfig) target() string {
	if config.URLTemplate != "" {
		return config.URLTemplate
	}
	return config.BaseURL
}

// clone returns a copy of the config that shares no maps with it
func (config *SRAIXConfig) clone() *SRAIXConfig {
	c := *config
	c.Headers = cloneStringMap(config.Headers)
	c.Params = cloneStringMap(config.Params)
	c.Set = cloneStringMap(config.Set)
	c.Defaults = cloneStringMap(config.Defaults)
	return &c
}

func cloneStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	c := make(map[string]string, len(m))
	for key, value := range m {
		c[key] = value
	}
	return c
}

// maxSRAIXResponseSize limits how much of a service's response is read
const maxSRAIXResponseSize = 1 << 20

// SRAIXManager manages external service configurations and HTTP client
type SRAIXManager struct {
	// mu guards configs and fromProperties, which change when a knowledge
	// base is loaded while sessions may be making SRAIX requests. A config
	// is never changed once added, only replaced.
	mu      sync.RWMutex
	configs map[string]*SRAIXConfig
	// fromProperties names the services configured from properties, so that
	// a reload can remove the ones whose properties are gone
	fromProperties map[string]bool
	client         *http.Client
	logger         *log.Logger
	verbose        bool
	// reminders is the store of the "reminders" handler; see SetReminderStore
	reminders *ReminderStore
	// now returns the current time; tests replace it
	now func() time.Time
}

// NewSRAIXManager creates a new SRAIX manager
func NewSRAIXManager(logger *log.Logger, verbose bool) *SRAIXManager {
	return &SRAIXManager{
		configs:        make(map[string]*SRAIXConfig),
		fromProperties: make(map[string]bool),
		// No client-wide timeout: each request gets its config's timeout,
		// and a client timeout would cap any config timeout above it
		client:  &http.Client{},
		logger:  logger,
		verbose: verbose,
		now:     time.Now,
	}
}

// AddConfig adds a copy of a SRAIX service configuration, with defaults
// filled in
func (sm *SRAIXManager) AddConfig(config *SRAIXConfig) error {
	if err := sm.addConfig(config); err != nil {
		return err
	}
	sm.mu.Lock()
	delete(sm.fromProperties, config.Name)
	sm.mu.Unlock()
	return nil
}

func (sm *SRAIXManager) addConfig(config *SRAIXConfig) error {
	if config.Name == "" {
		return fmt.Errorf("SRAIX config name cannot be empty")
	}
	if config.BaseURL == "" && config.URLTemplate == "" && config.Handler != "reminders" {
		return fmt.Errorf("SRAIX config base URL or URL template is required")
	}
	config = config.clone()
	config.applyDefaults()
	if err := validateHandler(config); err != nil {
		return err
	}
	if len(config.Set) > 0 && config.ResponseFormat != "json" {
		return fmt.Errorf("SRAIX service '%s' sets predicates, which needs the json response format", config.Name)
	}
	if config.ResponsePath != "" && config.ResponseFormat != "json" {
		sm.logger.Printf("Warning: SRAIX service '%s' has a response path, which is used only with the json response format", config.Name)
	}

	sm.mu.Lock()
	sm.configs[config.Name] = config
	sm.mu.Unlock()
	if sm.verbose {
		sm.logger.Printf("Added SRAIX config: %s -> %s", config.Name, config.target())
	}
	return nil
}

// GetConfig retrieves a SRAIX service configuration. The config must not be
// changed; use AddConfig to replace it.
func (sm *SRAIXManager) GetConfig(name string) (*SRAIXConfig, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	config, exists := sm.configs[name]
	return config, exists
}

// ListConfigs returns copies of all configured SRAIX services
func (sm *SRAIXManager) ListConfigs() map[string]*SRAIXConfig {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	configs := make(map[string]*SRAIXConfig, len(sm.configs))
	for name, config := range sm.configs {
		configs[name] = config.clone()
	}
	return configs
}

// ProcessSRAIX makes the request a SRAIX tag asks for. When the request
// fails it returns the service's fallback response, if one is configured.
// params supplies the pattern wildcards and any other values for the
// placeholders of the URL template and headers; values given in the input
// take precedence over them. It does not set the service's predicates,
// which belong to a session.
func (sm *SRAIXManager) ProcessSRAIX(serviceName, input string, params map[string]string) (string, error) {
	config, exists := sm.GetConfig(serviceName)
	if !exists {
		return "", fmt.Errorf("SRAIX service '%s' not configured", serviceName)
	}
	response, err := sm.send(config, input, params, params)
	if err != nil {
		if sm.verbose {
			sm.logger.Printf("SRAIX request to %s failed: %v", serviceName, err)
		}
		if config.FallbackResponse != "" {
			return config.FallbackResponse, nil
		}
		return "", err
	}
	return response.text, nil
}

// validateHandler checks a service's handler and operation
func validateHandler(config *SRAIXConfig) error {
	switch config.Handler {
	case "":
		return nil
	case "caldav":
		if config.URLTemplate == "" {
			return fmt.Errorf("SRAIX service '%s': the caldav handler needs a URL template for the calendar", config.Name)
		}
		if !caldavOperations[config.Operation] {
			return fmt.Errorf("SRAIX service '%s': unknown caldav operation %q", config.Name, config.Operation)
		}
		return nil
	case "reminders":
		if !reminderOperations[config.Operation] {
			return fmt.Errorf("SRAIX service '%s': unknown reminders operation %q", config.Name, config.Operation)
		}
		return nil
	}
	return fmt.Errorf("SRAIX service '%s': unknown handler %q", config.Name, config.Handler)
}

// methodHasBody reports whether requests with the method carry the input
// as a body. The others pass it in the URL.
func methodHasBody(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodDelete, http.MethodOptions:
		return false
	}
	return true
}

// sraixResponse is the result of a SRAIX request: the response text, and
// the values at the paths of the service's Set
type sraixResponse struct {
	text string
	set  map[string]string
}

// send builds the request for a service, sends it and returns the
// response. wildcards are the pattern wildcards, which a wrapped JSON body
// includes when the service asks for them; they and the other params fill
// placeholders.
func (sm *SRAIXManager) send(config *SRAIXConfig, input string, wildcards, given map[string]string) (sraixResponse, error) {
	hasBody := methodHasBody(config.Method)

	// Values given in the input, as a JSON object or as form fields, take
	// precedence over the params passed in, and those over the wildcards
	params := make(map[string]string, len(wildcards)+len(given))
	for key, value := range wildcards {
		params[key] = value
	}
	for key, value := range given {
		params[key] = value
	}
	inputParams := parseInputParams(input)
	for key, value := range inputParams {
		params[key] = value
	}
	for key, value := range config.Defaults {
		if params[key] != "" {
			continue
		}
		if value, _, err := sm.substitutePlaceholders(value, "", nil, nil, nil); err == nil {
			params[key] = value
		}
	}

	if config.Handler == "caldav" {
		return sm.sendCalDAV(config, input, params)
	}
	if config.Handler == "reminders" {
		return sm.sendReminders(config, input, params)
	}

	var requestURL, logURL string
	if config.URLTemplate != "" {
		// When the input is plain text and is not the body, it fills a
		// template's only placeholder, e.g. the {list_id} of /lists/{list_id}
		trimmedInput := strings.TrimSpace(input)
		if !hasBody && inputParams == nil && trimmedInput != "" {
			if placeholders := extractPlaceholders(config.URLTemplate); len(placeholders) == 1 {
				if _, exists := params[placeholders[0]]; !exists {
					params[placeholders[0]] = trimmedInput
				}
			}
		}
		var err error
		requestURL, logURL, err = sm.substituteURLTemplate(config.URLTemplate, input, params, config.Headers)
		if err != nil {
			return sraixResponse{}, err
		}
	} else {
		requestURL = config.BaseURL
		if !hasBody {
			// Pass the input as a query parameter
			paramName := config.QueryParam
			if paramName == "" {
				paramName = "input"
			}
			separator := "?"
			if strings.Contains(requestURL, "?") {
				separator = "&"
			}
			requestURL += separator + url.QueryEscape(paramName) + "=" + url.QueryEscape(input)
		}
		logURL = requestURL
	}

	var body []byte
	var contentType string
	if hasBody {
		var err error
		body, contentType, err = sm.requestBody(config, input, wildcards, params)
		if err != nil {
			return sraixResponse{}, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	defer cancel()
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, config.Method, requestURL, bodyReader)
	if err != nil {
		return sraixResponse{}, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	if err := sm.setHeaders(req, config, input, params); err != nil {
		return sraixResponse{}, err
	}
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}

	if sm.verbose {
		// The URL and headers are logged with environment variables and the
		// API key masked, and Authorization not at all. Bodies, which often
		// carry credentials or tokens, are logged only by size.
		sm.logger.Printf("=== SRAIX Request to %s ===", config.Name)
		sm.logger.Printf("Method: %s", config.Method)
		sm.logger.Printf("URL: %s", logURL)
		sm.logger.Printf("Headers:")
		for key, value := range config.Headers {
			if strings.EqualFold(key, "Authorization") {
				value = "[redacted]"
			} else {
				_, value, _ = sm.substitutePlaceholders(value, "", nil, nil, nil)
			}
			sm.logger.Printf("  %s: %s", key, value)
		}
		if body != nil {
			sm.logger.Printf("Body: %d bytes, %s", len(body), req.Header.Get("Content-Type"))
		}
		sm.logger.Printf("=========================")
	}

	return sm.doRequest(req, config)
}

// setHeaders sets a service's configured headers, whose placeholders are
// substituted (e.g. Bearer {access_token}), and its basic auth
func (sm *SRAIXManager) setHeaders(req *http.Request, config *SRAIXConfig, input string, params map[string]string) error {
	for key, value := range config.Headers {
		substitutedValue, _, err := sm.substitutePlaceholders(value, input, params, config.Headers, nil)
		if err != nil {
			return fmt.Errorf("header %s: %v", key, err)
		}
		req.Header.Set(key, substitutedValue)
	}
	if config.Username != "" {
		username, _, err := sm.substitutePlaceholders(config.Username, "", params, nil, nil)
		if err != nil {
			return fmt.Errorf("username: %v", err)
		}
		password, _, err := sm.substitutePlaceholders(config.Password, "", params, nil, nil)
		if err != nil {
			return fmt.Errorf("password: %v", err)
		}
		req.SetBasicAuth(username, password)
	}
	return nil
}

// formInputRe matches input written as form fields: name=value&name=value
var formInputRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*=[^&]*(&[A-Za-z_][A-Za-z0-9_.-]*=[^&]*)*$`)

// parseInputParams returns the fields of input written as a JSON object or
// as form fields, or nil for any other input
func parseInputParams(input string) map[string]string {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "{") {
		var fields map[string]interface{}
		if decodeJSON([]byte(trimmed), &fields) != nil {
			return nil
		}
		params := make(map[string]string, len(fields))
		for key, value := range fields {
			params[key] = jsonText(value)
		}
		return params
	}
	if formInputRe.MatchString(trimmed) {
		values, err := url.ParseQuery(trimmed)
		if err != nil {
			return nil
		}
		params := make(map[string]string, len(values))
		for key, value := range values {
			params[key] = value[0]
		}
		return params
	}
	return nil
}

// decodeJSON decodes JSON keeping numbers as they are written, so that an
// ID such as 12345678 does not become 1.2345678e+07
func decodeJSON(data []byte, v interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	if decoder.More() {
		return fmt.Errorf("unexpected data after JSON value")
	}
	return nil
}

// jsonText returns a decoded JSON value as text: a string as it is, null as
// "", and anything else as JSON
func jsonText(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case json.Number:
		return v.String()
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

// headerValue returns a configured header, whatever the case of its name
func headerValue(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

// requestBody returns the body of a request and, if the service does not
// configure one, its content type. Form, XML, iCalendar and other non-JSON
// services get the input as it is. JSON services get JSON input as it is,
// and other input wrapped as {"input": ...}, with the wildcards if the
// service asks for them, and the tag's botid, host and hint.
func (sm *SRAIXManager) requestBody(config *SRAIXConfig, input string, wildcards, params map[string]string) ([]byte, string, error) {
	mediaType := ""
	if configured := headerValue(config.Headers, "Content-Type"); configured != "" {
		parsed, _, err := mime.ParseMediaType(configured)
		if err != nil {
			return nil, "", fmt.Errorf("invalid Content-Type %q: %v", configured, err)
		}
		mediaType = parsed
	}
	if config.BodyTemplate != "" {
		return sm.templateBody(config, mediaType, input, params)
	}
	if mediaType != "" && mediaType != "application/json" && !strings.HasSuffix(mediaType, "+json") {
		return []byte(input), "", nil
	}

	trimmed := strings.TrimSpace(input)
	if (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) && json.Valid([]byte(trimmed)) {
		return []byte(trimmed), "application/json", nil
	}

	requestData := map[string]interface{}{
		"input": input,
	}
	if config.IncludeWildcards && len(wildcards) > 0 {
		requestData["wildcards"] = wildcards
	}
	for _, key := range []string{"botid", "host", "hint"} {
		if value := params[key]; value != "" {
			requestData[key] = value
		}
	}
	data, err := json.Marshal(requestData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request data: %v", err)
	}
	return data, "application/json", nil
}

// templateBody fills a service's body template, escaping each value for the
// body's media type (JSON when the service configures none)
func (sm *SRAIXManager) templateBody(config *SRAIXConfig, mediaType, input string, params map[string]string) ([]byte, string, error) {
	contentType := ""
	if mediaType == "" {
		mediaType, contentType = "application/json", "application/json"
	}
	var escape func(value string, pos int) string
	switch {
	case mediaType == "application/x-www-form-urlencoded":
		escape = func(value string, _ int) string { return url.QueryEscape(value) }
	case mediaType == "application/json" || strings.HasSuffix(mediaType, "+json"):
		escape = func(value string, _ int) string {
			var quoted bytes.Buffer
			encoder := json.NewEncoder(&quoted)
			encoder.SetEscapeHTML(false)
			_ = encoder.Encode(value)
			text := strings.TrimSpace(quoted.String())
			return text[1 : len(text)-1]
		}
	}
	body, _, err := sm.substitutePlaceholders(config.BodyTemplate, input, params, config.Headers, escape)
	if err != nil {
		return nil, "", fmt.Errorf("SRAIX service '%s' body: %v", config.Name, err)
	}
	return []byte(body), contentType, nil
}

// maxSRAIXErrorBody limits how much of an error response goes into the
// error, and from there into logs
const maxSRAIXErrorBody = 200

// doRequest sends the request and extracts the response text. Any failure,
// including a response with no text (such as a missing response path), is
// returned as an error, so that the caller falls back. A 204 No Content
// response is a success with no text.
func (sm *SRAIXManager) doRequest(req *http.Request, config *SRAIXConfig) (sraixResponse, error) {
	resp, err := sm.client.Do(req)
	if err != nil {
		return sraixResponse{}, fmt.Errorf("SRAIX request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body, up to the size limit
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxSRAIXResponseSize+1))
	if err != nil {
		return sraixResponse{}, fmt.Errorf("failed to read response body: %v", err)
	}
	if len(responseBody) > maxSRAIXResponseSize {
		return sraixResponse{}, fmt.Errorf("SRAIX response exceeds %d bytes", maxSRAIXResponseSize)
	}

	if sm.verbose {
		sm.logger.Printf("SRAIX response from %s: %s, %d bytes", config.Name, resp.Status, len(responseBody))
	}

	if resp.StatusCode >= 400 {
		detail := string(responseBody)
		if len(detail) > maxSRAIXErrorBody {
			detail = detail[:maxSRAIXErrorBody] + "..."
		}
		return sraixResponse{}, &sraixStatusError{status: resp.StatusCode, detail: detail}
	}
	if resp.StatusCode == http.StatusNoContent && len(config.Set) == 0 {
		return sraixResponse{}, nil
	}

	result := sraixResponse{text: string(responseBody)}
	if config.ResponseFormat == "json" && (config.ResponsePath != "" || len(config.Set) > 0) {
		var jsonData interface{}
		if err := decodeJSON(responseBody, &jsonData); err != nil {
			return sraixResponse{}, fmt.Errorf("failed to parse JSON response: %v", err)
		}
		if config.ResponsePath != "" {
			value, found := extractJSONPath(jsonData, config.ResponsePath)
			if !found {
				return sraixResponse{}, fmt.Errorf("SRAIX response has no value at path '%s'", config.ResponsePath)
			}
			result.text = value
		}
		// Every path must have a value, so that a predicate is never left
		// with a value from an earlier response
		if len(config.Set) > 0 {
			result.set = make(map[string]string, len(config.Set))
			for predicate, path := range config.Set {
				value, found := extractJSONPath(jsonData, path)
				if !found {
					return sraixResponse{}, fmt.Errorf("SRAIX response has no value at path '%s'", path)
				}
				result.set[predicate] = value
			}
		}
	}

	result.text = strings.TrimSpace(result.text)
	if result.text == "" && len(result.set) == 0 {
		return sraixResponse{}, fmt.Errorf("SRAIX response is empty")
	}
	return result, nil
}

// extractPlaceholders finds all {placeholder} patterns in a string
func extractPlaceholders(template string) []string {
	var placeholders []string
	for _, m := range urlPlaceholderRe.FindAllStringSubmatch(template, -1) {
		if m[3] != "" {
			placeholders = append(placeholders, m[3])
		}
	}
	return placeholders
}

// urlPlaceholderRe matches the placeholders of a URL template: ${ENV_VAR},
// ${ENV_VAR:-fallback} and {name}
var urlPlaceholderRe = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)(?::-([^}]*))?\}|\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// substituteURLTemplate replaces placeholders in URL template with actual values.
// Supported placeholders:
//
//	{input} - the SRAIX input text
//	{apikey} - API key from headers (Authorization header)
//	{param_name}, {PARAM_NAME} - any param value, e.g. {list_id} from the
//	                             input, {access_token} from a predicate the
//	                             service names, or {STAR1} for the first
//	                             pattern wildcard
//	${ENV_VAR} - environment variable (e.g., ${PIRATE_WEATHER_API_KEY})
//	${ENV_VAR:-fallback} - environment variable, or fallback when it is not set
//
// All placeholders are replaced in one pass, so placeholder text inside a
// substituted value (such as user input containing "{apikey}") is left alone.
// Values that can come from the user are escaped, as a path segment before
// the template's "?" and as a query value after it; environment variables
// and the API key are configured by the bot author and inserted as they are.
// It returns the URL and a copy with the environment variables and API key
// masked, for logging. A placeholder with no value, or an environment
// variable that is not set, is an error.
func (sm *SRAIXManager) substituteURLTemplate(template, input string, params map[string]string, headers map[string]string) (string, string, error) {
	queryStart := strings.IndexByte(template, '?')
	escape := func(value string, pos int) string {
		if queryStart >= 0 && pos > queryStart {
			return url.QueryEscape(value)
		}
		return url.PathEscape(value)
	}
	return sm.substitutePlaceholders(template, input, params, headers, escape)
}

// substitutePlaceholders replaces the placeholders described at
// substituteURLTemplate, passing values that can come from the user through
// escape (if not nil) with the placeholder's position in the template.
// Header values use it with no escaping. Placeholders with no value are left
// in the logging copy, and reported in the error.
func (sm *SRAIXManager) substitutePlaceholders(template, input string, params map[string]string, headers map[string]string, escape func(value string, pos int) string) (string, string, error) {
	if escape == nil {
		escape = func(value string, _ int) string { return value }
	}

	values := make(map[string]string, len(params)*2)
	for key, value := range params {
		values[strings.ToUpper(key)] = value
	}
	// Params as given take precedence over uppercased ones
	for key, value := range params {
		values[key] = value
	}

	var missing []string
	var redacted strings.Builder
	var result strings.Builder
	last := 0
	for _, m := range urlPlaceholderRe.FindAllStringSubmatchIndex(template, -1) {
		result.WriteString(template[last:m[0]])
		redacted.WriteString(template[last:m[0]])
		last = m[1]

		if m[2] >= 0 {
			varName := template[m[2]:m[3]]
			envValue := os.Getenv(varName)
			if envValue == "" && m[4] >= 0 {
				envValue = template[m[4]:m[5]]
			} else if envValue == "" {
				missing = append(missing, "${"+varName+"}")
			}
			result.WriteString(envValue)
			redacted.WriteString("${" + varName + "}")
			continue
		}

		name := template[m[6]:m[7]]
		switch {
		case name == "input":
			value := escape(input, m[0])
			result.WriteString(value)
			redacted.WriteString(value)
		case name == "apikey" && headers["Authorization"] != "":
			result.WriteString(headers["Authorization"])
			redacted.WriteString("{apikey}")
		default:
			value, exists := values[name]
			if !exists {
				missing = append(missing, template[m[0]:m[1]])
				redacted.WriteString(template[m[0]:m[1]])
				continue
			}
			value = escape(value, m[0])
			result.WriteString(value)
			redacted.WriteString(value)
		}
	}
	result.WriteString(template[last:])
	redacted.WriteString(template[last:])

	if len(missing) > 0 {
		return "", redacted.String(), fmt.Errorf("no value for %s", strings.Join(missing, ", "))
	}
	return result.String(), redacted.String(), nil
}

// extractJSONPath returns the value at a dot-separated path, such as
// "data.message" or "0.lat" (array indices are numbers), as text. Strings
// are returned as they are, and other values as JSON.
func extractJSONPath(data interface{}, path string) (string, bool) {
	current := data
	for _, part := range strings.Split(path, ".") {
		switch v := current.(type) {
		case map[string]interface{}:
			next, ok := v[part]
			if !ok {
				return "", false
			}
			current = next
		case []interface{}:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(v) {
				return "", false
			}
			current = v[index]
		default:
			return "", false
		}
	}
	return jsonText(current), true
}

// LoadSRAIXConfigsFromFile loads SRAIX configurations from a JSON file
func (sm *SRAIXManager) LoadSRAIXConfigsFromFile(filename string) error {
	data, err := os.ReadFile(filename)
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
//
//	sraix.servicename.baseurl = https://api.example.com
//	sraix.servicename.apikey = your-api-key
//	sraix.servicename.method = POST
//	sraix.servicename.timeout = 30
//	sraix.servicename.responseformat = json
//	sraix.servicename.responsepath = data.response
//	sraix.servicename.fallback = Service unavailable
//	sraix.servicename.header.Authorization = Bearer TOKEN
//	sraix.servicename.header.Content-Type = application/json
//	sraix.servicename.param.access_token = list_access_token
//	sraix.servicename.set.geocode_lon = 0.lon
//
// Services configured from earlier properties that these no longer
// configure are removed.
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
	configured := make(map[string]bool, len(serviceProps))
	for serviceName, props := range serviceProps {
		config, err := sm.buildConfigFromProperties(serviceName, props)
		if err != nil {
			sm.logger.Printf("Warning: Failed to build SRAIX config for service '%s': %v", serviceName, err)
			continue
		}

		if err := sm.addConfig(config); err != nil {
			sm.logger.Printf("Warning: Failed to add SRAIX config for service '%s': %v", serviceName, err)
			continue
		}
		configured[serviceName] = true
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()
	for serviceName := range sm.fromProperties {
		if !configured[serviceName] {
			delete(sm.configs, serviceName)
			if sm.verbose {
				sm.logger.Printf("Removed SRAIX service no longer in properties: %s", serviceName)
			}
		}
	}
	sm.fromProperties = configured

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
		case strings.HasPrefix(key, "param.") && len(key) > len("param."):
			if config.Params == nil {
				config.Params = make(map[string]string)
			}
			config.Params[strings.TrimPrefix(key, "param.")] = value
		case key == "username":
			config.Username = value
		case key == "password":
			config.Password = value
		case key == "handler":
			config.Handler = value
		case key == "reauth":
			config.Reauth = value
		case key == "body":
			config.BodyTemplate = value
		case key == "operation":
			config.Operation = value
		case strings.HasPrefix(key, "default.") && len(key) > len("default."):
			if config.Defaults == nil {
				config.Defaults = make(map[string]string)
			}
			config.Defaults[strings.TrimPrefix(key, "default.")] = value
		case strings.HasPrefix(key, "set.") && len(key) > len("set."):
			if config.Set == nil {
				config.Set = make(map[string]string)
			}
			config.Set[strings.TrimPrefix(key, "set.")] = value
		default:
			sm.logger.Printf("Warning: Unknown SRAIX property for service '%s': %s", serviceName, key)
		}
	}

	// Validate required fields
	if config.BaseURL == "" && config.URLTemplate == "" && config.Handler != "reminders" {
		return nil, fmt.Errorf("baseurl or urltemplate is required for SRAIX service '%s'", serviceName)
	}

	config.applyDefaults()

	return config, nil
}

// ListServices returns the names of all configured SRAIX services, sorted
func (sm *SRAIXManager) ListServices() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	services := make([]string, 0, len(sm.configs))
	for name := range sm.configs {
		services = append(services, name)
	}
	sort.Strings(services)
	return services
}
