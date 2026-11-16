package golem

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

// ASTNodeType represents the type of an AST node
type ASTNodeType int

const (
	NodeTypeText ASTNodeType = iota
	NodeTypeTag
	NodeTypeSelfClosingTag
	NodeTypeComment
	NodeTypeCDATA
)

// ASTNode represents a node in the Abstract Syntax Tree
type ASTNode struct {
	Type        ASTNodeType
	TagName     string            // For tag nodes
	Content     string            // For text nodes
	Children    []*ASTNode        // For tag nodes with content
	Attributes  map[string]string // For tag nodes with attributes
	SelfClosing bool              // For self-closing tags
	StartPos    int               // Start position in original string
	EndPos      int               // End position in original string
}

// ASTParser handles parsing of AIML templates into AST
//
// DO NOT replace this with encoding/xml (xml.Unmarshal, xml.Decoder, or its
// tokenizer). That approach has been tried and fails badly on AIML-specific
// structures. Fix bugs within this hand-written parser instead.
type ASTParser struct {
	input string
	pos   int
	len   int
}

// NewASTParser creates a new AST parser
func NewASTParser(input string) *ASTParser {
	return &ASTParser{
		input: input,
		pos:   0,
		len:   len(input),
	}
}

// Parse parses the input string into an AST
func (p *ASTParser) Parse() (*ASTNode, error) {
	root := &ASTNode{
		Type:     NodeTypeText, // Root node type
		Children: []*ASTNode{},
	}

	p.parseNodes(root, "")

	return root, nil
}

// implicitlySelfClosingTags are treated as self-closing when written without
// a slash and never closed (e.g. <get name="x">)
var implicitlySelfClosingTags = map[string]bool{
	"star":     true,
	"sr":       true,
	"get":      true,
	"bot":      true,
	"that":     true,
	"input":    true,
	"loop":     true,
	"date":     true,
	"time":     true,
	"size":     true,
	"version":  true,
	"id":       true,
	"request":  true,
	"response": true,
	"repeat":   true,
	"topic":    true,
	"subj":     true,
	"pred":     true,
	"obj":      true,
	"uniq":     true,
}

// parseNodes parses nodes into parent until the end of input or, when
// closeTag is set, until that tag's closing tag, which it consumes. It returns
// whether the closing tag was found. Other closing tags are dropped.
func (p *ASTParser) parseNodes(parent *ASTNode, closeTag string) bool {
	for p.pos < p.len {
		if p.peek(2) == "</" {
			p.pos += 2
			name := p.parseTagName()
			if closeTag != "" && name == closeTag {
				p.consume('>')
				return true
			}
			// Not this element's closing tag: drop it
			if name != "" {
				p.consume('>')
			}
			continue
		}

		oldPos := p.pos
		var node *ASTNode
		switch {
		case p.peek(4) == "<!--":
			node = p.parseComment()
		case p.peek(9) == "<![CDATA[":
			node = p.parseCDATA()
		case p.peek(1) == "<":
			node = p.parseTag()
		default:
			node = p.parseText()
		}
		if node != nil {
			parent.Children = append(parent.Children, node)
		}

		// If we didn't make progress, break to avoid infinite loop
		if p.pos == oldPos {
			break
		}
	}
	return false
}

// parseTag parses a tag (opening or self-closing) and its content
func (p *ASTParser) parseTag() *ASTNode {
	startPos := p.pos

	// Consume opening '<'
	if !p.consume('<') {
		return nil
	}

	// Parse tag name
	tagName := p.parseTagName()
	if tagName == "" {
		return nil
	}

	// Parse attributes
	attributes := p.parseAttributes()

	// Check for self-closing tag
	if p.peek(1) == "/" {
		p.consume('/')
		if !p.consume('>') {
			return nil
		}

		return &ASTNode{
			Type:        NodeTypeSelfClosingTag,
			TagName:     tagName,
			Attributes:  attributes,
			SelfClosing: true,
			StartPos:    startPos,
			EndPos:      p.pos,
		}
	}

	// Consume '>'
	if !p.consume('>') {
		return nil
	}

	// Create opening tag node and parse its content up to the matching closing tag
	tagNode := &ASTNode{
		Type:       NodeTypeTag,
		TagName:    tagName,
		Attributes: attributes,
		Children:   []*ASTNode{},
		StartPos:   startPos,
	}
	if p.parseNodes(tagNode, tagName) {
		tagNode.EndPos = p.pos
		return tagNode
	}

	// Unclosed tag: implicitly self-closing tags (e.g. <get name="x">) are
	// treated as self-closing
	if implicitlySelfClosingTags[tagName] {
		return &ASTNode{
			Type:        NodeTypeSelfClosingTag,
			TagName:     tagName,
			Attributes:  attributes,
			SelfClosing: true,
			StartPos:    startPos,
			EndPos:      p.pos,
		}
	}

	// Otherwise it's malformed; return it verbatim as text so the problem
	// is visible in the output
	return &ASTNode{
		Type:     NodeTypeText,
		Content:  p.input[startPos:p.pos],
		StartPos: startPos,
		EndPos:   p.pos,
	}
}

