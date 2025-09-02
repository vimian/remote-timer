"use client";
import { useEffect, useRef, useState } from "react";

type Session = {
  id: string;
};

export default function Timer() {
  const [status, setStatus] = useState("Connecting...");
  const [session, setSession] = useState<Session | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    const ws = new window.WebSocket(
      "ws://127.0.0.1/ws"
    ); /* TODO: Replace with actual WebSocket URL */
    wsRef.current = ws;

    ws.onopen = () => {
      setStatus("Connected");
      ws.send(JSON.stringify({ event: "create_session" }));
    };
    ws.onerror = () => setStatus("Error");
    ws.onclose = () => setStatus("Closed");
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      switch (data.event) {
        case "session_created":
          setSession({ id: data.id });
          break;
        default:
          console.warn("Unknown event:", data);
      }
    };

    return () => {
      ws.close();
    };
  }, []);

  return (
    <div>
      <p>WebSocket status: {status}</p>
      {session && (
        <div>
          <p>Session ID: {session.id}</p>
        </div>
      )}
    </div>
  );
}
