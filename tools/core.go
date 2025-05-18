package tools

import "path/filepath"

type Parameter struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

func (p *Parameter) Json() any {
	return map[string]any{
		"description": p.Description,
		"type":        p.Type,
	}
}

type Tool interface {
	Name() string
	Description() string
	Parameters() []*Parameter
	Returns() []*Parameter
	Run(args map[string]any) (any, error)
}

type BaseFileTool struct {
	ProjectRoot string
}

func (b *BaseFileTool) ResolvePath(path string) (fullpath string, err error) {
	fullpath, err = filepath.Abs(filepath.Join(b.ProjectRoot, path))
	return
}
