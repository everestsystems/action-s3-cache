package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
)

//nolint:funlen // Keep the existing archive creation flow; refactor in a follow-up.
func Tar(filename string, artifacts []string) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	gzw := gzip.NewWriter(outFile)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for _, pattern := range artifacts {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return err
		}

		for _, match := range matches {
			filepath.Walk(match, func(file string, fi os.FileInfo, err error) error {
				// return on any error
				if err != nil {
					return err
				}

				// for symbolic link only add to the tar file list
				link := file
				if fi.Mode()&os.ModeSymlink != 0 {
					var err error
					link, err = os.Readlink(file)
					if err != nil {
						return err
					}
				}

				// create a new dir/file header
				header, err := tar.FileInfoHeader(fi, link)
				if err != nil {
					return err
				}
				header.Name = file

				// write the header
				if err := tw.WriteHeader(header); err != nil {
					return err
				}

				// copy file content only for regular files
				if fi.Mode().IsRegular() {
					// open files for taring
					f, err := os.Open(file)
					if err != nil {
						return err
					}

					// copy file data into tar writer
					if _, err := io.Copy(tw, f); err != nil {
						return err
					}

					// manually close here after each file operation; defering would cause each file close
					// to wait until all operations have completed.
					f.Close()
				}

				return nil
			})
		}
	}

	return nil
}

//nolint:funlen // Keep the existing archive extraction flow; refactor in a follow-up.
func Untar(filename string) error {
	f, err := os.Open(filename)

	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {
		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := header.Name

		// check the file type
		switch header.Typeflag {
		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			dir := filepath.Dir(target)
			if _, err := os.Stat(dir); err != nil {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()

		case tar.TypeSymlink:
			dir := filepath.Dir(target)
			if _, err := os.Stat(dir); err != nil {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
			}

			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		}
	}
}
