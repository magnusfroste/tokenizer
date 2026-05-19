// Package classifier extracts deterministic, safe prompt signals for routing.
package classifier

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/magnusfroste/tokenizer/internal/openai"
)

const (
	MaxKeywords         = 32
	MaxFilesTouched     = 32
	MaxSensitivityHints = 16

	largeContextCharThreshold = 32000
)

var (
	codeBlockPattern  = regexp.MustCompile("(?s)```[^`]*```")
	inlineCodePattern = regexp.MustCompile("`([^`\n]+)`")
	pathPattern       = regexp.MustCompile(`(?i)(?:^|[\s"'(:\[])(` +
		`(?:[A-Za-z0-9_.@-]+/)+[A-Za-z0-9_.@-]+\.(?:go|ts|tsx|js|jsx|py|sql|json|ya?ml|toml|md|rs|java|kt|swift|rb|php|cs|cpp|cc|cxx|c|h|hpp|css|scss|html|svelte|vue|sh)|` +
		`(?:package|tsconfig|go\.mod|go\.sum|Dockerfile|Makefile|README)\.(?:json|lock|mod|sum|md)?|` +
		`(?:package\.json|go\.mod|go\.sum|Dockerfile|Makefile|README\.md)` +
		`)`)
	stackLinePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bpanic:`),
		regexp.MustCompile(`(?i)\bfatal error:`),
		regexp.MustCompile(`(?i)\btraceback \(most recent call last\):`),
		regexp.MustCompile(`\bFile "[^"]+", line \d+`),
		regexp.MustCompile(`\bat\s+.+\([^)]*:\d+(?::\d+)?\)`),
		regexp.MustCompile(`\bat\s+[^\s]+:\d+(?::\d+)?`),
		regexp.MustCompile(`\b[A-Za-z0-9_./-]+\.go:\d+\s+\+0x[0-9a-fA-F]+`),
	}
)

type Features struct {
	HasCodeBlock         bool
	CodeBlockCount       int
	HasInlineCode        bool
	InlineCodeCount      int
	HasStackTrace        bool
	StackTraceCount      int
	FilesTouched         []string
	Keywords             []string
	SensitivityHints     []string
	RequiresCode         bool
	RequiresToolUse      bool
	RequiresJSONSchema   bool
	RequiresVision       bool
	RequiresLargeContext bool
}

type RequestHints struct {
	Tools          []any
	HasTools       bool
	ResponseFormat any
	Metadata       map[string]any
	Model          string
	MaxTokens      *int
}

type keywordRule struct {
	keyword string
	terms   []string
}

var keywordRules = []keywordRule{
	{keyword: "sql", terms: []string{"select from", "insert into", "update set", "delete from", "alter table", "create table", "create index", "drop table", ".sql"}},
	{keyword: "migration", terms: []string{"migration", "migrations", "schema", "rollback", "db/migrations"}},
	{keyword: "auth", terms: []string{"auth", "authentication", "authorization", "oauth", "session", "jwt", "token", "password", "permission", "rbac"}},
	{keyword: "payment", terms: []string{"payment", "payments", "billing", "checkout", "stripe", "invoice", "subscription", "refund"}},
	{keyword: "security", terms: []string{"security", "vulnerability", "xss", "csrf", "ssrf", "injection", "secret", "exploit", "cve"}},
	{keyword: "exploit", terms: []string{"exploit", "exploited", "exploitation", "rce", "remote code execution"}},
	{keyword: "production", terms: []string{"production", "prod", "incident", "down", "outage"}},
	{keyword: "urgent", terms: []string{"urgent", "asap", "hotfix", "rollback"}},
	{keyword: "change_intent", terms: []string{"fix", "change", "edit", "modify", "patch", "update", "implement", "rollback"}},
	{keyword: "json_schema", terms: []string{"json schema", "json_schema", "structured output", "return json", "respond with json", "valid json", "matching schema"}},
	{keyword: "tool_use", terms: []string{"tool call", "function calling", "call the tool", "use tools"}},
	{keyword: "vision", terms: []string{"image", "screenshot", "photo", "diagram", "vision"}},
	{keyword: "large_context", terms: []string{"large context", "long context", "entire codebase", "whole repository", "full repo"}},
	{keyword: "source_code", terms: []string{"source code", "codebase", "repository", "repo"}},
	{keyword: "pii", terms: []string{"ssn", "personnummer", "email", "phone", "address", "customer data"}},
	{keyword: "secret", terms: []string{"api key", "api_key", "secret", "private key", "password", "token"}},
}