// parseTagName parses a tag name
func (p *ASTParser) parseTagName() string {
	start := p.pos
	for p.pos < p.len && (p.isAlphaNumeric() || p.peek(1) == "-" || p.peek(1) == "_") {
		p.pos++
	}
	return p.input[start:p.pos]
}

// parseAttributes parses tag attributes
func (p *ASTParser) parseAttributes() map[string]string {
	attributes := make(map[string]string)

	for p.pos < p.len && p.peek(1) != ">" && p.peek(2) != "/>" {
		// Skip whitespace
		p.consumeWhitespace()

		// Parse attribute name
		attrName := p.parseAttributeName()
		if attrName == "" {
			break
		}

		// Skip whitespace
		p.consumeWhitespace()

		// Check for '=' or just attribute name (boolean attribute)
		if p.peek(1) == "=" {
			p.consume('=')
			p.consumeWhitespace()

			// Parse attribute value
			attrValue := p.parseAttributeValue()
			attributes[attrName] = attrValue
		} else {
			// Boolean attribute
			attributes[attrName] = ""
		}
	}

	return attributes
}

// parseAttributeName parses an attribute name
func (p *ASTParser) parseAttributeName() string {
	start := p.pos
	for p.pos < p.len && (p.isAlphaNumeric() || p.peek(1) == "-" || p.peek(1) == "_") {
		p.pos++
	}
	return p.input[start:p.pos]
}

// parseAttributeValue parses an attribute value
func (p *ASTParser) parseAttributeValue() string {
	// Handle backslash-escaped quotes (common in Go string literals)
	// Values are sliced before consuming the closing quote so that an
	// unterminated value at end of input can't produce out-of-range slices.
	if p.peek(2) == "\\\"" {
		p.consumeString("\\\"")
		start := p.pos
		for p.pos < p.len && p.peek(2) != "\\\"" {
			p.pos++
		}
		value := html.UnescapeString(p.input[start:p.pos])
		p.consumeString("\\\"")
		return value
	}

	if p.peek(1) == "\"" || p.peek(1) == "'" {
		quote := p.input[p.pos]
		p.pos++
		start := p.pos
		for p.pos < p.len && p.input[p.pos] != quote {
			p.pos++
		}
		value := html.UnescapeString(p.input[start:p.pos])
		p.consume(rune(quote))
		return value
	} else {
		// Unquoted value
		start := p.pos
		for p.pos < p.len && !p.isWhitespace() && p.peek(1) != ">" {
			p.pos++
		}
		return p.input[start:p.pos]
	}
}

// parseText parses text content
func (p *ASTParser) parseText() *ASTNode {
	start := p.pos
	for p.pos < p.len && p.peek(1) != "<" {
		p.pos++
	}

	content := p.input[start:p.pos]
	if content == "" {
		return nil
	}

	return &ASTNode{
		Type:     NodeTypeText,
		Content:  content,
		StartPos: start,
		EndPos:   p.pos,
	}
}

// parseComment parses XML comments
func (p *ASTParser) parseComment() *ASTNode {
	start := p.pos
	if !p.consumeString("<!--") {
		return nil
	}

	// Find end of comment
	for p.pos < p.len-2 {
		if p.peek(3) == "-->" {
			p.consumeString("-->")
			return &ASTNode{
				Type:     NodeTypeComment,
				Content:  p.input[start+4 : p.pos-3],
				StartPos: start,
				EndPos:   p.pos,
			}
		}
		p.pos++
	}

	return nil
}

// parseCDATA parses CDATA sections
func (p *ASTParser) parseCDATA() *ASTNode {
	start := p.pos
	if !p.consumeString("<![CDATA[") {
		return nil
	}

	// Find end of CDATA
	for p.pos < p.len-2 {
		if p.peek(3) == "]]>" {
			p.consumeString("]]>")
			return &ASTNode{
				Type:     NodeTypeCDATA,
				Content:  p.input[start+9 : p.pos-3],
				StartPos: start,
				EndPos:   p.pos,
			}
		}
		p.pos++
	}

	return nil
}

// Helper methods

