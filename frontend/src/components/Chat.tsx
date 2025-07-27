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
    const connectWebSocket = () => {
      console.log('Attempting to connect to WebSocket...');
      ws.current = new WebSocket('ws://localhost:8080/ws/chat');
      
      ws.current.onopen = () => {
        console.log('Connected to chat server');
        setIsConnected(true);
      };
      
      ws.current.onclose = (event) => {
        console.log('Disconnected from chat server:', event.code, event.reason);
        setIsConnected(false);
        
        // Attempt to reconnect after 3 seconds
        setTimeout(() => {
          if (document.visibilityState === 'visible') {
            connectWebSocket();
          }
        }, 3000);
      };
      
      ws.current.onerror = (error) => {
        console.error('WebSocket error:', error);
        setIsConnected(false);
      };
      
      ws.current.onmessage = (event) => {
        console.log('Received message:', event.data);
        try {
          const data = JSON.parse(event.data);
          if (data.reply) {
            addMessage(data.reply, true);
            setIsLoading(false);
          } else if (data.error) {
            addMessage(`Error: ${data.error}`, true);
            setIsLoading(false);
          }
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
          addMessage('Sorry, I received an invalid response. Please try again.', true);
          setIsLoading(false);
        }
      };
    };
    
    connectWebSocket();
    
    // Reconnect when tab becomes visible again
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible' && (!ws.current || ws.current.readyState === WebSocket.CLOSED)) {
        connectWebSocket();
      }
    };
    
    document.addEventListener('visibilitychange', handleVisibilityChange);
    
    // Cleanup on unmount
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
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
    if (!input.trim() || isLoading) return;
    
    // Add user message
    const userMessage = input.trim();
    addMessage(userMessage, false);
    setIsLoading(true);
    setInput(''); // Clear input immediately for better UX
    
    // Send message via WebSocket if connected
    if (isConnected && ws.current?.readyState === WebSocket.OPEN) {
      console.log('Sending message via WebSocket:', userMessage);
      try {
        ws.current.send(JSON.stringify({ message: userMessage }));
      } catch (error) {
        console.error('Error sending WebSocket message:', error);
        // If WebSocket send fails, fall back to HTTP
        sendViaHttp(userMessage);
      }
    } else {
      console.log('WebSocket not connected, falling back to HTTP');
      // Fallback to HTTP if WebSocket is not available
      sendViaHttp(userMessage);
    }
  };
  
  const sendViaHttp = (message: string) => {
    console.log('Sending message via HTTP:', message);
    fetch('http://localhost:8080/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message: message }),
    })
      .then(response => {
        if (!response.ok) {
          throw new Error(`HTTP error! Status: ${response.status}`);
        }
        return response.json();
      })
      .then(data => {
        console.log('Received HTTP response:', data);
        if (data.reply) {
          addMessage(data.reply, true);
        } else {
          addMessage('Received empty response from server.', true);
        }
        setIsLoading(false);
      })
      .catch(error => {
        console.error('Error:', error);
        addMessage('Sorry, there was an error connecting to the server.', true);
        setIsLoading(false);
      });
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