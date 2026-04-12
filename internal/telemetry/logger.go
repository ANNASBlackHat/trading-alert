package telemetry

import (
	"bufio"
	"io"
	"os"
)

// LogProvider defines the agnostic interface for accessing logs
type LogProvider interface {
	GetRecentLogs(lines int) ([]string, error)
	GetLogFile() (io.ReadCloser, error)
}

// FileLogProvider implements LogProvider for file-based logs
type FileLogProvider struct {
	filePath string
}

// NewFileLogProvider creates a new FileLogProvider
func NewFileLogProvider(filePath string) *FileLogProvider {
	return &FileLogProvider{
		filePath: filePath,
	}
}

// GetRecentLogs retrieves the last N lines from the log file
func (f *FileLogProvider) GetRecentLogs(lines int) ([]string, error) {
	if lines <= 0 {
		return []string{}, nil
	}

	file, err := os.Open(f.filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// A simple approach to read the last N lines efficiently.
	// For production we could seek backwards from the end of the file,
	// but a ring buffer reading line-by-line is simpler and generally fast enough
	// for relatively small rotating log files.
	scanner := bufio.NewScanner(file)
	buffer := make([]string, 0, lines)

	for scanner.Scan() {
		if len(buffer) == lines {
			// Shift elements left by 1 and append to the end
			copy(buffer, buffer[1:])
			buffer[lines-1] = scanner.Text()
		} else {
			buffer = append(buffer, scanner.Text())
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return buffer, nil
}

// GetLogFile returns an io.ReadCloser to stream the entire file
func (f *FileLogProvider) GetLogFile() (io.ReadCloser, error) {
	return os.Open(f.filePath)
}
