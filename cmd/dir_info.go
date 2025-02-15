package cmd

import "path/filepath"

// DirInfo tracks information about a directory
type DirInfo struct {
	HasUnmanaged bool
	ManagedFiles map[string]bool
	IsManaged    bool
}

// InitDir initializes directory tracking information
func InitDir(dir string, dirMap map[string]*DirInfo) *DirInfo {
	if info, exists := dirMap[dir]; exists {
		return info
	}
	info := &DirInfo{
		HasUnmanaged: false,
		ManagedFiles: make(map[string]bool),
		IsManaged:    false,
	}
	dirMap[dir] = info

	// Initialize parent directory if it exists
	if dir != "overlay" {
		parent := filepath.Dir(dir)
		InitDir(parent, dirMap)
	}
	return info
}
