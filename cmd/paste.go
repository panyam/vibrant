package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png" // Added for encoding clipboard image to PNG
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.design/x/clipboard"
)

func clientPasteCmd() *cobra.Command {
	out := &cobra.Command{
		Use:   "paste",
		Short: "Pastes data (e.g., image from file, data URL, or clipboard) into a specified element.",
		Long: `Simulates a paste event on a target DOM element.
You can provide the content to paste as a base64 data URL via --data,
by specifying an image file path via --file, or from the system clipboard.
If image data is on the clipboard, it will be converted to a PNG data URL.
If text data is on the clipboard, it will be used directly if it's a data URL.`,
		Run: func(cmd *cobra.Command, args []string) {
			selector, _ := cmd.Flags().GetString("selector")
			dataURLFlag, _ := cmd.Flags().GetString("data")
			filePathFlag, _ := cmd.Flags().GetString("file")

			if selector == "" {
				log.Fatal("Error: --selector flag is required.")
			}

			if rootCurrentClientId == "" {
				log.Fatal("Client ID not provided. Use -i flag or set VIBRANT_CLIENT_ID env var.")
			}

			var finalDataURL string
			var dataSource string // To indicate where the data came from

			if filePathFlag != "" {
				if dataURLFlag != "" {
					log.Fatal("Error: --data and --file flags are mutually exclusive.")
				}
				dataSource = "file " + filePathFlag
				fileBytes, err := os.ReadFile(filePathFlag)
				if err != nil {
					log.Fatalf("Error reading file %s: %v", filePathFlag, err)
				}
				mimeType := mime.TypeByExtension(filepath.Ext(filePathFlag))
				if mimeType == "" {
					log.Printf("Warning: could not determine MIME type for %s. Attempting to infer.", filePathFlag)
					if strings.HasSuffix(strings.ToLower(filePathFlag), ".png") {
						mimeType = "image/png"
					} else if strings.HasSuffix(strings.ToLower(filePathFlag), ".jpg") || strings.HasSuffix(strings.ToLower(filePathFlag), ".jpeg") {
						mimeType = "image/jpeg"
					} else if strings.HasSuffix(strings.ToLower(filePathFlag), ".gif") {
						mimeType = "image/gif"
					} else if strings.HasSuffix(strings.ToLower(filePathFlag), ".webp") {
						mimeType = "image/webp"
					} else {
						mimeType = "application/octet-stream"
					}
				}
				finalDataURL = fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(fileBytes))
			} else if dataURLFlag != "" {
				dataSource = "data flag"
				finalDataURL = dataURLFlag
			} else if rootFromClipboard || (filePathFlag == "" && dataURLFlag == "") { // Use clipboard if explicitly requested or as default
				dataSource = "clipboard"
				// Try reading image data first
				imgBytes := clipboard.Read(clipboard.FmtImage)
				if len(imgBytes) > 0 {
					// Assume PNG for now. More sophisticated type detection might be needed for other formats.
					// For simplicity, we convert to PNG data URL directly.
					// Note: clipboard.Read(clipboard.FmtImage) often gives raw RGBA or similar, needs encoding.
					// The `golang.design/x/clipboard` library's FmtImage often gives raw pixel data.
					// For it to be useful as a data URL, it needs to be in a standard image format (like PNG).
					// Let's try to decode it as an image and re-encode as PNG.
					img, err := png.Decode(bytes.NewReader(imgBytes)) // image.Decode can identify format
					if err == nil {
						var pngBytes bytes.Buffer
						err = png.Encode(&pngBytes, img)
						if err == nil {
							finalDataURL = fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(pngBytes.Bytes()))
							log.Println("Successfully converted clipboard image to PNG data URL.")
						} else {
							log.Printf("Warning: Could not encode clipboard image to PNG: %v. Falling back to text.", err)
						}
					} else {
						log.Printf("Warning: Clipboard contained image data, but could not decode it: %v. Falling back to text clipboard.", err)
					}
				}

				// If image processing didn't yield a data URL, try text clipboard
				if finalDataURL == "" {
					clipboardContent := string(clipboard.Read(clipboard.FmtText))
					if clipboardContent == "" {
						msg := "Error: Clipboard is empty or content is not recognized as image or text data URL."
						if rootFromClipboard {
							msg = "Error: --from-clipboard was specified, but clipboard is empty or content is not recognized as image or text data URL."
						}
						log.Fatal(msg)
					}
					finalDataURL = clipboardContent
					log.Println("Using text content from clipboard.")
				}
			} else {
				log.Fatal("Error: No data source specified. Use --file, --data, or ensure clipboard content is available.")
			}

			if finalDataURL == "" {
				log.Fatalf("Error: Could not obtain data URL from any source (%s).", dataSource)
			}

			if !strings.HasPrefix(finalDataURL, "data:") {
				log.Fatalf("Error: Input from %s must be a valid data URL (e.g., data:image/png;base64,...). Received: %.100s...", dataSource, finalDataURL)
			}

			requestBody := map[string]string{
				"selector": selector,
				"dataUrl":  finalDataURL,
			}
			jsonBody, err := json.Marshal(requestBody)
			if err != nil {
				log.Fatalf("Error marshalling request body: %v", err)
			}

			endpointURL := fmt.Sprintf("http://%s/agents/%s/paste?wait=true", rootVibrantHost, rootCurrentClientId)
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
				} `json:"response"`
			}

			var rawResponseOuter map[string]any
			if err := json.Unmarshal(respBodyBytes, &rawResponseOuter); err != nil {
				log.Fatalf("Error unmarshalling outer response JSON from /paste: %v. Raw response: %s", err, string(respBodyBytes))
			}

			responseField, ok := rawResponseOuter["response"]
			if !ok {
				log.Fatalf("Paste command failed. Raw server response: %s", string(respBodyBytes))
			}

			responseBytes, err := json.Marshal(responseField)
			if err != nil {
				log.Fatalf("Error re-marshalling 'response' field: %v", err)
			}

			if err := json.Unmarshal(responseBytes, &pasteResponse.Response); err != nil {
				log.Fatalf("Error unmarshalling paste command result: %v. Response field content: %s", err, string(responseBytes))
			}
			pasteResponse.RequestID, _ = rawResponseOuter["requestId"].(string)

			if pasteResponse.Response.Success {
				log.Printf("Paste command successful for selector '%s' from %s. Message: %s (RequestID: %s)", selector, dataSource, pasteResponse.Response.Message, pasteResponse.RequestID)
			} else {
				log.Printf("Paste command failed for selector '%s' (source: %s). Error: %s (RequestID: %s)", selector, dataSource, pasteResponse.Response.Error, pasteResponse.RequestID)
			}
		},
	}
	out.Flags().StringP("selector", "s", "ms-prompt-input-wrapper textarea", "CSS selector of the target element (required)")
	out.Flags().String("data", "", "Base64 data URL of the content to paste (e.g., data:image/png;base64,...)")
	out.Flags().StringP("file", "f", "", "Path to an image file to paste")
	return out
}

func init() {
	AddCommand(clientPasteCmd())
	// This command is added to rootCmd via AddCommand(clientPasteCmd()) in its own init in the provided file structure.
	// If it were to be a subcommand of `client`, it would be `clientCmd.AddCommand(clientPasteCmd())` in `client.go`.
}
