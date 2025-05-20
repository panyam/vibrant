package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.design/x/clipboard"
)

func clientScreenshotCmd() *cobra.Command {
	out := &cobra.Command{
		Use:   "screenshot",
		Short: "Captures screenshots of specified elements on the page.",
		Long: `Captures screenshots of one or more DOM elements identified by CSS selectors.
The captured images are saved to a specified output directory.
Optionally, the first successful screenshot can be copied to the clipboard as a data URL.`,
		Run: func(cmd *cobra.Command, args []string) {
			selectors, _ := cmd.Flags().GetStringArray("selector")
			outputDir, _ := cmd.Flags().GetString("output-dir")
			toClipboard, _ := cmd.Flags().GetBool("to-clipboard")

			if len(selectors) == 0 {
				log.Fatal("At least one selector must be provided with -s or --selector.")
			}

			if rootCurrentClientId == "" {
				log.Fatal("Client ID not provided. Use -i flag or set VIBRANT_CLIENT_ID env var.")
			}

			if outputDir == "" {
				outputDir = "." // Default to current directory
			}

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				log.Fatalf("Failed to create output directory %s: %v", outputDir, err)
			}

			requestBody := map[string][]string{
				"selectors": selectors,
			}
			jsonBody, err := json.Marshal(requestBody)
			if err != nil {
				log.Fatalf("Error marshalling request body: %v", err)
			}

			endpointURL := fmt.Sprintf("http://%s/agents/%s/screenshots?wait=true", rootVibrantHost, rootCurrentClientId)
			req, err := http.NewRequest("POST", endpointURL, bytes.NewBuffer(jsonBody))
			if err != nil {
				log.Fatalf("Error creating new HTTP request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			httpClient := &http.Client{Timeout: 70 * time.Second} // Slightly longer timeout for screenshots
			resp, err := httpClient.Do(req)
			if err != nil {
				log.Fatalf("Error calling screenshot_elements endpoint: %v", err)
			}
			defer resp.Body.Close()

			respBodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatalf("Error reading response body: %v", err)
			}

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("Server returned non-OK status %d: %s", resp.StatusCode, string(respBodyBytes))
			}

			var screenshotResponse struct {
				RequestID string             `json:"requestId"`
				Response  map[string]*string `json:"response"` // Value can be string (dataURL) or null
			}

			var rawResponseOuter map[string]any
			if err := json.Unmarshal(respBodyBytes, &rawResponseOuter); err != nil {
				log.Fatalf("Error unmarshalling outer response JSON from /screenshot_elements: %v. Raw response: %s", err, string(respBodyBytes))
			}

			responseField, ok := rawResponseOuter["response"]
			if !ok {
				log.Fatalf("Missing 'response' field in /screenshot_elements JSON: %s", string(respBodyBytes))
			}

			responseBytes, err := json.Marshal(responseField)
			if err != nil {
				log.Fatalf("Error re-marshalling 'response' field: %v", err)
			}

			var errorResp struct {
				ErrorMsg string `json:"error"`
			}
			if json.Unmarshal(responseBytes, &errorResp) == nil && errorResp.ErrorMsg != "" {
				log.Fatalf("Received error from agent: %s", errorResp.ErrorMsg)
			}

			imageDataMap := make(map[string]*string)
			if err := json.Unmarshal(responseBytes, &imageDataMap); err != nil {
				log.Fatalf("Error unmarshalling imageData from response field: %v. Response field content: %s", err, string(responseBytes))
			}

			screenshotResponse.Response = imageDataMap
			screenshotResponse.RequestID, _ = rawResponseOuter["requestId"].(string)

			log.Printf("Received screenshot data for RequestID: %s", screenshotResponse.RequestID)
			savedCount := 0
			firstSuccessfulDataURL := ""

			// Iterate through original selectors to maintain order for `firstSuccessfulDataURL`
			for _, selector := range selectors {
				dataURL, dataOk := screenshotResponse.Response[selector]
				if !dataOk || dataURL == nil {
					log.Printf("No screenshot captured for selector '%s' (element not found or error).", selector)
					continue
				}

				// Store the first successful data URL if needed for clipboard
				if toClipboard && firstSuccessfulDataURL == "" {
					firstSuccessfulDataURL = *dataURL
				}

				parts := strings.SplitN(*dataURL, ",", 2)
				if len(parts) != 2 || !strings.HasPrefix(parts[0], "data:image/png;base64") {
					log.Printf("Invalid data URL format for selector '%s'. Skipping saving.", selector)
					continue
				}

				imgData, err := base64.StdEncoding.DecodeString(parts[1])
				if err != nil {
					log.Printf("Error decoding base64 image for selector '%s': %v. Skipping saving.", selector, err)
					continue
				}

				reg := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
				safeSelector := reg.ReplaceAllString(selector, "_")
				if len(safeSelector) > 50 {
					safeSelector = safeSelector[:50]
				}
				if safeSelector == "" {
					safeSelector = "element"
				}

				timestamp := time.Now().Format("20060102150405")
				filename := fmt.Sprintf("%s_%s.png", safeSelector, timestamp)
				filePath := filepath.Join(outputDir, filename)

				err = os.WriteFile(filePath, imgData, 0644)
				if err != nil {
					log.Printf("Error writing image file %s for selector '%s': %v", filePath, selector, err)
				} else {
					log.Printf("Saved screenshot for selector '%s' to %s", selector, filePath)
					savedCount++
				}
			}

			if savedCount == 0 && !toClipboard {
				log.Println("No screenshots were successfully saved.")
			} else if toClipboard {
				if firstSuccessfulDataURL != "" {
					// Initialize clipboard (safe to call multiple times)
					err := clipboard.Init()
					if err != nil {
						log.Printf("Warning: Failed to initialize clipboard: %v. Cannot copy screenshot.", err)
					} else {
						clipboard.Write(clipboard.FmtText, []byte(firstSuccessfulDataURL))
						log.Println("First successful screenshot data URL copied to clipboard.")
					}
				} else {
					log.Println("No screenshots were successful, nothing to copy to clipboard.")
				}
			}
		},
	}
	out.Flags().StringArrayP("selector", "s", []string{}, "CSS selector of the element to screenshot (can be repeated).")
	out.Flags().StringP("output-dir", "o", "./screenshots", "Directory to save the screenshots.")
	out.Flags().Bool("to-clipboard", false, "Copy the first successful screenshot (as data URL) to the clipboard.")
	return out
}

func init() {
	AddCommand(clientScreenshotCmd())
}