func ExtractFromRequest(req openai.ChatRequest) Features {
	return ExtractFromMessages(req.Messages, RequestHints{
		Tools:          req.Tools,
		HasTools:       len(req.Tools) > 0,
		ResponseFormat: req.ResponseFormat,
		Metadata:       req.Metadata,
		Model:          req.Model,
		MaxTokens:      req.MaxTokens,
	})
}

func ExtractFromMessages(messages []openai.Message, hints RequestHints) Features {
	var f Features
	seenKeywords := make(map[string]struct{})
	seenFiles := make(map[string]struct{})
	seenSensitivity := make(map[string]struct{})

	addKeyword := func(keyword string) {
		addCapped(&f.Keywords, seenKeywords, strings.ToLower(keyword), MaxKeywords)
	}
	addSensitivity := func(hint string) {
		addCapped(&f.SensitivityHints, seenSensitivity, strings.ToLower(hint), MaxSensitivityHints)
	}

	if hints.HasTools || len(hints.Tools) > 0 || metadataTruthy(hints.Metadata, "tools", "tool_choice", "function_call", "functions") {
		f.RequiresToolUse = true
		addKeyword("tool_use")
	}
	if responseFormatRequiresJSONSchema(hints.ResponseFormat) || metadataTruthy(hints.Metadata, "json_schema", "schema") {
		f.RequiresJSONSchema = true
		addKeyword("json_schema")
	}
	if metadataTruthy(hints.Metadata, "vision", "image", "modalities") {
		f.RequiresVision = true
		addKeyword("vision")
	}
	if metadataTruthy(hints.Metadata, "large_context", "long_context", "context_tokens") {
		f.RequiresLargeContext = true
		addKeyword("large_context")
	}

	totalChars := 0
	for _, msg := range messages {
		text := msg.Content
		totalChars += len(text)

		codeBlocks := codeBlockPattern.FindAllStringIndex(text, -1)
		f.CodeBlockCount += len(codeBlocks)

		inlineText := codeBlockPattern.ReplaceAllString(text, " ")
		f.InlineCodeCount += len(inlineCodePattern.FindAllStringSubmatch(inlineText, -1))

		stackCount := countStackTraceLines(text)
		f.StackTraceCount += stackCount

		for _, match := range pathPattern.FindAllStringSubmatch(text, -1) {
			if len(match) < 2 {
				continue
			}
			path := cleanPath(match[1])
			if path == "" {
				continue
			}
			addCapped(&f.FilesTouched, seenFiles, strings.ToLower(path), MaxFilesTouched, path)
			if strings.HasSuffix(strings.ToLower(path), ".sql") || strings.Contains(strings.ToLower(path), "/migrations/") {
				addKeyword("sql")
				addKeyword("migration")
			}
		}

		lower := strings.ToLower(text)
		for _, rule := range keywordRules {
			if containsAnyTerm(lower, rule.terms) {
				addKeyword(rule.keyword)
			}
		}
	}

	f.HasCodeBlock = f.CodeBlockCount > 0
	f.HasInlineCode = f.InlineCodeCount > 0
	f.HasStackTrace = f.StackTraceCount > 0
	f.RequiresCode = f.HasCodeBlock || f.HasInlineCode || f.HasStackTrace || len(f.FilesTouched) > 0 || hasAnyKeyword(f.Keywords, "sql", "migration", "source_code")

	if hasAnyKeyword(f.Keywords, "json_schema") {
		f.RequiresJSONSchema = true
	}
	if hasAnyKeyword(f.Keywords, "tool_use") {
		f.RequiresToolUse = true
	}
	if hasAnyKeyword(f.Keywords, "vision") {
		f.RequiresVision = true
	}
	if hasAnyKeyword(f.Keywords, "large_context") || totalChars >= largeContextCharThreshold {
		f.RequiresLargeContext = true
		addKeyword("large_context")
	}

	if f.RequiresCode {
		addSensitivity("source_code")
	}
	if hasAnyKeyword(f.Keywords, "secret") {
		addSensitivity("secrets_possible")
	}
	if hasAnyKeyword(f.Keywords, "pii") {
		addSensitivity("pii")
	}
	if hasAnyKeyword(f.Keywords, "auth") {
		addSensitivity("auth")
	}
	if hasAnyKeyword(f.Keywords, "payment") {
		addSensitivity("payment")
	}
	if hasAnyKeyword(f.Keywords, "security") {
		addSensitivity("security")
	}

	return f
}

