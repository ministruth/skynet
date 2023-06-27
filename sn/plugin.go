package sn

import (
	"plugin"

	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
)

var (
	ErrPluginNotFound    = tracerr.New("plugin not found")
	ErrPluginIDDuplicate = tracerr.New("plugin ID duplicated")
	ErrPluginInvalid     = tracerr.New("plugin invalid")
	ErrPluginLoaded      = tracerr.New("plugin already loaded")
)

type PluginEntry struct {
	ID      uuid.UUID `json:"id" yaml:"id"`           // plugin unique ID
	Name    string    `json:"name" yaml:"name"`       // plugin name, unique suggested
	Version string    `json:"version" yaml:"version"` // plugin version

	Path     string         `json:"path" yaml:"-"`    // runtime relative path
	Enable   bool           `json:"enable" yaml:"-"`  // is plugin enabled
	LibPath  string         `json:"libpath" yaml:"-"` // library path
	Loader   *plugin.Plugin `json:"-" yaml:"-"`       // golang plugin loader
	Instance PluginInstance `json:"-" yaml:"-"`       // plugin instance
}

type Plugin interface {
	Parse(dir string) (*PluginEntry, error)
	Load(id uuid.UUID) error
	Unload()
	Get(id uuid.UUID) *PluginEntry
	GetAll() []*PluginEntry
	Disable(id uuid.UUID) error
	Enable(id uuid.UUID) error
	Delete(id uuid.UUID) error
}

type PluginInstance interface {
	PluginLoad() error
	PluginEnable() error
	PluginDisable() error
	PluginUnload() error
}
