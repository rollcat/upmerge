// upmerge - maintain local changes to /etc on macOS (and maybe other systems) across
// upgrades. Check the readme.md for usage instructions.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	getopt "github.com/timtadh/getopt"
)

var (
	logInfo   = log.New(ioutil.Discard, "", 0)
	logError  = log.New(os.Stderr, "", 0)
	destDir   = "/etc"
	srcDir    = "/usr/local/upmerge/etc"
	dryRun    = false
	errRefuse = errors.New("refusing operation")
	progName  = path.Base(os.Args[0])
)

const (
	backupSuffix = ".upmerge~"
)

func errUsage() {
	fmt.Printf("Usage: %s [-hnv] [-s src] [-d dest]\n", progName)
	os.Exit(1)
}

func help() {
	fmt.Printf("Usage: %s [-hnv] [-s src] [-d dest]\n", progName)
	fmt.Printf("Maintain local overrides to /etc.\n")
	fmt.Printf("Flags:\n")
	fmt.Printf("    -h      Show this help and exit\n")
	fmt.Printf("    -n      Dry run (don't try making any changes)\n")
	fmt.Printf("    -v      Be verbose\n")
	fmt.Printf("    -s dir  Use dir (default /usr/local/upmerge/etc) as the source\n")
	fmt.Printf("    -d dir  Use dir (default /etc) as the destination\n")
}

// fileContentsAreIdentical returns true if the contents of files named by path1 and
// path2 are identical.
func fileContentsAreIdentical(path1, path2 string) (bool, error) {
	// Take a shortcut: if the files have different sizes, they must be different.
	s1, err := os.Stat(path1)
	if err != nil {
		return false, err
	}
	s2, err := os.Stat(path2)
	if err != nil {
		return false, err
	}
	if s1.Size() != s2.Size() {
		return false, nil
	}
	// TODO: don't read the whole file at once, compare slice by slice.
	buf1, err := os.ReadFile(path1)
	if err != nil {
		return false, err
	}
	buf2, err := os.ReadFile(path1)
	if err != nil {
		return false, err
	}
	return bytes.Equal(buf1, buf2), nil
}

// copyFile copies named srcPath into destPath, matching permission bits (and applying
// umask). As a precaution, destPath must not exist.
func copyFile(srcPath, destPath string) error {
	st, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	fr, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer fr.Close()
	fw, err := os.OpenFile(destPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, st.Mode())
	if err != nil {
		return err
	}
	defer fw.Close()
	_, err = io.Copy(fw, fr)
	return err
}

func main() {
	args, opts, err := getopt.GetOpt(os.Args[1:], "hnvs:d:", nil)
	if err != nil || len(args) != 0 {
		errUsage()
		return
	}
	for _, opt := range opts {
		switch opt.Opt() {
		case "-h":
			help()
			os.Exit(0)
		case "-n":
			dryRun = true
		case "-v":
			logInfo = log.New(os.Stderr, "", 0)
		case "-s":
			srcDir = opt.Arg()
		case "-d":
			destDir = opt.Arg()
		default:
			errUsage()
			return
		}
	}

	err = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		var err error
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		srcPath := filepath.Join(srcDir, rel)
		destPath := filepath.Join(destDir, rel)
		if d.IsDir() {
			// Ensure the directory exists in the destination
			st, err := d.Info()
			if err != nil {
				return err
			}
			err = os.Mkdir(destPath, st.Mode())
			if err == nil {
				logInfo.Printf("MKDIR:\t%s", destPath)
				return nil
			}
			if os.IsExist(err) {
				return nil
			}
			return err
		}
		if strings.HasSuffix(srcPath, "~") {
			logInfo.Printf("IGNORE:\t%s", srcPath)
			return nil
		}
		if _, err = os.Stat(destPath); os.IsNotExist(err) {
			if !dryRun {
				if err = copyFile(srcPath, destPath); err != nil {
					return err
				}
			}
			logInfo.Printf("COPY:\t%s <- %s", destPath, srcPath)
			// There shouldn't be a need to check for the backup here.
			return nil
		}

		backupPath := fmt.Sprintf("%s%s", destPath, backupSuffix)
		same, err := fileContentsAreIdentical(srcPath, destPath)
		if err != nil {
			return err
		}
		if same {
			logInfo.Printf("OK:\t%s <- %s", destPath, srcPath)
			same, _ = fileContentsAreIdentical(destPath, backupPath)
			if !same {
				// destination is up to date with source, but there's still a backup
				// with contents different from our version.
				logInfo.Printf("CHECK:\t%s", backupPath)
			}
			return nil
		}
		if !dryRun {
			same, _ = fileContentsAreIdentical(destPath, backupPath)
			if !same {
				logError.Printf("ERROR:\trefusing to overwrite backup: %s\n", backupPath)
				return errRefuse
			}
			if err = os.Rename(destPath, backupPath); err != nil {
				return err
			}
		}
		logInfo.Printf("MOVE:\t%s <- %s", backupPath, destPath)
		if !dryRun {
			if err = copyFile(srcPath, destPath); err != nil {
				return err
			}
		}
		logInfo.Printf("COPY:\t%s <- %s", destPath, srcPath)
		return nil
	})

	if err != nil {
		logError.Printf("%s: %s\n", progName, err)
		os.Exit(2)
	}
}