func (p *ASTParser) isAlphaNumeric() bool {
	c := p.peek(1)
	if len(c) == 0 {
		return false
	}
	char := rune(c[0])
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
}

func (p *ASTParser) isWhitespace() bool {
	c := p.peek(1)
	if len(c) == 0 {
		return false
	}
	char := rune(c[0])
	return char == ' ' || char == '\t' || char == '\n' || char == '\r'
}

func (p *ASTParser) isAtEnd() bool {
	return p.pos >= p.len
}

func (p *ASTParser) peek(n int) string {
	if p.pos+n > p.len {
		return p.input[p.pos:]
	}
	return p.input[p.pos : p.pos+n]
}

func (p *ASTParser) consume(c rune) bool {
	if p.pos < p.len && rune(p.input[p.pos]) == c {
		p.pos++
		return true
	}
	return false
}

func (p *ASTParser) consumeString(s string) bool {
	if p.pos+len(s) <= p.len && p.input[p.pos:p.pos+len(s)] == s {
		p.pos += len(s)
		return true
	}
	return false
}

func (p *ASTParser) consumeWhitespace() {
	for p.pos < p.len && p.isWhitespace() {
		p.pos++
	}
}

// String returns a string representation of the AST node
func (n *ASTNode) String() string {
	switch n.Type {
	case NodeTypeText:
		// If this is a text node with children, return children
		if len(n.Children) > 0 {
			children := ""
			for _, child := range n.Children {
				children += child.String()
			}
			return children
		}
		return n.Content
	case NodeTypeComment:
		return fmt.Sprintf("<!--%s-->", n.Content)
	case NodeTypeCDATA:
		return fmt.Sprintf("<![CDATA[%s]]>", n.Content)
	case NodeTypeSelfClosingTag:
		attrStr := formatAttributes(n.Attributes)
		return fmt.Sprintf("<%s%s/>", n.TagName, attrStr)
	case NodeTypeTag:
		attrStr := formatAttributes(n.Attributes)

		children := ""
		for _, child := range n.Children {
			children += child.String()
		}

		return fmt.Sprintf("<%s%s>%s</%s>", n.TagName, attrStr, children, n.TagName)
	default:
		// For root node or unknown type, return children
		children := ""
		for _, child := range n.Children {
			children += child.String()
		}
		return children
	}
}

// attributeEscaper escapes attribute values so they survive re-parsing
var attributeEscaper = strings.NewReplacer(`&`, "&amp;", `"`, "&quot;", `<`, "&lt;")

// formatAttributes serializes attributes as ` name="value"` pairs in sorted
// order, escaping values. Attributes with empty values are written bare.
func formatAttributes(attributes map[string]string) string {
	if len(attributes) == 0 {
		return ""
	}
	names := make([]string, 0, len(attributes))
	for name := range attributes {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		b.WriteString(" ")
		b.WriteString(name)
		if value := attributes[name]; value != "" {
			b.WriteString(`="`)
			b.WriteString(attributeEscaper.Replace(value))
			b.WriteString(`"`)
		}
	}
	return b.String()
}

// GetTextContent returns all text content from a node and its children
func (n *ASTNode) GetTextContent() string {
	switch n.Type {
	case NodeTypeText:
		// If this is a text node with children, process children
		if len(n.Children) > 0 {
			var content strings.Builder
			for _, child := range n.Children {
				content.WriteString(child.GetTextContent())
			}
			return content.String()
		}
		return n.Content
	case NodeTypeComment, NodeTypeCDATA:
		return ""
	case NodeTypeSelfClosingTag, NodeTypeTag:
		var content strings.Builder
		for _, child := range n.Children {
			content.WriteString(child.GetTextContent())
		}
		return content.String()
	default:
		// Handle root node or unknown type by processing children
		if len(n.Children) > 0 {
			var content strings.Builder
			for _, child := range n.Children {
				content.WriteString(child.GetTextContent())
			}
			return content.String()
		}
		return ""
	}
}

// FindTagsByName finds all tags with the specified name
func (n *ASTNode) FindTagsByName(tagName string) []*ASTNode {
	var result []*ASTNode

	if n.Type == NodeTypeTag && n.TagName == tagName {
		result = append(result, n)
	}

	for _, child := range n.Children {
		result = append(result, child.FindTagsByName(tagName)...)
	}

	return result
}

// FindFirstTagByName finds the first tag with the specified name
func (n *ASTNode) FindFirstTagByName(tagName string) *ASTNode {
	if n.Type == NodeTypeTag && n.TagName == tagName {
		return n
	}

	for _, child := range n.Children {
		if found := child.FindFirstTagByName(tagName); found != nil {
			return found
		}
	}

	return nil
}
