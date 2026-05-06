package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/zip"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
)

const (
	writeBufSize      = 4 << 20 // 4 MB write buffer
	largeFileMaxBytes = 4 << 20 // files larger than this are streamed, not buffered
)

// ── parallel read pipeline (shared by TarGzip and TarZstd) ───────────────────

type readResult struct {
	path string
	info fs.FileInfo
	link string // non-empty for symlinks
	data []byte // body of regular files; nil for dirs/symlinks
	err  error
}

type fileRef struct {
	path string
	info fs.FileInfo
}

// collectFiles walks all artifact glob patterns and returns file metadata.
func collectFiles(artifacts []string) ([]fileRef, error) {
	var files []fileRef
	for _, pattern := range artifacts {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			if err := filepath.WalkDir(match, func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				info, err := d.Info()
				if err != nil {
					return err
				}
				files = append(files, fileRef{path, info})
				return nil
			}); err != nil {
				return nil, err
			}
		}
	}
	return files, nil
}

// tarPackFiles reads files in parallel and writes tar entries to tw in order.
// windowSem bounds the number of reads that are in-flight ahead of the writer,
// preventing unbounded memory growth when the writer is slower than the readers.
func tarPackFiles(tw *tar.Writer, artifacts []string) error {
	files, err := collectFiles(artifacts)
	if err != nil || len(files) == 0 {
		return err
	}

	resultChans := make([]chan readResult, len(files))
	for i := range resultChans {
		resultChans[i] = make(chan readResult, 1)
	}

	windowSem := make(chan struct{}, runtime.NumCPU()*2)
	done := make(chan struct{})
	writeErrCh := make(chan error, 1)

	// Writer goroutine: drains resultChans in order, releases windowSem slots.
	go func() {
		defer close(done)
		for _, ch := range resultChans {
			r := <-ch
			<-windowSem
			if r.err != nil {
				writeErrCh <- r.err
				return
			}
			if err := writeTarEntry(tw, r); err != nil {
				writeErrCh <- err
				return
			}
		}
		writeErrCh <- nil
	}()

	// Dispatcher: fills the pipeline; stops early if the writer exits.
dispatch:
	for i, f := range files {
		select {
		case windowSem <- struct{}{}:
		case <-done:
			break dispatch
		}
		go func(idx int, ref fileRef) {
			r := readResult{path: ref.path, info: ref.info}
			if ref.info.Mode()&os.ModeSymlink != 0 {
				r.link, r.err = os.Readlink(ref.path)
			} else if ref.info.Mode().IsRegular() {
				r.data, r.err = os.ReadFile(ref.path)
			}
			resultChans[idx] <- r
		}(i, f)
	}

	return <-writeErrCh
}

func writeTarEntry(tw *tar.Writer, r readResult) error {
	header, err := tar.FileInfoHeader(r.info, r.link)
	if err != nil {
		return err
	}
	header.Name = r.path
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if r.info.Mode().IsRegular() {
		_, err = tw.Write(r.data)
	}
	return err
}

// tarUnpackFiles extracts a tar stream. Small file writes run in a goroutine
// pool; files larger than largeFileMaxBytes are written synchronously.
func tarUnpackFiles(tr *tar.Reader) error {
	sem := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := header.Name
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if header.Size > largeFileMaxBytes {
				// Stream large files directly; no buffering.
				out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fs.FileMode(header.Mode))
				if err != nil {
					return err
				}
				if _, err := io.Copy(out, tr); err != nil {
					out.Close()
					return err
				}
				out.Close()
			} else {
				data, err := io.ReadAll(tr)
				if err != nil {
					return err
				}
				wg.Add(1)
				sem <- struct{}{}
				go func(name string, mode fs.FileMode, content []byte) {
					defer wg.Done()
					defer func() { <-sem }()
					if err := os.WriteFile(name, content, mode); err != nil {
						errOnce.Do(func() { firstErr = err })
					}
				}(target, fs.FileMode(header.Mode), data)
			}

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		}
	}

	wg.Wait()
	return firstErr
}

// ── TarGzip ───────────────────────────────────────────────────────────────────

// TarGzip creates a .tar.gz archive. pgzip compresses across all CPU cores.
func TarGzip(filename string, artifacts []string) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	bw := bufio.NewWriterSize(outFile, writeBufSize)
	gzw, err := pgzip.NewWriterLevel(bw, pgzip.BestSpeed)
	if err != nil {
		return err
	}

	tw := tar.NewWriter(gzw)
	packErr := tarPackFiles(tw, artifacts)
	if cerr := tw.Close(); packErr == nil {
		packErr = cerr
	}
	if cerr := gzw.Close(); packErr == nil {
		packErr = cerr
	}
	if cerr := bw.Flush(); packErr == nil {
		packErr = cerr
	}
	return packErr
}

// UntarGzip extracts a .tar.gz archive.
func UntarGzip(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := pgzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return tarUnpackFiles(tar.NewReader(gzr))
}

