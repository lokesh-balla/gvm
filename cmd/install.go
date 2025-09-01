package cmd

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lokesh-balla/gvm/pkg/osc"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
)

// installCmd represents the install command.
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Specify the version of golang needed to install",
	Long: `Specify the version of golang which needs to be installed

	# To install a specific version
	$ gvm install go1.19
	
	# To install the latest stable version
	$ gvm install --latest
	`,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		latestFlag, _ := cmd.Flags().GetBool("latest")

		if latestFlag {
			if len(args) > 0 {
				return errors.New("cannot specify version when using --latest flag")
			}
			version, err := fetchLatestVersion()
			if err != nil {
				return fmt.Errorf("failed to fetch latest version: %w", err)
			}
			return installGolang(version)
		}

		if len(args) == 1 {
			return installGolang(args[0])
		}

		return errors.New("must specify version or use --latest flag")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().Bool("latest", false, "Install latest stable Go version")
}

func fetchLatestVersion() (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/VERSION?m=text", strings.TrimRight(GoDownloadServerURL, "/dl")))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.Split(strings.TrimSpace(string(body)), "\n")[0], nil
}

func drawProgressBar(output *termenv.Output, file string, width int, percent float64) {
	// setting OSC 9;4 for terminal progress if supported
	osc.SetOSCProgress(percent)

	if width < 0 || width > 120 {
		width = 120
	}
	drawString := fmt.Sprintf("%s ", termenv.String(file).Bold().Foreground(termenv.ANSIBlue))
	width = width - len(file) - 10
	barPercent := (percent / 100) * float64(width)
	for i := 0; i < width; i++ {
		if i <= int(barPercent) {
			drawString = fmt.Sprintf("%s%s", drawString, termenv.String(" ").Background(termenv.ANSIGreen))
		} else {
			drawString = fmt.Sprintf("%s%s", drawString, " ")
		}
	}
	drawString = fmt.Sprintf("%s %.0f%%", drawString, percent)
	_, _ = output.WriteString(drawString)
}

func progressBar(file string, size int, done chan bool) error {
	if size <= 0 {
		return fmt.Errorf("invalid size %v received", size)
	}

	output := termenv.NewOutput(os.Stdout)
	output.SaveCursorPosition()

	t := time.NewTicker(time.Millisecond * 50)
	defer t.Stop()
	for {
		select {
		case <-done:
			output.ClearLine()
			output.RestoreCursorPosition()
			drawProgressBar(output, file, TerminalWidth, 100)
			fmt.Println()
			return nil
		case <-t.C:
			fileStat, err := os.Stat(file)
			if err != nil {
				return err
			}
			percent := float64(fileStat.Size()) / float64(size) * 100
			output.ClearLineLeft()
			output.RestoreCursorPosition()
			drawProgressBar(output, file, TerminalWidth, percent)
		}
	}
}

func downloadGolang(filePath, url string) error {
	headResp, err := http.Head(url)
	if err != nil {
		return err
	}
	defer headResp.Body.Close()

	size, err := strconv.Atoi(headResp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}
	done := make(chan bool)
	go func() {
		if err := progressBar(filePath, size, done); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	done <- true

	return nil
}

func installGolang(version string) error {
	if !strings.HasPrefix(version, "go") {
		version = fmt.Sprintf("go%s", version)
	}

	pInfo, err := getGoDownloadPackageInfo(version)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%v/%v", GoDownloadServerURL, pInfo.Filename)

	filePath := fmt.Sprintf("%s/%s", GoInstallationDirectory, pInfo.Filename)

	if _, err := os.Stat(filePath); err != nil {
		if err := downloadGolang(filePath, url); err != nil {
			return err
		}
	}

	// check sha256 sum
	valid, err := checkSHA256Sum(filePath, pInfo.Sha256)
	if err != nil {
		return err
	}
	if !valid {
		fmt.Printf("%v sha256 failed\n", termenv.String("✗ ").Foreground(termenv.ANSIBrightRed))
		os.Remove(filePath)
		return fmt.Errorf("downloaded file has invalid sha256sum")
	}
	fmt.Printf("%v sha256 passed\n", termenv.String("✔ ").Foreground(termenv.ANSIBrightGreen))

	// extract Go
	fmt.Println("📦 un-taring go package")
	if err := extractGoTar(filePath, fmt.Sprintf("%s/%s", GoInstallationDirectory, version)); err != nil {
		fmt.Printf("%v un-taring failed\n", termenv.String("✗ ").Foreground(termenv.ANSIBrightRed))
		return err
	}
	fmt.Printf("%v un-taring successful\n", termenv.String("✔ ").Foreground(termenv.ANSIBrightGreen))

	// update metadate DB with installed version
	if err := DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(DBBucketName))
		if err != nil {
			return err
		}
		return bucket.Put([]byte(version), []byte{})
	}); err != nil {
		return err
	}

	fmt.Println("🧹 cleaning up downloaded tar")
	if err := os.Remove(filePath); err != nil {
		return err
	}

	fmt.Printf("🥳 %v successfully installed\n\n# To set %v as default:\n\t$ gvm use %v\n\n", version, version, version)
	return nil
}

func extractGoTar(filePath, targetPath string) error {
	madeDir := map[string]bool{}

	reader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("not a gzip compressed body: %w", err)
	}

	tarReader := tar.NewReader(gzipReader)

	for {
		f, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("tar error: %w", err)
		}
		rel := filepath.FromSlash(f.Name)
		abs := filepath.Join(targetPath, rel)

		fi := f.FileInfo()
		mode := fi.Mode()
		switch {
		case mode.IsRegular():
			dir := filepath.Dir(abs)
			if !madeDir[dir] {
				if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
					return err
				}
				madeDir[dir] = true
			}
			wf, err := os.OpenFile(abs, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode.Perm())
			if err != nil {
				return err
			}
			n, err := io.Copy(wf, tarReader)
			if closeErr := wf.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				return fmt.Errorf("error writing to %s: %w", abs, err)
			}
			if n != f.Size {
				return fmt.Errorf("only wrote %d bytes to %s; expected %d", n, abs, f.Size)
			}
		case mode.IsDir():
			if err := os.MkdirAll(abs, 0o755); err != nil {
				return err
			}
			madeDir[abs] = true
		default:
			return fmt.Errorf("tar file entry %s contained unsupported file type %v", f.Name, mode)
		}
	}

	return nil
}

func checkSHA256Sum(filePath, sha256hash string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, err
	}

	if sha256hash != hex.EncodeToString(hash.Sum(nil)) {
		return false, nil
	}

	return true, nil
}
