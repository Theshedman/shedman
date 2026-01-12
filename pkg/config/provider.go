package config

// SourceProvider defines the interface for retrieving the original content of a configuration file
// from its package source (e.g., package cache).
type SourceProvider interface {
	// GetOriginalContent retrieves the original content of the file at the given absolute path.
	// It returns the content bytes or an error if the content cannot be retrieved.
	GetOriginalContent(filePath string) ([]byte, error)

	// GetOwner returns the name of the package that owns the given file.
	GetOwner(filePath string) (string, error)
}
