package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type Config struct {
	HostURL  string `yaml:"host_url"`
	AuthCode string `yaml:"auth_code"`
}

var config Config
var configDir string
var configFile string

var rootCmd = &cobra.Command{
	Use:   "surl",
	Short: "SURL is a command line interface for shortening long URLs",
	Long:  `SURL is a command line interface that allows you to shorten long URLs and manage them easily.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello, World!")
	},
}

var setCmd = &cobra.Command{
	Use:   "set <host> <url>",
	Short: "Set the host URL and authentication code",
	Long:  `Set the host URL and authentication code in the configuration file.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		hostUrl := args[0]
		authCode := args[1]
		err := saveConfig(Config{HostURL: hostUrl, AuthCode: authCode}, filepath.Join(configDir, "config.yml"))
		if err != nil {
			log.Fatalf("Error saving configuration: %v", err)
			return
		}
		log.Printf("Configuration saved.")
	},
}

var setHostUrlCmd = &cobra.Command{
	Use:   "set-host <host>",
	Short: "Set the host URL",
	Long:  `Set the host URL in the configuration file.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hostUrl := args[0]
		config.HostURL = hostUrl
		err := saveConfig(config, configFile)
		if err != nil {
			log.Fatalf("Error saving configuration: %v", err)
			return
		}
		fmt.Println("Host URL successfully saved.")
	},
}

var getHostUrlCmd = &cobra.Command{
	Use:   "get-host",
	Short: "Get the host URL",
	Long:  `Get the host URL from the configuration file.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Host URL:", config.HostURL)
	},
}

var setAuthCodeCmd = &cobra.Command{
	Use:   "set-code <code>",
	Short: "Set the authentication code",
	Long:  `Set the authentication code in the configuration file.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		authCode := args[0]
		config.AuthCode = authCode
		err := saveConfig(config, configFile)
		if err != nil {
			log.Fatalf("Error saving configuration: %v", err)
			return
		}
		fmt.Println("Authentication code successfully saved.")
	},
}

var getAuthCodeCmd = &cobra.Command{
	Use:   "get-code",
	Short: "Get the authentication code",
	Long:  `Get the authentication code from the configuration file.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Authentication code:", config.AuthCode)
	},
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all URLs",
	Long:    `This command lists all URLs that have been shortened.`,
	Run: func(cmd *cobra.Command, args []string) {
		if config.HostURL == "" {
			log.Fatalf("Host URL not set. Please set the host URL using the 'set-host' command.")
			return
		} else if config.AuthCode == "" {
			log.Fatalf("Auth code not set. Please set the auth code using the 'set-code' command.")
			return
		}

		listResponseBytes, err := fetch("GET", fmt.Sprintf("%s/list?code=%s", config.HostURL, config.AuthCode), nil, nil)
		if err != nil {
			log.Fatalf("Error fetching list of URLs: %v", err)
			return
		}

		var urlList struct {
			Data map[string]string `json:"data"`
		}
		err = json.Unmarshal(listResponseBytes, &urlList)
		if err != nil {
			log.Fatalf("Error parsing JSON: %v\n", err)
			return
		}

		for shortUrl, longUrl := range urlList.Data {
			fmt.Printf("%s => %s\n", shortUrl, longUrl)
		}
	},
}

var shortenCmd = &cobra.Command{
	Use:     "shorten",
	Aliases: []string{"create"},
	Short:   "Shorten a URL",
	Long:    `Shorten a long URL using the specified host URL and authentication code.`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if config.HostURL == "" {
			log.Fatalf("Host URL not set. Please set the host URL using the 'set-host' command.")
			return
		}

		log.Printf("Shortening URL: %s\n", args[0])

		requestData := url.Values{}
		requestData.Set("url", args[0])
		responseBytes, err := fetch("POST", fmt.Sprintf("%s/shorten", config.HostURL), strings.NewReader(requestData.Encode()), map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		})
		if err != nil {
			log.Fatalf("Error making request: %v", err)
			return
		}

		var shortUrlId struct {
			ShortUrl string `json:"shortUrl"`
		}
		err = json.Unmarshal(responseBytes, &shortUrlId)
		if err != nil {
			log.Fatalf("Error parsing JSON: %v\n", err)
			return
		}

		fmt.Printf("Shortened URL: %s\n", shortUrlId.ShortUrl)
	},
}

