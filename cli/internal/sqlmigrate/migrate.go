package sqlmigrate

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed sql/memory/*.sql sql/graph/*.sql sql/dolt/*.sql
var embeddedFS embed.FS

// MaxVersion is the highest migration version this binary knows for a store kind.
func MaxVersion(kind string) (int, error) {
	files, err := listMigrations(kind)
	if err != nil {
		return 0, err
	}
	if len(files) == 0 {
		return 0, nil
	}
	return files[len(files)-1].Version, nil
}

type migrationFile struct {
	Version  int
	Filename string
	SQL      string
}

func listMigrations(kind string) ([]migrationFile, error) {
	dir := path.Join("sql", kind)
	entries, err := fs.ReadDir(embeddedFS, dir)
	if err != nil {
		return nil, err
	}
	var out []migrationFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		ver, err := parseVersion(e.Name())
		if err != nil {
			return nil, fmt.Errorf("sqlmigrate: %s: %w", e.Name(), err)
		}
		b, err := fs.ReadFile(embeddedFS, path.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, migrationFile{Version: ver, Filename: e.Name(), SQL: string(b)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func parseVersion(name string) (int, error) {
	i := strings.Index(name, "_")
	if i < 1 {
		return 0, fmt.Errorf("expected NNN_name.sql")
	}
	return strconv.Atoi(name[:i])
}

// ErrSchemaNewer is returned when the database schema version exceeds the binary.
type ErrSchemaNewer struct {
	Store   string
	Have    int
	Binary  int
}

func (e *ErrSchemaNewer) Error() string {
	return fmt.Sprintf("fabrik: %s schema %d is newer than this fabrik (max %d); upgrade fabrik", e.Store, e.Have, e.Binary)
}

// AppliedAt returns RFC3339 UTC now.
func AppliedAt() string {
	return time.Now().UTC().Format(time.RFC3339)
}
