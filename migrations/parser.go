package migrations

import (
	"bufio"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

// TODO: put in order.
// TODO: very fragile code, fix it.

const (
	delimUp   = `-- delimUp`
	delimDown = `-- delimDown`

	migrationExt = `.sql`
)

func parse(f fs.File) (*Migration, error) {
	scan := bufio.NewScanner(f)

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("f.Stat: %w", err)
	}

	fName := info.Name()
	ext := filepath.Ext(fName)
	if ext != migrationExt {
		return nil, ErrInvalidMigrationExt
	}

	slice := strings.Split(fName, ".")
	if len(slice) != 3 {
		return nil, ErrInvalidMigrationName
	}

	version, err := strconv.Atoi(slice[0])
	if err != nil {
		return nil, fmt.Errorf("strconv.Atoi: %w", err)
	}

	migrationName := slice[1]

	m := Migration{
		Version: uint(version),
		Name:    migrationName,
	}

	currentParse := 0
	for scan.Scan() {
		str := scan.Text()
		switch str {
		case delimUp:
			currentParse = 1
			continue
		case delimDown:
			currentParse = 2
			continue
		}

		switch currentParse {
		case 1:
			m.Up += str
		case 2:
			m.Down += str
		}
	}

	err = scan.Err()
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	return &m, nil
}
