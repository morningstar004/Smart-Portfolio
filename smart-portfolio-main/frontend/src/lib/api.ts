import type {
  APIResponse,
  Project,
  ContactRequest,
  ContactMessageResponse,
  ChatResponse,
} from "./types";
import { apiUrl } from "./public-api";

async function fetchAPI<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const res = await fetch(apiUrl(path), {
    headers: { "Content-Type": "application/json" },
    ...options,
  });

  const envelope: APIResponse<T> = await res.json();
  if (!envelope.success) {
    throw new Error(envelope.error?.message || "Unknown error");
  }
  return envelope.data as T;
}

export const getProjects = () => fetchAPI<Project[]>("/api/projects");

export const getProject = (id: string) =>
  fetchAPI<Project>(`/api/projects/${id}`);

export const submitContact = (data: ContactRequest) =>
  fetchAPI<ContactMessageResponse>("/api/contact", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const askChat = (question: string) =>
  fetchAPI<ChatResponse>("/api/chat", {
    method: "POST",
    body: JSON.stringify({ question }),
  });
