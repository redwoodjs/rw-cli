package files

import (
	"os"
	"path/filepath"

	"github.com/redwoodjs/rw-cli/cli/config"
)

func EnsureDotRWExists() error {
	uDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dRWDir := filepath.Join(uDir, config.RW_DIR_NAME)
	if _, err := os.Stat(dRWDir); os.IsNotExist(err) {
		err = os.MkdirAll(dRWDir, 0755)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil

}
