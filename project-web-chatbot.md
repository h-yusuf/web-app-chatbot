# Web Chatbot Application Blueprint

## Project Overview

This blueprint outlines the development of a web-based chatbot application with real-time chat functionality that connects to n8n for workflow automation. The application consists of a Go backend using Fiber framework and an Astro frontend.

## Architecture

```
+----------------+     +----------------+     +----------------+
|                |     |                |     |                |
|  Astro         |     |  Go/Fiber      |     |  n8n           |
|  Frontend      +---->+  Backend       +---->+  Workflow      |
|                |     |                |     |                |
+----------------+     +----------------+     +----------------+
```

## Tech Stack

### Backend
- Go 1.24.5
- Fiber v2 (Web framework)
- WebSocket for real-time communication

### Frontend
- Astro 5.x
- TypeScript
- Tailwind CSS (for styling)

### External Services
- n8n for workflow automation

## Implementation Plan

### Phase 1: Backend Development

1. Set up the Go Fiber server with basic endpoints
2. Implement WebSocket support for real-time chat
3. Create n8n integration with proper error handling
4. Add CORS configuration for frontend communication

### Phase 2: Frontend Development

1. Design and implement the chat UI
2. Add WebSocket client for real-time communication
3. Implement message history and user interface
4. Add loading states and error handling

### Phase 3: Integration and Testing

1. Connect frontend to backend
2. Test real-time communication
3. Verify n8n workflow integration
4. Performance optimization

## Detailed Implementation

### Backend (Go/Fiber)

#### Server Setup

```go
package main

import (
	"log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"bytes"
	"encoding/json"
	"net/http"
)

func main() {
	app := fiber.New()

	// Enable CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:4321", // Astro default port
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// Regular HTTP endpoint for chat
	app.Post("/chat", func(c *fiber.Ctx) error {
		var body map[string]string
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
		}

		// Forward message to n8n webhook
		webhookURL := "https://YOUR_N8N_URL/webhook/kawaibot"
		payload, _ := json.Marshal(map[string]string{"message": body["message"]})

		resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to contact bot"})
		}
		defer resp.Body.Close()

		var n8nResp map[string]string
		json.NewDecoder(resp.Body).Decode(&n8nResp)

		return c.JSON(fiber.Map{"reply": n8nResp["reply"]})
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

		// Forward message to n8n webhook
		webhookURL := "https://YOUR_N8N_URL/webhook/kawaibot"
		payload, _ := json.Marshal(map[string]string{"message": msg.Message})

		resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			c.WriteJSON(fiber.Map{"error": "Failed to contact bot"})
			continue
		}

		var n8nResp map[string]string
		json.NewDecoder(resp.Body).Decode(&n8nResp)
		resp.Body.Close()

		// Send response back to client
		if err := c.WriteJSON(fiber.Map{"reply": n8nResp["reply"]}); err != nil {
			log.Println("write error:", err)
			break
		}
	}
}
```

### Frontend (Astro)

#### Chat Component

Create a new file at `frontend/src/components/Chat.tsx`:

