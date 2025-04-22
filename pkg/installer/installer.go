package installer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	frpBaseURL = "https://github.com/fatedier/frp/releases/download/v%s/frp_%s_%s_%s.%s"
)

// EnsureFRPInstalled 确保指定版本的FRP已安装
func EnsureFRPInstalled(binDir, version string) (string, error) {
	// 创建frp目录
	if err := os.MkdirAll(filepath.Join(binDir, version), 0755); err != nil {
		return "", fmt.Errorf("创建frp目录失败: %v", err)
	}

	// 已安装
	frpPath, err := IsFRPInstalled(binDir, version)
	if err == nil {
		return frpPath, nil
	}

	// 获取系统信息
	osType, arch, ext := getSystemInfo()
	if osType == "" || arch == "" {
		return "", fmt.Errorf("不支持的操作系统: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// 下载FRP
	url := fmt.Sprintf(frpBaseURL, version, version, osType, arch, ext)
	if err := downloadAndExtract(url, filepath.Join(binDir, version)); err != nil {
		return "", fmt.Errorf("下载并解压FRP失败: %v", err)
	}

	return filepath.Join(binDir, fmt.Sprintf("frpc-%s%s", version, ext)), nil
}

// IsFRPInstalled 检查指定版本的FRP是否已安装
func IsFRPInstalled(binDir, version string) (string, error) {
	// 检查frpc或frpc.exe是否存在
	frpcPath := filepath.Join(binDir, version, "frpc")
	if runtime.GOOS == "windows" {
		frpcPath += ".exe"
	}

	_, err := os.Stat(frpcPath)
	return frpcPath, err
}

// getSystemInfo 获取系统信息
func getSystemInfo() (osType, arch, ext string) {
	osType = runtime.GOOS
	arch = runtime.GOARCH

	switch osType {
	case "windows":
		ext = "zip"
	case "linux", "darwin":
		ext = "tar.gz"
	default:
		return "", "", ""
	}

	// 转换架构名称
	switch arch {
	case "amd64":
		arch = "amd64"
	case "386":
		arch = "386"
	case "arm64":
		arch = "arm64"
	default:
		return "", "", ""
	}

	return osType, arch, ext
}

// downloadAndExtract 下载并解压FRP
func downloadAndExtract(url, targetDir string) error {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "frp-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// 下载文件
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 保存到临时文件
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}

	// 解压文件
	if strings.HasSuffix(url, ".zip") {
		return extractZip(tmpFile.Name(), targetDir)
	} else if strings.HasSuffix(url, ".tar.gz") {
		return extractTarGz(tmpFile.Name(), targetDir)
	}

	return fmt.Errorf("不支持的文件格式")
}

// extractZip 解压zip文件
func extractZip(zipFile, targetDir string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, "frpc") && !strings.HasSuffix(f.Name, "frpc.exe") {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(targetDir, filepath.Base(f.Name))
		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, rc)
		if err != nil {
			return err
		}
	}

	return nil
}

// extractTarGz 解压tar.gz文件
func extractTarGz(tarFile, targetDir string) error {
	f, err := os.Open(tarFile)
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
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if !strings.HasSuffix(hdr.Name, "frpc") {
			continue
		}

		path := filepath.Join(targetDir, filepath.Base(hdr.Name))
		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(hdr.Mode))
		if err != nil {
			return err
		}
		defer outFile.Close()

		if _, err := io.Copy(outFile, tr); err != nil {
			return err
		}
	}

	return nil
}
