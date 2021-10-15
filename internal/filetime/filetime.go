package filetime

import (
	"os"
)

func Copy(dstPath, srcPath string) (err error) {
	fileInfo, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	return os.Chtimes(dstPath, fileInfo.ModTime(), fileInfo.ModTime())
	// TODO creation time, which is OS dependant
}
