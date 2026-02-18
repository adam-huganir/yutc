package templates

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/adam-huganir/yutc/pkg/loader"
	"github.com/rs/zerolog"
)

// Info holds metadata for template file renaming.
type Info struct {
	NewName string // For templates, if we are renaming the file, this is the new name
}

// ContainerInfo holds the parent/root/children tree for directory-based template inputs.
type ContainerInfo struct {
	Parent   *Input   // Parent of the file if it is a directory or archive
	Root     *Input   // Root of the file if it is a directory or archive
	children []*Input // Children of the file if it is a directory or archive
}

// Input represents a template or common/shared template file.
type Input struct {
	*loader.FileEntry
	Template  Info
	Container ContainerInfo
	IsCommon  bool // true if this is a common/shared template
}

// NewInput creates an Input with the given name and FileEntry options.
func NewInput(name string, isCommon bool, opts ...loader.FileEntryOption) *Input {
	fe := loader.NewFileEntry(name, opts...)
	return &Input{
		FileEntry: fe,
		IsCommon:  isCommon,
	}
}

// TemplateName resolves the template name by executing the filename as a template with the given data.
func (ti *Input) TemplateName(t *template.Template, data map[string]any) (string, error) {
	if ti.Template.NewName != "" {
		return ti.Template.NewName, nil
	}
	newName := bytes.NewBufferString("")
	t, err := t.New(ti.Name).Parse(ti.Name)
	if err != nil {
		return "", err
	}
	if err := t.ExecuteTemplate(newName, ti.Name, data); err != nil {
		return "", err
	}
	ti.Template.NewName = newName.String()
	return ti.Template.NewName, nil
}

// RelativePath returns the relative path of the file from its root container.
func (ti *Input) RelativePath() (string, error) {
	if ti.Container.Root == nil || ti.Container.Root == ti {
		return filepath.Base(ti.Name), nil
	}
	n := filepath.FromSlash(ti.Name)
	rn := filepath.FromSlash(ti.Container.Root.Name)
	return filepath.Rel(rn, n)
}

// RelativeNewPath returns the relative path of the file from its root using NewName if available.
func (ti *Input) RelativeNewPath() (string, error) {
	name := ti.Name
	if ti.Template.NewName != "" {
		name = ti.Template.NewName
	}
	if ti.Container.Root == nil || ti.Container.Root == ti {
		return filepath.Base(name), nil
	}
	n := filepath.FromSlash(name)
	rn := filepath.FromSlash(ti.Container.Root.Name)
	return filepath.Rel(rn, n)
}

// AllChildren returns all descendant Input entries (flattened).
func (ti *Input) AllChildren() []*Input {
	if ti.Container.children == nil {
		return nil
	}
	return unravelTemplateChildren(ti)
}

func unravelTemplateChildren(ti *Input) []*Input {
	if ti.Container.children == nil {
		return []*Input{}
	}
	children := make([]*Input, 0)
	for _, child := range ti.Container.children {
		children = append(children, child)
		children = append(children, unravelTemplateChildren(child)...)
	}
	return children
}

// CollectContainerChildren populates the children of a container (directory or archive) Input.
func (ti *Input) CollectContainerChildren() error {
	if ic, err := ti.IsContainer(); err != nil || !ic {
		if !ic {
			return fmt.Errorf("file %s is not a container", ti.Name)
		}
		return err
	}
	if ti.Container.children != nil {
		return nil
	}
	ti.Container.children = make([]*Input, 0)
	entries, err := loader.GetEntries(ti.FileEntry, ti.Logger())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		child := &Input{
			FileEntry: entry,
			IsCommon:  ti.IsCommon,
		}
		child.Container.Parent = ti
		if ti.Container.Root != nil {
			child.Container.Root = ti.Container.Root
		} else {
			child.Container.Root = ti
		}
		ti.Container.children = append(ti.Container.children, child)
	}
	return nil
}

// LoadContainer recursively loads all children of a container Input.
func (ti *Input) LoadContainer() error {
	err := ti.CollectContainerChildren()
	if err != nil {
		return err
	}
	if ti.Container.children != nil {
		for _, child := range ti.Container.children {
			if isContainer, err := child.IsContainer(); err != nil {
				return err
			} else if isContainer {
				err = child.LoadContainer()
				if err != nil {
					return err
				}
			} else {
				err = child.Load()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// TemplateFilenames resolves template-based filenames for a list of Input entries.
func TemplateFilenames(fas []*Input, t *template.Template, data map[string]any) error {
	for _, fa := range fas {
		_, err := fa.TemplateName(t, data)
		if err != nil {
			return err
		}
	}
	return nil
}

// ResolveTemplatePaths parses template path strings, loads their content, and expands directories.
func ResolveTemplatePaths(paths []string, isCommon bool, logger *zerolog.Logger) ([]*Input, error) {
	var outFiles []*Input
	for _, p := range paths {
		ti, err := ParseTemplateArg(p, isCommon)
		if err != nil {
			return nil, err
		}
		ti.SetLogger(logger)
		err = ti.Load()
		if err != nil && !errors.Is(err, loader.ErrIsContainer) {
			return nil, err
		} else if err != nil {
			err = ti.LoadContainer()
			if err != nil {
				return nil, err
			}
		}
		outFiles = append(outFiles, ti)
	}
	return outFiles, nil
}

// CountTemplateRecursables counts the number of recursable (directory or archive) items in the Input list.
func CountTemplateRecursables(paths []*Input) (int, error) {
	recursables := 0
	for _, f := range paths {
		if f.Source != loader.SourceKindFile {
			if f.Source == loader.SourceKindURL {
				if loader.IsArchive(f.Name) {
					recursables++
				}
			}
			continue
		}
		isDir, err := loader.IsDir(f.Name)
		if err != nil {
			return recursables, err
		} else if isDir || loader.IsArchive(f.Name) {
			recursables++
		}
	}
	return recursables, nil
}