var openCmd = &cobra.Command{
	Use:     "open <id>",
	Aliases: []string{"o"},
	Short:   "Open a shortened URL",
	Long:    `Open a shortened URL in the default web browser.`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if config.HostURL == "" {
			log.Fatalf("Host URL not set. Please set the host URL using the 'set-host' command.")
			return
		}

		url := fmt.Sprintf("%s/r/%s", config.HostURL, args[0])

		var executeCmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			executeCmd = exec.Command("cmd", "/c", "start", url)
		case "darwin":
			executeCmd = exec.Command("open", url)
		default:
			executeCmd = exec.Command("xdg-open", url)
		}
		executeCmd.Run()
	},
}

var deleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"del"},
	Short:   "Delete a shortened URL",
	Long:    `Delete a shortened URL using the specified host URL and authentication code.`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if config.HostURL == "" {
			log.Fatalf("Host URL not set. Please set the host URL using the 'set-host' command.")
			return
		} else if config.AuthCode == "" {
			log.Fatalf("Auth code not set. Please set the auth code using the 'set-code' command.")
			return
		}

		shortUrl := args[0]
		deleteResponseBytes, err := fetch("DELETE", fmt.Sprintf("%s/d/%s?code=%s", config.HostURL, shortUrl, config.AuthCode), nil, nil)
		if err != nil {
			log.Fatalf("Error deleting URL: %v", err)
			return
		}

		var deleteResponse struct {
			ShortUrl string `json:"shortUrl"`
		}

		err = json.Unmarshal(deleteResponseBytes, &deleteResponse)
		if err != nil {
			log.Fatalf("Error parsing JSON: %v\n", err)
			return
		}

		fmt.Printf("Deleted URL: %s\n", deleteResponse.ShortUrl)
	},
}

func init() {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	configDir = filepath.Join(currentUser.HomeDir, ".config", rootCmd.Name())
	configFile = filepath.Join(configDir, "config.yml")

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		err = saveConfig(Config{}, configFile)
		if err != nil {
			log.Fatalf("Error saving empty configuration file: %v", err)
			return
		}
		log.Printf("Configuration file %s was created!", configFile)
	} else {
		file, err := os.Open(configFile)
		if err != nil {
			log.Fatalf("Error opening configuration file: %v", err)
			return
		}
		defer file.Close()

		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(&config); err != nil {
			log.Fatalf("Error reading configuration file: %v", err)
			return
		}
	}

	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(setHostUrlCmd)
	rootCmd.AddCommand(getHostUrlCmd)
	rootCmd.AddCommand(setAuthCodeCmd)
	rootCmd.AddCommand(getAuthCodeCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(shortenCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(deleteCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// saveConfig saves the provided configuration to a file.
// It creates the necessary directory structure if it doesn't exist.
// The configuration is encoded as YAML and written to the specified configFile.
// If any error occurs during the process, it is returned.
func saveConfig(config Config, configFile string) error {
	err := os.MkdirAll(filepath.Dir(configFile), os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating configuration directory: %v", err)
		return err
	}

	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	if err := encoder.Encode(config); err != nil {
		return err
	}

	return nil
}

// fetch makes an HTTP request with the specified method, URL, body, and headers,
// and returns the response body as a byte slice.
// If the request encounters an error or the response status code is not 200 OK,
// an error is returned.
func fetch(method string, url string, body io.Reader, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error making request: %s", resp.Status)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return responseBody, nil
}
