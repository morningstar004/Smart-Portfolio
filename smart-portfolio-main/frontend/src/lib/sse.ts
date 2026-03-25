import { apiUrl } from "./public-api";

export async function streamChat(
  question: string,
  onToken: (token: string) => void,
  onDone: () => void,
  onError: (error: string) => void,
): Promise<void> {
  try {
    const response = await fetch(apiUrl("/api/chat/stream"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ question }),
    });

    if (!response.ok || !response.body) {
      onError("Failed to start chat stream");
      return;
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (line.startsWith("event: done")) {
          onDone();
          return;
        }
        if (line.startsWith("data: ")) {
          onToken(line.slice(6));
        }
      }
    }
    onDone();
  } catch {
    onError("Connection error. Please try again.");
  }
}
