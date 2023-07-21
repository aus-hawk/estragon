package subcmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type DotfileUndeployer struct {
	own OwnershipManager
	dry bool
}

func (d DotfileUndeployer) Undeploy(dot string) error {
	owned, err := d.own.OwnedFiles()
	if err != nil {
		return err
	}

	fmt.Println("Removing files for dot", dot)
	files, ok := owned[dot]

	if !ok {
		return nil
	}

	for _, file := range files {
		fmt.Println("  Removing file", file)
		if !d.dry {
			err = os.Remove(file)
			if err != nil {
				return err
			}

			err = removeEmptyParents(file)
			if err != nil {
				return err
			}
		}
	}

	if !d.dry {
		d.own.DisownDot(dot)
	} else {
		fmt.Println()
		fmt.Println("Directories that would be empty after these removals")
		fmt.Println("as well as their parents will also be removed")
	}

	return err
}

func removeEmptyParents(file string) error {
	dir := filepath.Dir(file)
	for dir != "." {
		dirFile, err := os.Open(dir)
		if err != nil {
			return err
		}
		defer dirFile.Close()

		_, err = dirFile.Readdirnames(1)
		if errors.Is(err, io.EOF) {
			// Folder is empty.
			fmt.Println("  Removing empty directory", dir)
			err = os.Remove(dir)
			if err != nil {
				return err
			}
		} else {
			return err
		}

		dir = filepath.Dir(dir)
	}
	return nil
}
