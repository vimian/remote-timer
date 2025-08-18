import logo from './logo.svg';
import './App.css';
import useWebSocket from 'react-use-websocket';
import { useState } from 'react';

function App() {
  // Replace with your WebSocket server URL
  const socketUrl = 'ws://127.0.0.1/ws';
  const [messageHistory, setMessageHistory] = useState([]);
  const { sendMessage, lastMessage, readyState } = useWebSocket(socketUrl, {
    onOpen: () => console.log('WebSocket Connected'),
    onMessage: (message) => {
      setMessageHistory((prev) => [...prev, message.data]);
    },
    onClose: () => console.log('WebSocket Disconnected'),
    onError: (event) => console.error('WebSocket Error:', event),
    shouldReconnect: () => true, // Auto-reconnect on close
  });

  // Example: send a message when button is clicked
  const handleClick = () => {
    sendMessage(JSON.stringify({event: "create_session"}));
  };


  return (
    <div className="App">
      <header className="App-header">
        <img src={logo} className="App-logo" alt="logo" />
        <p>
          Edit <code>src/App.js</code> and save to reload.
        </p>
        <a
          className="App-link"
          href="https://reactjs.org"
          target="_blank"
          rel="noopener noreferrer"
        >
          Learn React
        </a>
        <button onClick={handleClick}>Send Message</button>
      </header>
    </div>
  );
}

export default App;
