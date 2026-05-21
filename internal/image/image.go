package image

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const imagesBase = "/var/lib/krate/images"

var knownImages = map[string]string{
	"alpine":      "https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.1-x86_64.tar.gz",
	"alpine:3.19": "https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.1-x86_64.tar.gz",
	"alpine:3.18": "https://dl-cdn.alpinelinux.org/alpine/v3.18/releases/x86_64/alpine-minirootfs-3.18.6-x86_64.tar.gz",
}

type ImageInfo struct {
	Name string
	Size string
}

func Pull(name string) error {
	url, ok := knownImages[name]
	if !ok {
		return fmt.Errorf("unknown image: %s\navailable: alpine, alpine:3.19, alpine:3.18", name)
	}

	imgPath := filepath.Join(imagesBase, sanitize(name))
	if _, err := os.Stat(imgPath); err == nil {
		fmt.Printf("Image '%s' already exists\n", name)
		return nil
	}

	fmt.Printf("Pulling %s...\n", name)
	if err := os.MkdirAll(imgPath, 0755); err != nil {
		return err
	}

	resp, err := http.Get(url)
	if err != nil {
		os.RemoveAll(imgPath)
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if err := extractTarGz(resp.Body, imgPath); err != nil {
		os.RemoveAll(imgPath)
		return fmt.Errorf("extract: %w", err)
	}

	fmt.Printf("Pulled '%s' successfully\n", name)
	return nil
}

func Ensure(name string) (string, error) {
	imgPath := filepath.Join(imagesBase, sanitize(name))
	if _, err := os.Stat(imgPath); os.IsNotExist(err) {
		if err := Pull(name); err != nil {
			return "", err
		}
	}
	return imgPath, nil
}

func List() ([]ImageInfo, error) {
	entries, err := os.ReadDir(imagesBase)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var imgs []ImageInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		size := dirSize(filepath.Join(imagesBase, e.Name()))
		imgs = append(imgs, ImageInfo{Name: e.Name(), Size: humanSize(size)})
	}
	return imgs, nil
}

func sanitize(name string) string {
	return strings.ReplaceAll(name, ":", "-")
}

func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func extractTarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, hdr.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, os.FileMode(hdr.Mode))
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				continue
			}
			io.Copy(f, tr)
			f.Close()
		case tar.TypeSymlink:
			os.Symlink(hdr.Linkname, target)
		case tar.TypeLink:
			os.Link(filepath.Join(dest, hdr.Linkname), target)
		}
	}
	return nil
}