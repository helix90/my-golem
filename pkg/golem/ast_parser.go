package golem

import (
	"fmt"
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

	p.parseChildren(root)

	// If root has children, change its type to indicate it's a container
	if len(root.Children) > 0 {
		root.Type = NodeTypeText // Keep as text but handle in String method
	}

	return root, nil
}

// parseChildren parses all children of a node
func (p *ASTParser) parseChildren(parent *ASTNode) {
	for p.pos < p.len {
		oldPos := p.pos

		// Parse whitespace as text content
		if p.isWhitespace() {
			node := p.parseText()
			if node != nil {
				parent.Children = append(parent.Children, node)
			}
			continue
		}

		// Check for comment
		if p.peek(4) == "<!--" {
			node := p.parseComment()
			if node != nil {
				parent.Children = append(parent.Children, node)
			}
			continue
		}

		// Check for CDATA
		if p.peek(9) == "<![CDATA[" {
			node := p.parseCDATA()
			if node != nil {
				parent.Children = append(parent.Children, node)
			}
			continue
		}

		// Check for opening tag
		if p.peek(1) == "<" && !p.isAtEnd() {
			node := p.parseTag()
			if node != nil {
				parent.Children = append(parent.Children, node)
			}
			// Continue parsing even if parseTag() returned nil (for closing tags)
			// This ensures we parse any text that follows the closing tag
			continue
		}

		// Parse text content
		node := p.parseText()
		if node != nil {
			parent.Children = append(parent.Children, node)
		}

		// If we didn't make progress, break to avoid infinite loop
		if p.pos == oldPos {
			break
		}
	}
}

// parseTag parses a tag (opening, closing, or self-closing)
func (p *ASTParser) parseTag() *ASTNode {
	startPos := p.pos

	// Consume opening '<'
	if !p.consume('<') {
		return nil
	}

	// Check for closing tag
	if p.peek(1) == "/" {
		return p.parseClosingTag(startPos)
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

	// Check if this is an implicitly self-closing tag
	implicitlySelfClosing := map[string]bool{
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

	// If this is an implicitly self-closing tag and we're at the end or next non-whitespace is '<'
	if implicitlySelfClosing[tagName] {
		// Save current position
		savedPos := p.pos

		// Skip whitespace
		for p.pos < p.len && p.isWhitespace() {
			p.pos++
		}

		// If we're at the end or the next character is '<', treat as self-closing
		if p.pos >= p.len || p.peek(1) == "<" {
			// Consume the '>' character
			if p.consume('>') {
				return &ASTNode{
					Type:        NodeTypeSelfClosingTag,
					TagName:     tagName,
					Attributes:  attributes,
					SelfClosing: true,
					StartPos:    startPos,
					EndPos:      p.pos,
				}
			}
		}

		// Reset position if not self-closing
		p.pos = savedPos
	}

	// Consume '>'
	if !p.consume('>') {
		return nil
	}

	// Create opening tag node
	tagNode := &ASTNode{
		Type:       NodeTypeTag,
		TagName:    tagName,
		Attributes: attributes,
		Children:   []*ASTNode{},
		StartPos:   startPos,
	}

	// Parse children (content between opening and closing tags)
	// We need to parse until we find the matching closing tag
	for p.pos < p.len {
		// Check if we're at a closing tag
		if p.peek(2) == "</" {
			// Check if it's the matching closing tag
			closingStart := p.pos
			p.consume('<')
			p.consume('/')
			closingTagName := p.parseTagName()
			if closingTagName == tagName {
				// Found matching closing tag
				p.consume('>')
				tagNode.EndPos = p.pos
				// Successfully parsed tag
				return tagNode
			} else {
				// Not the right closing tag, reset position and continue parsing
				p.pos = closingStart
			}
		}

		// Parse the next node
		oldPos := p.pos

		// Parse whitespace as text content (don't skip it)
		if p.isWhitespace() {
			node := p.parseText()
			if node != nil {
				tagNode.Children = append(tagNode.Children, node)
			}
			continue
		}

		// Check for comment
		if p.peek(4) == "<!--" {
			node := p.parseComment()
			if node != nil {
				tagNode.Children = append(tagNode.Children, node)
			}
			continue
		}

		// Check for CDATA
		if p.peek(9) == "<![CDATA[" {
			node := p.parseCDATA()
			if node != nil {
				tagNode.Children = append(tagNode.Children, node)
			}
			continue
		}

		// Check for opening tag
		if p.peek(1) == "<" && !p.isAtEnd() {
			node := p.parseTag()
			if node != nil {
				tagNode.Children = append(tagNode.Children, node)
			}
			continue
		}

		// Parse text content
		node := p.parseText()
		if node != nil {
			tagNode.Children = append(tagNode.Children, node)
		}

		// If we didn't make progress, break to avoid infinite loop
		if p.pos == oldPos {
			break
		}
	}

	// If we get here, check if it should be a self-closing tag
	implicitlySelfClosingTags := map[string]bool{
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

	if implicitlySelfClosingTags[tagName] {
		// This is an implicitly self-closing tag
		return &ASTNode{
			Type:        NodeTypeSelfClosingTag,
			TagName:     tagName,
			Attributes:  attributes,
			SelfClosing: true,
			StartPos:    startPos,
			EndPos:      p.pos,
		}
	}

	// If we get here, it's a malformed tag
	// Return as text node
	return &ASTNode{
		Type:     NodeTypeText,
		Content:  p.input[startPos:p.pos],
		StartPos: startPos,
		EndPos:   p.pos,
	}
}

// parseClosingTag parses a closing tag
func (p *ASTParser) parseClosingTag(_ int) *ASTNode {
	p.consume('/')
	tagName := p.parseTagName()
	if tagName == "" {
		return nil
	}

	if !p.consume('>') {
		return nil
	}

	// Closing tags are typically not stored as separate nodes
	// They're handled during the opening tag parsing
	return nil
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
	if p.peek(2) == "\\\"" {
		p.consume('\\')
		p.consume('"')
		start := p.pos
		for p.pos < p.len {
			if p.peek(2) == "\\\"" {
				p.consume('\\')
				p.consume('"')
				break
			}
			p.pos++
		}
		return p.input[start : p.pos-2]
	}

	if p.peek(1) == "\"" {
		p.consume('"')
		start := p.pos
		for p.pos < p.len && p.peek(1) != "\"" {
			p.pos++
		}
		p.consume('"')
		return p.input[start : p.pos-1]
	} else if p.peek(1) == "'" {
		p.consume('\'')
		start := p.pos
		for p.pos < p.len && p.peek(1) != "'" {
			p.pos++
		}
		p.consume('\'')
		return p.input[start : p.pos-1]
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
		attrStr := ""
		if len(n.Attributes) > 0 {
			var attrs []string
			for k, v := range n.Attributes {
				if v == "" {
					attrs = append(attrs, k)
				} else {
					attrs = append(attrs, fmt.Sprintf(`%s="%s"`, k, v))
				}
			}
			attrStr = " " + strings.Join(attrs, " ")
		}
		return fmt.Sprintf("<%s%s/>", n.TagName, attrStr)
	case NodeTypeTag:
		attrStr := ""
		if len(n.Attributes) > 0 {
			var attrs []string
			for k, v := range n.Attributes {
				if v == "" {
					attrs = append(attrs, k)
				} else {
					attrs = append(attrs, fmt.Sprintf(`%s="%s"`, k, v))
				}
			}
			attrStr = " " + strings.Join(attrs, " ")
		}

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
