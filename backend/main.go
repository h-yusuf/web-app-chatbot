package main

import (
	"log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// WebSocket clients manager
type Client struct {
	Conn *websocket.Conn
}

var clients = make(map[*websocket.Conn]bool)

func handleWebSocket(c *websocket.Conn) {
	// Register new client
	clients[c] = true

	// Cleanup when the connection closes
	defer func() {
		delete(clients, c)
		c.Close()
	}()

	for {
		// Read message from client
		type Message struct {
			Message string `json:"message"`
		}
		var msg Message
		if err := c.ReadJSON(&msg); err != nil {
			log.Println("read error:", err)
			break
		}

		log.Printf("Received message: %s", msg.Message)

		// Forward message to n8n webhook
		webhookURL := "https://n8n.tspbrand.id/webhook/web-chatbot"
		payload, _ := json.Marshal(map[string]string{"message": msg.Message})

		resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			log.Printf("Error contacting webhook: %v", err)
			c.WriteJSON(fiber.Map{"reply": "Sorry, I couldn't process your message. Please try again later."})
			continue
		}

		// First try to read as plain text
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("Error reading response body: %v", err)
			c.WriteJSON(fiber.Map{"reply": "Sorry, I couldn't read the response from the server."})
			continue
		}

		log.Printf("Raw response body: %s", string(bodyBytes))

		// Determine response type and extract reply
		var reply string
		
		// Check if the response starts with common text response patterns
		responseText := string(bodyBytes)
		if strings.HasPrefix(responseText, "H") || strings.HasPrefix(responseText, "S") {
			// Likely a plain text response in Indonesian (Halo, Selamat, etc.)
			log.Printf("Detected plain text response starting with H/S, treating as plain text")
			reply = responseText
		} else if strings.TrimSpace(responseText) == "" {
			// Empty response
			log.Printf("Empty response received")
			reply = "No response received from the server."
		} else {
			// Try to parse as JSON
			var n8nResp map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &n8nResp); err == nil {
				// Successfully parsed as JSON
				log.Printf("Parsed JSON response: %v", n8nResp)
				
				// Check for error response
				if code, ok := n8nResp["code"]; ok {
					if code == float64(404) {
						if msg, ok := n8nResp["message"].(string); ok {
							reply = fmt.Sprintf("Error: %s", msg)
						} else {
							reply = "Error: Webhook not found or not registered."
						}
					}
				} else if replyVal, ok := n8nResp["reply"]; ok {
					// Extract reply from JSON
					switch v := replyVal.(type) {
					case string:
						reply = v
					case float64, int, int64, float32: // Handle numeric types
						reply = fmt.Sprintf("%v", v)
					default:
						reply = fmt.Sprintf("%v", v)
					}
				} else {
					// If no "reply" field, check if this is an error message
					reply = responseText
				}
			} else {
				// Not valid JSON, treat as plain text
				log.Printf("Response is not JSON, treating as plain text: %v", err)
				reply = responseText
			}
		}

		log.Printf("Sending reply: %s", reply)

		// Send response back to client
		if err := c.WriteJSON(fiber.Map{"reply": reply}); err != nil {
			log.Println("write error:", err)
			break
		}
	}
}

func main() {
	app := fiber.New()

	// Enable CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:4321", // Astro default port
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	app.Post("/chat", func(c *fiber.Ctx) error {
		var body map[string]string
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
		}

		log.Printf("Received HTTP message: %s", body["message"])

		// Forward message to webhook n8n
		webhookURL := "https://n8n.tspbrand.id/webhook/web-chatbot"
		payload, _ := json.Marshal(map[string]string{"message": body["message"]})

		resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			log.Printf("Error contacting webhook: %v", err)
			return c.Status(500).JSON(fiber.Map{"reply": "Sorry, I couldn't process your message. Please try again later."})
		}
		defer resp.Body.Close()

		// First try to read as plain text
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("Error reading response body: %v", err)
			return c.Status(500).JSON(fiber.Map{"reply": "Sorry, I couldn't read the response from the server."})
		}

		log.Printf("Raw HTTP response body: %s", string(bodyBytes))

		// Determine response type and extract reply
		var reply string
		
		// Check if the response starts with common text response patterns
		responseText := string(bodyBytes)
		if strings.HasPrefix(responseText, "H") || strings.HasPrefix(responseText, "S") {
			// Likely a plain text response in Indonesian (Halo, Selamat, etc.)
			log.Printf("Detected plain text response starting with H/S, treating as plain text")
			reply = responseText
		} else if strings.TrimSpace(responseText) == "" {
			// Empty response
			log.Printf("Empty response received")
			reply = "No response received from the server."
		} else {
			// Try to parse as JSON
			var n8nResp map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &n8nResp); err == nil {
				// Successfully parsed as JSON
				log.Printf("Parsed HTTP JSON response: %v", n8nResp)
				
				// Check for error response
				if code, ok := n8nResp["code"]; ok {
					if code == float64(404) {
						if msg, ok := n8nResp["message"].(string); ok {
							reply = fmt.Sprintf("Error: %s", msg)
						} else {
							reply = "Error: Webhook not found or not registered."
						}
					}
				} else if replyVal, ok := n8nResp["reply"]; ok {
					// Extract reply from JSON
					switch v := replyVal.(type) {
					case string:
						reply = v
					case float64, int, int64, float32: // Handle numeric types
						reply = fmt.Sprintf("%v", v)
					default:
						reply = fmt.Sprintf("%v", v)
					}
				} else {
					// If no "reply" field, check if this is an error message
					reply = responseText
				}
			} else {
				// Not valid JSON, treat as plain text
				log.Printf("HTTP response is not JSON, treating as plain text: %v", err)
				reply = responseText
			}
		}

		log.Printf("Sending HTTP reply: %s", reply)

		return c.JSON(fiber.Map{"reply": reply})
	})

		// WebSocket setup
	app.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client requested upgrade to the WebSocket protocol
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws/chat", websocket.New(handleWebSocket))

	log.Fatal(app.Listen(":8080"))
}