```typescript
import { useState, useEffect, useRef } from 'react';

interface Message {
  text: string;
  isBot: boolean;
  timestamp: Date;
}

export default function Chat() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isConnected, setIsConnected] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const ws = useRef<WebSocket | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Connect to WebSocket
  useEffect(() => {
    // Initialize WebSocket connection
    ws.current = new WebSocket('ws://localhost:8080/ws/chat');
    
    ws.current.onopen = () => {
      console.log('Connected to chat server');
      setIsConnected(true);
    };
    
    ws.current.onclose = () => {
      console.log('Disconnected from chat server');
      setIsConnected(false);
    };
    
    ws.current.onerror = (error) => {
      console.error('WebSocket error:', error);
      setIsConnected(false);
    };
    
    ws.current.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.reply) {
        addMessage(data.reply, true);
        setIsLoading(false);
      }
    };
    
    // Cleanup on unmount
    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
  }, []);

  // Scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const addMessage = (text: string, isBot: boolean) => {
    setMessages(prev => [...prev, { text, isBot, timestamp: new Date() }]);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || !isConnected || isLoading) return;
    
    // Add user message
    addMessage(input, false);
    setIsLoading(true);
    
    // Send message via WebSocket
    if (ws.current?.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify({ message: input }));
      setInput('');
    } else {
      // Fallback to HTTP if WebSocket is not available
      fetch('http://localhost:8080/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: input }),
      })
        .then(response => response.json())
        .then(data => {
          if (data.reply) {
            addMessage(data.reply, true);
          }
          setIsLoading(false);
        })
        .catch(error => {
          console.error('Error:', error);
          addMessage('Sorry, there was an error connecting to the server.', true);
          setIsLoading(false);
        });
      
      setInput('');
    }
  };

  return (
    <div className="flex flex-col h-[80vh] max-w-2xl mx-auto border rounded-lg overflow-hidden bg-white shadow-lg">
      <div className="p-4 bg-blue-600 text-white font-bold">
        Chatbot
        <span className={`ml-2 inline-block w-3 h-3 rounded-full ${isConnected ? 'bg-green-400' : 'bg-red-500'}`}></span>
      </div>
      
      <div className="flex-1 p-4 overflow-y-auto bg-gray-50">
        {messages.length === 0 ? (
          <div className="text-center text-gray-500 mt-10">
            Send a message to start chatting!
          </div>
        ) : (
          messages.map((msg, index) => (
            <div key={index} className={`mb-4 ${msg.isBot ? 'text-left' : 'text-right'}`}>
              <div 
                className={`inline-block p-3 rounded-lg ${msg.isBot 
                  ? 'bg-gray-200 text-gray-800' 
                  : 'bg-blue-500 text-white'}`}
              >
                {msg.text}
              </div>
              <div className="text-xs text-gray-500 mt-1">
                {msg.timestamp.toLocaleTimeString()}
              </div>
            </div>
          ))
        )}
        {isLoading && (
          <div className="text-left mb-4">
            <div className="inline-block p-3 rounded-lg bg-gray-200">
              <div className="flex space-x-2">
                <div className="w-2 h-2 rounded-full bg-gray-400 animate-bounce"></div>
                <div className="w-2 h-2 rounded-full bg-gray-400 animate-bounce" style={{ animationDelay: '0.2s' }}></div>
                <div className="w-2 h-2 rounded-full bg-gray-400 animate-bounce" style={{ animationDelay: '0.4s' }}></div>
              </div>
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>
      
      <form onSubmit={handleSubmit} className="p-4 border-t border-gray-200 bg-white">
        <div className="flex space-x-2">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Type your message..."
            className="flex-1 p-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            disabled={!isConnected || isLoading}
          />
          <button 
            type="submit" 
            className="bg-blue-500 text-white px-4 py-2 rounded-lg hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
            disabled={!isConnected || !input.trim() || isLoading}
          >
            Send
          </button>
        </div>
      </form>
    </div>
  );
}
```

#### Update Index Page

Update `frontend/src/pages/index.astro`:

```astro
---
import Layout from '../layouts/Layout.astro';
import Chat from '../components/Chat';
---

<Layout title="Chatbot Application">
  <main class="container mx-auto p-4">
    <h1 class="text-3xl font-bold text-center my-8">Web Chatbot</h1>
    <Chat client:load />
  </main>
</Layout>
```

#### Create Layout Component

Create `frontend/src/layouts/Layout.astro`:

```astro
---
interface Props {
  title: string;
}

const { title } = Astro.props;
---

<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width" />
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <meta name="generator" content={Astro.generator} />
    <title>{title}</title>
  </head>
  <body class="bg-gray-100 min-h-screen">
    <slot />
  </body>
</html>
```

## Required Dependencies

### Backend

Add the following dependencies to your Go project:

```bash
go get github.com/gofiber/fiber/v2
go get github.com/gofiber/websocket/v2
go get github.com/gofiber/fiber/v2/middleware/cors
```

### Frontend

Update `package.json` with the following dependencies:

```json
{
  "name": "web-chatbot-frontend",
  "type": "module",
  "version": "0.1.0",
  "scripts": {
    "dev": "astro dev",
    "build": "astro build",
    "preview": "astro preview",
    "astro": "astro"
  },
  "dependencies": {
    "@astrojs/react": "^3.0.0",
    "@astrojs/tailwind": "^5.0.0",
    "@types/react": "^18.2.21",
    "@types/react-dom": "^18.2.7",
    "astro": "^5.12.3",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "tailwindcss": "^3.3.3"
  }
}
```

Update Astro configuration in `astro.config.mjs`:

```javascript
import { defineConfig } from 'astro/config';
import tailwind from '@astrojs/tailwind';
import react from '@astrojs/react';

export default defineConfig({
  integrations: [tailwind(), react()],
});
```

## n8n Integration

1. Set up an n8n instance (self-hosted or cloud)
2. Create a webhook node as the trigger
3. Process the incoming message with n8n workflows
4. Return a response in the format: `{ "reply": "Bot response here" }`

## Deployment Considerations

1. **Backend**: Deploy the Go application to a server with proper environment variables for the n8n webhook URL
2. **Frontend**: Build and deploy the Astro application to a static hosting service
3. **WebSockets**: Ensure your hosting environment supports WebSocket connections
4. **Security**: Add proper authentication and rate limiting for production use

## Getting Started

1. Clone the repository
2. Install backend dependencies: `cd backend && go mod tidy`
3. Install frontend dependencies: `cd frontend && npm install`
4. Update the n8n webhook URL in the backend code
5. Start the backend: `cd backend && go run main.go`
6. Start the frontend: `cd frontend && npm run dev`
7. Access the application at `http://localhost:4321`

## Next Steps and Enhancements

1. Add user authentication
2. Implement message persistence with a database
3. Add typing indicators and read receipts
4. Support for file uploads and rich media
5. Add analytics and conversation tracking
6. Implement chatbot training interface