package installer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	frpBaseURL  = "https://github.com/fatedier/frp/releases/download/v%s/frp_%s_%s_%s.%s"
	httpTimeout = 30 * time.Second // HTTP请求超时时间
)

// ProgressWriter 进度写入器
type ProgressWriter struct {
	Total      int64              // 总大小
	Downloaded int64              // 已下载大小
	LastOutput int64              // 上次输出进度时的大小
	OnProgress func(int64, int64) // 进度回调函数
}

// Write 实现io.Writer接口
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.Downloaded += int64(n)

	// 每下载1%或者至少1MB数据更新一次进度
	threshold := pw.Total / 100
	if threshold < 1024*1024 {
		threshold = 1024 * 1024
	}

	if pw.Downloaded-pw.LastOutput >= threshold {
		pw.LastOutput = pw.Downloaded
		if pw.OnProgress != nil {
			pw.OnProgress(pw.Downloaded, pw.Total)
		}
	}

	return n, nil
}

// Installer FRP安装器
type Installer struct {
	BinDir string // FRP二进制文件存储目录
	Proxy  string // GitHub代理前缀，为空则不使用代理
	logger zerolog.Logger
}

// NewInstaller 创建一个新的FRP安装器
func NewInstaller(binDir string, proxy string, logger zerolog.Logger) (*Installer, error) {
	if binDir == "" {
		return nil, fmt.Errorf("binDir is empty")
	}
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("binDir is not exist")
	}
	if proxy != "" {
		proxy = "https://ghfast.top"
	}
	return &Installer{
		BinDir: binDir,
		Proxy:  proxy,
		logger: logger,
	}, nil
}

// GetFRPBinaryPath 根据版本获取FRP二进制文件路径
func (i *Installer) GetFRPBinaryPath(version string) string {
	frpcPath := filepath.Join(i.BinDir, fmt.Sprintf("frpc-%s", version))
	if runtime.GOOS == "windows" {
		frpcPath += ".exe"
	}
	return frpcPath
}

// EnsureFRPInstalled 确保指定版本的FRP已安装
func (i *Installer) EnsureFRPInstalled(version string) (string, error) {
	// 已安装
	frpPath, exists, err := i.IsFRPInstalled(version)
	if err != nil {
		return "", err
	}
	if exists {
		return frpPath, nil
	}

	// 获取系统信息
	osType, arch, ext := i.getSystemInfo()
	if osType == "" || arch == "" {
		return "", fmt.Errorf("不支持的操作系统: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// 下载FRP
	url := fmt.Sprintf(frpBaseURL, version, version, osType, arch, ext)

	// 使用GitHub代理前缀（如果设置）
	if i.Proxy != "" {
		url = strings.Join([]string{i.Proxy, url}, "/")
	}

	if err := i.downloadAndExtract(url, version); err != nil {
		return "", fmt.Errorf("下载并解压FRP失败: %v", err)
	}

	return i.GetFRPBinaryPath(version), nil
}

// IsFRPInstalled 检查指定版本的FRP是否已安装
// 返回值：二进制路径，是否存在，错误信息
func (i *Installer) IsFRPInstalled(version string) (string, bool, error) {
	// 获取二进制路径
	frpcPath := i.GetFRPBinaryPath(version)

	// 检查文件是否存在
	_, err := os.Stat(frpcPath)
	if err == nil {
		return frpcPath, true, nil
	}

	if os.IsNotExist(err) {
		return frpcPath, false, nil
	}

	return frpcPath, false, err
}

// getSystemInfo 获取系统信息
func (i *Installer) getSystemInfo() (osType, arch, ext string) {
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
func (i *Installer) downloadAndExtract(url, version string) error {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "frp-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// 创建带超时的HTTP客户端
	client := &http.Client{
		Timeout: httpTimeout,
	}

	// 创建带上下文的请求
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	// 发送请求
	i.logger.Info().Msgf("正在下载FRP：%s", url)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 获取文件大小
	contentLength := resp.ContentLength

	// 创建进度写入器
	progressWriter := &ProgressWriter{
		Total: contentLength,
		OnProgress: func(downloaded, total int64) {
			percent := float64(downloaded) / float64(total) * 100
			fmt.Printf("\r下载进度: %.2f%% (%d/%d bytes)", percent, downloaded, total)
			if downloaded >= total {
				i.logger.Info().Msg("下载完成！")
			}
		},
	}

	// 使用多写入器同时写入文件和更新进度
	writer := io.MultiWriter(tmpFile, progressWriter)

	// 保存到临时文件
	if _, err := io.Copy(writer, resp.Body); err != nil {
		return err
	}

	i.logger.Info().Msg("正在解压文件...")
	// 解压文件
	if strings.HasSuffix(url, ".zip") {
		return i.extractZip(tmpFile.Name(), version)
	} else if strings.HasSuffix(url, ".tar.gz") {
		return i.extractTarGz(tmpFile.Name(), version)
	}

	return fmt.Errorf("不支持的文件格式")
}

// extractZip 解压zip文件
func (i *Installer) extractZip(zipFile, version string) error {
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

		// 获取保存路径
		path := i.GetFRPBinaryPath(version)
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
func (i *Installer) extractTarGz(tarFile, version string) error {
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

		// 获取保存路径
		path := i.GetFRPBinaryPath(version)
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
