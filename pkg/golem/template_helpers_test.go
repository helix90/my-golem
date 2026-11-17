package golem

// processTemplateForTest runs template through the tree processor, building a
// context on the knowledge base when the test doesn't supply one
func processTemplateForTest(g *Golem, template string, wildcards map[string]string, ctx *VariableContext) string {
	if ctx == nil {
		g.rlockKnowledgeBase()
		ctx = &VariableContext{
			LocalVars:     make(map[string]string),
			KnowledgeBase: g.aimlKB,
		}
		g.kbMu.RUnlock()
	}
	return g.processTemplateWithContext(template, wildcards, ctx)
}
