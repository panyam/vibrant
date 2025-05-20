package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func clientPasteCmd() *cobra.Command {
	out := &cobra.Command{
		Use:   "paste",
		Short: "Pastes data (e.g., image from a data URL or file) into a specified element.",
		Long: `Simulates a paste event on a target DOM element.
You can provide the content to paste either as a base64 data URL via --data
or by specifying an image file path via --file.`,
		Run: func(cmd *cobra.Command, args []string) {
			selector, _ := cmd.Flags().GetString("selector")
			dataURL, _ := cmd.Flags().GetString("data")
			filePath, _ := cmd.Flags().GetString("file")

			if selector == "" {
				log.Fatal("Error: --selector flag is required.")
			}
			if dataURL == "" && filePath == "" {
				log.Fatal("Error: either --data or --file flag must be provided.")
			}
			if dataURL != "" && filePath != "" {
				log.Fatal("Error: --data and --file flags are mutually exclusive.")
			}

			if rootCurrentClientId == "" {
				log.Fatal("Client ID not provided. Use -i flag or set VIBRANT_CLIENT_ID env var.")
			}

			var finalDataURL string
			if filePath != "" {
				fileBytes, err := os.ReadFile(filePath)
				if err != nil {
					log.Fatalf("Error reading file %s: %v", filePath, err)
				}
				mimeType := mime.TypeByExtension(filepath.Ext(filePath))
				if mimeType == "" {
					// Default to application/octet-stream or attempt to detect, but for images, common types are better.
					// For this tool's primary use case (images), let's assume common image types or require dataURL for others.
					// A more robust solution might use http.DetectContentType, but that needs the first 512 bytes.
					// For simplicity here, we'll encourage common image extensions or direct dataURL.
					log.Printf("Warning: could not determine MIME type for %s. Defaulting to image/png if it looks like one, else application/octet-stream.", filePath)
					if strings.HasSuffix(strings.ToLower(filePath), ".png") {
						mimeType = "image/png"
					} else if strings.HasSuffix(strings.ToLower(filePath), ".jpg") || strings.HasSuffix(strings.ToLower(filePath), ".jpeg") {
						mimeType = "image/jpeg"
					} else if strings.HasSuffix(strings.ToLower(filePath), ".gif") {
						mimeType = "image/gif"
					} else if strings.HasSuffix(strings.ToLower(filePath), ".webp") {
						mimeType = "image/webp"
					} else {
						mimeType = "application/octet-stream" // Fallback
					}
				}
				finalDataURL = fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(fileBytes))
			} else {
				finalDataURL = dataURL
				if !strings.HasPrefix(finalDataURL, "data:") {
					log.Fatal("Error: --data value must be a valid data URL (e.g., data:image/png;base64,...).")
				}
			}

			requestBody := map[string]string{
				"selector": selector,
				"dataUrl":  finalDataURL,
			}
			jsonBody, err := json.Marshal(requestBody)
			if err != nil {
				log.Fatalf("Error marshalling request body: %v", err)
			}

			endpointURL := fmt.Sprintf("http://localhost:9999/agents/%s/paste?wait=true", rootCurrentClientId)
			req, err := http.NewRequest("POST", endpointURL, bytes.NewBuffer(jsonBody))
			if err != nil {
				log.Fatalf("Error creating new HTTP request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			httpClient := &http.Client{Timeout: 40 * time.Second}
			resp, err := httpClient.Do(req)
			if err != nil {
				log.Fatalf("Error calling paste endpoint: %v", err)
			}
			defer resp.Body.Close()

			respBodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatalf("Error reading response body: %v", err)
			}

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("Server returned non-OK status %d: %s", resp.StatusCode, string(respBodyBytes))
			}

			var pasteResponse struct {
				RequestID string `json:"requestId"`
				Response  struct {
					Success bool   `json:"success"`
					Message string `json:"message,omitempty"`
					Error   string `json:"error,omitempty"`
				} `json:"response"` // Assuming the actual result from the JS is nested here
			}

			// Handle cases where the top-level response might be an error string itself
			var rawResponseOuter map[string]any
			if err := json.Unmarshal(respBodyBytes, &rawResponseOuter); err != nil {
				log.Fatalf("Error unmarshalling outer response JSON from /paste: %v. Raw response: %s", err, string(respBodyBytes))
			}

			responseField, ok := rawResponseOuter["response"]
			if !ok {
				// If "response" field is missing, it might be a direct error message from the server (e.g. timeout before response struct is formed)
				log.Fatalf("Paste command failed. Raw server response: %s", string(respBodyBytes))
			}

			// Re-marshal and unmarshal just the 'response' field content to handle it being an object or an error string
			responseBytes, err := json.Marshal(responseField)
			if err != nil {
				log.Fatalf("Error re-marshalling 'response' field: %v", err)
			}

			if err := json.Unmarshal(responseBytes, &pasteResponse.Response); err != nil {
				// If unmarshalling into the specific success/error struct fails, print the raw field
				log.Fatalf("Error unmarshalling paste command result: %v. Response field content: %s", err, string(responseBytes))
			}
			pasteResponse.RequestID, _ = rawResponseOuter["requestId"].(string)

			if pasteResponse.Response.Success {
				log.Printf("Paste command successful for selector '%s'. Message: %s (RequestID: %s)", selector, pasteResponse.Response.Message, pasteResponse.RequestID)
			} else {
				log.Printf("Paste command failed for selector '%s'. Error: %s (RequestID: %s)", selector, pasteResponse.Response.Error, pasteResponse.RequestID)
			}
		},
	}
	out.Flags().StringP("selector", "s", "", "CSS selector of the target element (required)")
	out.Flags().String("data", "", "Base64 data URL of the content to paste (e.g., data:image/png;base64,...)")
	out.Flags().StringP("file", "f", "", "Path to an image file to paste")
	return out
}

func init() {
	AddCommand(clientPasteCmd()) // Add to the main client command group
}
