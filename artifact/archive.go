package artifact

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"strings"
)

func extract(archive string, spec runtimeSpec, destination string) error {
	return extractTarGzip(archive, spec.member, destination)
}

func extractTarGzip(archive, member, destination string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer compressed.Close()
	reader := tar.NewReader(compressed)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if strings.TrimPrefix(header.Name, "./") == member && header.Typeflag == tar.TypeReg {
			return writeExtracted(destination, reader, header.Size)
		}
	}
	return errors.New("runtime library is missing from archive")
}

func writeExtracted(path string, source io.Reader, size int64) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(source, size+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		return errors.Join(copyErr, closeErr)
	}
	if written != size {
		return errors.New("runtime archive member has unexpected size")
	}
	return nil
}