// ── TarZstd ───────────────────────────────────────────────────────────────────

// TarZstd creates a .tar.zst archive. The zstd encoder uses all CPU cores.
func TarZstd(filename string, artifacts []string) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	bw := bufio.NewWriterSize(outFile, writeBufSize)
	enc, err := zstd.NewWriter(bw,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderConcurrency(runtime.NumCPU()),
	)
	if err != nil {
		return err
	}

	tw := tar.NewWriter(enc)
	packErr := tarPackFiles(tw, artifacts)
	if cerr := tw.Close(); packErr == nil {
		packErr = cerr
	}
	if cerr := enc.Close(); packErr == nil {
		packErr = cerr
	}
	if cerr := bw.Flush(); packErr == nil {
		packErr = cerr
	}
	return packErr
}

// UntarZstd extracts a .tar.zst archive.
func UntarZstd(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	dec, err := zstd.NewReader(f, zstd.WithDecoderConcurrency(runtime.NumCPU()))
	if err != nil {
		return err
	}
	defer dec.Close()

	return tarUnpackFiles(tar.NewReader(dec))
}

// ── Zip ───────────────────────────────────────────────────────────────────────

// zipFileResult holds the outcome of compressing a single file in a goroutine.
type zipFileResult struct {
	header   *zip.FileHeader
	isDir    bool
	compData []byte
	err      error
}

// Zip archives artifacts to a .zip file. Regular file contents are deflate-
// compressed in parallel; the zip stream is written sequentially in walk order.
func Zip(filename string, artifacts []string) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	zw := zip.NewWriter(outFile)

	files, err := collectFiles(artifacts)
	if err != nil {
		zw.Close()
		return err
	}
	if len(files) == 0 {
		return zw.Close()
	}

	resultChans := make([]chan zipFileResult, len(files))
	for i := range resultChans {
		resultChans[i] = make(chan zipFileResult, 1)
	}

	windowSem := make(chan struct{}, runtime.NumCPU()*2)
	done := make(chan struct{})
	writeErrCh := make(chan error, 1)

	// Writer goroutine: writes zip entries in order.
	go func() {
		defer close(done)
		for _, ch := range resultChans {
			r := <-ch
			<-windowSem
			if r.err != nil {
				writeErrCh <- r.err
				return
			}
			if r.isDir {
				if _, err := zw.CreateHeader(r.header); err != nil {
					writeErrCh <- err
					return
				}
				continue
			}
			if err := writeZipEntry(zw, r.header, r.compData); err != nil {
				writeErrCh <- err
				return
			}
		}
		writeErrCh <- nil
	}()

dispatch:
	for i, f := range files {
		select {
		case windowSem <- struct{}{}:
		case <-done:
			break dispatch
		}
		go func(idx int, ref fileRef) {
			header, err := zip.FileInfoHeader(ref.info)
			if err != nil {
				resultChans[idx] <- zipFileResult{err: err}
				return
			}
			header.Name = ref.path

			if !ref.info.Mode().IsRegular() {
				resultChans[idx] <- zipFileResult{header: header, isDir: true}
				return
			}

			header.Method = zip.Deflate
			data, err := os.ReadFile(ref.path)
			if err != nil {
				resultChans[idx] <- zipFileResult{err: err}
				return
			}

			header.CRC32 = crc32.ChecksumIEEE(data)
			header.UncompressedSize64 = uint64(len(data))

			var buf bytes.Buffer
			fw, err := flate.NewWriter(&buf, flate.BestSpeed)
			if err != nil {
				resultChans[idx] <- zipFileResult{err: err}
				return
			}
			if _, err := fw.Write(data); err != nil {
				fw.Close()
				resultChans[idx] <- zipFileResult{err: err}
				return
			}
			fw.Close()

			header.CompressedSize64 = uint64(buf.Len())
			resultChans[idx] <- zipFileResult{header: header, compData: buf.Bytes()}
		}(i, f)
	}

	zipErr := <-writeErrCh
	if cerr := zw.Close(); zipErr == nil {
		zipErr = cerr
	}
	return zipErr
}

// writeZipEntry writes a pre-compressed file entry via CreateHeaderRaw.
func writeZipEntry(zw *zip.Writer, header *zip.FileHeader, compData []byte) error {
	w, err := zw.CreateHeaderRaw(header)
	if err != nil {
		return err
	}
	_, err = w.Write(compData)
	return err
}

// Unzip - Unzip all files and directories inside .zip file using a worker pool.
func Unzip(filename string) error {
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create all directories up-front to avoid races between goroutines.
	for _, file := range reader.File {
		if err := os.MkdirAll(filepath.Dir(file.Name), os.ModePerm); err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(file.Name, os.ModePerm); err != nil {
				return err
			}
		}
	}

	sem := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(zf *zip.File) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := extractZipFile(zf); err != nil {
				errOnce.Do(func() { firstErr = err })
			}
		}(f)
	}

	wg.Wait()
	return firstErr
}

func extractZipFile(f *zip.File) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(f.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
