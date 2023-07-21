package subcmd

import (
	"encoding/json"
	"errors"
	"os"
)

type OwnershipManager struct {
	ownJson string
	force   bool
}

// takeOwnership takes ownership of the
func (o OwnershipManager) EnsureOwnership(files []string, dot string) error {
	data, err := os.ReadFile(o.ownJson)
	if err != nil {
		return err
	}

	var dotOwn map[string][]string
	err = json.Unmarshal(data, &dotOwn)
	if err != nil {
		return err
	}

	dotOwn[dot], err = o.ensureOwnershipDot(files, dotOwn[dot])
	if err != nil {
		return err
	}

	data, err = json.Marshal(dotOwn)
	if err != nil {
		return err
	}

	err = os.WriteFile(o.ownJson, data, 0666)

	return err
}

func (o OwnershipManager) OwnedFiles() (map[string][]string, error) {
	data, err := os.ReadFile(o.ownJson)
	if err != nil {
		return nil, err
	}

	var dotOwn map[string][]string
	err = json.Unmarshal(data, &dotOwn)
	if err != nil {
		return nil, err
	}

	return dotOwn, nil
}

func (o OwnershipManager) DisownDot(dot string) error {
	data, err := os.ReadFile(o.ownJson)
	if err != nil {
		return err
	}

	var dotOwn map[string][]string
	err = json.Unmarshal(data, &dotOwn)
	if err != nil {
		return err
	}

	delete(dotOwn, dot)

	data, err = json.Marshal(dotOwn)
	if err != nil {
		return err
	}

	return os.WriteFile(o.ownJson, data, 0666)
}

// ensureOwnershipDot checks that the current directory owns the dots it is
// trying to possess, or take them by force if that's allowed. It returns a new
// list of the owned files and an error that is non-nil if something goes wrong.
func (o OwnershipManager) ensureOwnershipDot(
	files []string,
	ownedFiles []string,
) ([]string, error) {
	ownedFileSet := make(map[string]struct{})
	for _, file := range ownedFiles {
		ownedFileSet[file] = struct{}{}
	}

	for _, file := range files {
		_, owned := ownedFileSet[file]

		err := o.ensureOwnershipFile(file, owned)
		if err != nil {
			return nil, err
		}

		if !owned {
			ownedFiles = append(ownedFiles, file)
		}
	}

	return ownedFiles, nil
}

// ensureOwnershipFile ensures ownership of a single file. It does this by
// deleting the file if it is owned or forced. If something goes wrong doing so,
// or a file is not owned and ownership cannot be forced, a non-nil error is
// returned.
func (o OwnershipManager) ensureOwnershipFile(file string, owned bool) error {
	var fileExists bool
	if _, err := os.Stat(file); err == nil {
		fileExists = true
	} else if errors.Is(err, os.ErrNotExist) {
		fileExists = false
	} else {
		return err
	}

	if !fileExists {
		return nil
	} else if owned || o.force {
		return os.RemoveAll(file)
	} else {
		return errors.New(
			file + " exists but is not owned by this directory",
		)
	}
}
