package toolgen

// TagFilter filters operations by tag, respecting x-mcp-hidden.
type TagFilter struct {
	tag string
}

// NewTagFilter returns a filter for the given tag.
func NewTagFilter(tag string) *TagFilter {
	return &TagFilter{tag: tag}
}

// Include returns true if operation should be included.
// Included if x-mcp-hidden is false, or unset with matching tag.
func (f *TagFilter) Include(opTags []string, opExt, pathExt map[string]any) bool {
	ext := ExtractExtensions(opExt, pathExt)
	if ext.Hidden != nil {
		return !*ext.Hidden
	}
	for _, tag := range opTags {
		if tag == f.tag {
			return true
		}
	}
	return false
}