func countStackTraceLines(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		for _, pattern := range stackLinePatterns {
			if pattern.MatchString(line) {
				count++
				break
			}
		}
	}
	return count
}

func containsAnyTerm(lower string, terms []string) bool {
	for _, term := range terms {
		if containsTerm(lower, term) {
			return true
		}
	}
	return false
}

func containsTerm(lower, term string) bool {
	if term == "" {
		return false
	}
	if hasNonWordSearchChar(term) {
		return strings.Contains(lower, term)
	}
	start := 0
	for {
		idx := strings.Index(lower[start:], term)
		if idx == -1 {
			return false
		}
		idx += start
		before := idx == 0 || !isSearchWordByte(lower[idx-1])
		afterIdx := idx + len(term)
		after := afterIdx == len(lower) || !isSearchWordByte(lower[afterIdx])
		if before && after {
			return true
		}
		start = idx + len(term)
	}
}

func hasNonWordSearchChar(term string) bool {
	for i := 0; i < len(term); i++ {
		if !isSearchWordByte(term[i]) {
			return true
		}
	}
	return false
}

func isSearchWordByte(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_'
}

func hasAnyKeyword(keywords []string, wanted ...string) bool {
	for _, keyword := range keywords {
		for _, want := range wanted {
			if keyword == want {
				return true
			}
		}
	}
	return false
}

func metadataTruthy(metadata map[string]any, keys ...string) bool {
	if len(metadata) == 0 {
		return false
	}
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		if isTruthy(value) {
			return true
		}
	}
	return false
}

func isTruthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		trimmed := strings.ToLower(strings.TrimSpace(typed))
		return trimmed != "" && trimmed != "false"
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return fmt.Sprint(value) != ""
	}
}

func responseFormatRequiresJSONSchema(format any) bool {
	typed, ok := format.(map[string]any)
	if !ok {
		return false
	}
	formatType, _ := typed["type"].(string)
	if strings.EqualFold(strings.TrimSpace(formatType), "json_schema") {
		return true
	}
	_, hasSchema := typed["json_schema"]
	return hasSchema
}

func cleanPath(path string) string {
	path = strings.Trim(path, " \t\r\n\"'`()[]{}<>.,;:")
	if path == "." || path == ".." || strings.Contains(path, "://") {
		return ""
	}
	return path
}

func addCapped(values *[]string, seen map[string]struct{}, key string, cap int, value ...string) {
	if key == "" {
		return
	}
	if _, ok := seen[key]; ok {
		return
	}
	if len(*values) >= cap {
		return
	}
	seen[key] = struct{}{}
	out := key
	if len(value) > 0 {
		out = value[0]
	}
	*values = append(*values, out)
}
