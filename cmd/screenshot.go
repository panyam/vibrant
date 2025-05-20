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
)

func clientScreenshotCmd() *cobra.Command {
	out := &cobra.Command{
		Use:   "screenshot",
		Short: "Captures screenshots of specified elements on the page.",
		Long: `Captures screenshots of one or more DOM elements identified by CSS selectors.
The captured images are saved to a specified output directory.`,
		Run: func(cmd *cobra.Command, args []string) {
			selectors, _ := cmd.Flags().GetStringArray("selector")
			outputDir, _ := cmd.Flags().GetString("output-dir")

			if len(selectors) == 0 {
				log.Fatal("At least one selector must be provided with -s or --selector.")
			}

			if rootCurrentClientId == "" {
				log.Fatal("Client ID not provided. Use -i flag or set VIBRANT_CLIENT_ID env var.")
			}

			if outputDir == "" {
				outputDir = "." // Default to current directory
			}

			// Create output directory if it doesn't exist
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

			endpointURL := fmt.Sprintf("http://localhost:9999/agents/%s/screenshots?wait=true", rootCurrentClientId)
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

			// It's possible the entire "response" field itself is an error string from the server if JSON parsing failed there
			// For now, we attempt to unmarshal into the expected structure.
			// If Response is actually an error string from the top level (e.g. from server.go if it couldn't marshal imageData)
			// then this unmarshal will fail. More robust error checking might be needed here.
			var rawResponseOuter map[string]interface{}
			if err := json.Unmarshal(respBodyBytes, &rawResponseOuter); err != nil {
				log.Fatalf("Error unmarshalling outer response JSON from /screenshot_elements: %v. Raw response: %s", err, string(respBodyBytes))
			}

			responseField, ok := rawResponseOuter["response"]
			if !ok {
				log.Fatalf("Missing 'response' field in /screenshot_elements JSON: %s", string(respBodyBytes))
			}

			// Now, marshal and unmarshal just the 'response' field content.
			// This handles cases where 'response' is a JSON object (imageData) or a string (error from server).
			responseBytes, err := json.Marshal(responseField)
			if err != nil {
				log.Fatalf("Error re-marshalling 'response' field: %v", err)
			}

			// Check if the inner response is an error structure like {"error": "message"}
			var errorResp struct {
				ErrorMsg string `json:"error"`
			}
			if json.Unmarshal(responseBytes, &errorResp) == nil && errorResp.ErrorMsg != "" {
				log.Fatalf("Received error from agent: %s", errorResp.ErrorMsg)
			}

			// Try to unmarshal into the expected imageData map
			imageDataMap := make(map[string]*string)
			if err := json.Unmarshal(responseBytes, &imageDataMap); err != nil {
				log.Fatalf("Error unmarshalling imageData from response field: %v. Response field content: %s", err, string(responseBytes))
			}

			screenshotResponse.Response = imageDataMap
			screenshotResponse.RequestID, _ = rawResponseOuter["requestId"].(string)

			log.Printf("Received screenshot data for RequestID: %s", screenshotResponse.RequestID)
			savedCount := 0
			for selector, dataURL := range screenshotResponse.Response {
				if dataURL == nil {
					log.Printf("No screenshot captured for selector '%s' (element not found or error).", selector)
					continue
				}

				// data:image/png;base64,iVBORw0KGgoAAAANS...
				parts := strings.SplitN(*dataURL, ",", 2)
				if len(parts) != 2 || !strings.HasPrefix(parts[0], "data:image/png;base64") {
					log.Printf("Invalid data URL format for selector '%s'. Skipping.", selector)
					continue
				}

				imgData, err := base64.StdEncoding.DecodeString(parts[1])
				if err != nil {
					log.Printf("Error decoding base64 image for selector '%s': %v. Skipping.", selector, err)
					continue
				}

				// Sanitize selector for filename
				reg := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
				safeSelector := reg.ReplaceAllString(selector, "_")
				if len(safeSelector) > 50 { // Truncate if too long
					safeSelector = safeSelector[:50]
				}
				if safeSelector == "" {
					safeSelector = "element"
				}

				// Add a timestamp to ensure uniqueness if multiple screenshots of same selector (though not typical for one call)
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
			if savedCount == 0 {
				log.Println("No screenshots were successfully saved.")
			}
		},
	}
	out.Flags().StringArrayP("selector", "s", []string{}, "CSS selector of the element to screenshot (can be repeated).")
	out.Flags().StringP("output-dir", "o", "./screenshots", "Directory to save the screenshots.")
	return out
}

func init() {
	AddCommand(clientScreenshotCmd())
}
