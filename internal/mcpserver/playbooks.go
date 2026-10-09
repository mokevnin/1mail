package mcpserver

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// playbookFiles are the shipped playbooks (ADR 0016, phase 3): Markdown files
// with a one-line description in front matter, served as MCP prompts. They are
// content, not code: edit the files. A playbook names a tool as `tool_name()`,
// and a test checks every such name against the generated tool surface.
//
//go:embed playbooks/*.md
var playbookFiles embed.FS

type playbook struct {
	name        string
	description string
	body        string
}

// loadPlaybooks parses the embedded playbooks; the file name is the prompt name.
func loadPlaybooks() ([]playbook, error) {
	entries, err := fs.ReadDir(playbookFiles, "playbooks")
	if err != nil {
		return nil, err
	}
	out := make([]playbook, 0, len(entries))
	for _, e := range entries {
		raw, err := playbookFiles.ReadFile("playbooks/" + e.Name())
		if err != nil {
			return nil, err
		}
		head, body, ok := strings.Cut(strings.TrimPrefix(string(raw), "---\n"), "\n---\n")
		description, found := strings.CutPrefix(head, "description: ")
		if !ok || !found {
			return nil, fmt.Errorf("playbook %s: want front matter with a description", e.Name())
		}
		out = append(out, playbook{
			name:        strings.TrimSuffix(e.Name(), ".md"),
			description: strings.TrimSpace(description),
			body:        strings.TrimSpace(body),
		})
	}
	return out, nil
}

func (p playbook) prompt() *mcp.Prompt {
	return &mcp.Prompt{Name: p.name, Title: p.name, Description: p.description}
}

func (p playbook) handler(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: p.description,
		Messages:    []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: p.body}}},
	}, nil
}
