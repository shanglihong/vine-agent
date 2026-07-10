import { Session } from '../types';

export async function fetchSessions(userId: string): Promise<Session[]> {
  const res = await fetch(`/api/sessions?user_id=${userId}`);
  if (!res.ok) {
    throw new Error(`Failed to fetch sessions: status ${res.status}`);
  }
  return res.json();
}

export async function fetchSessionMessages(sessionId: string): Promise<{ messages: any[]; status?: string }> {
  const res = await fetch(`/api/sessions/${sessionId}/messages`);
  if (!res.ok) {
    throw new Error(`Failed to fetch session messages: status ${res.status}`);
  }
  return res.json();
}

export async function createSession(sessionId: string, userId: string, projectId?: string): Promise<void> {
  const body: any = { session_id: sessionId, user_id: userId };
  if (projectId && projectId !== 'all' && projectId !== 'unclassified') {
    body.project_id = projectId;
  }
  const res = await fetch('/api/sessions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    throw new Error(`Failed to create session: status ${res.status}`);
  }
}

export async function sendChatMessage(sessionId: string, userId: string, message: string, model: string, tools?: string[]): Promise<Response> {
  const res = await fetch(`/api/sessions/${sessionId}/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, message, model, tools }),
  });
  if (!res.ok) {
    throw new Error(`Failed to send chat message: status ${res.status}`);
  }
  return res;
}

export async function confirmInterrupt(sessionId: string, userId: string, confirmedToolCallIds: string[]): Promise<Response> {
  const res = await fetch(`/api/sessions/${sessionId}/confirm`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, confirmed_tool_call_ids: confirmedToolCallIds }),
  });
  if (!res.ok) {
    throw new Error(`Failed to confirm interrupt: status ${res.status}`);
  }
  return res;
}

export async function cancelSessionChat(sessionId: string): Promise<void> {
  const res = await fetch(`/api/sessions/${sessionId}/cancel`, {
    method: 'POST',
  });
  if (!res.ok) {
    throw new Error(`Failed to cancel session chat: status ${res.status}`);
  }
}

export async function deleteSession(sessionId: string): Promise<void> {
  const res = await fetch(`/api/sessions/${sessionId}`, {
    method: 'DELETE',
  });
  if (!res.ok) {
    throw new Error(`Failed to delete session: status ${res.status}`);
  }
}

export async function renameSession(sessionId: string, name: string): Promise<void> {
  const res = await fetch(`/api/sessions/${sessionId}/rename`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
  if (!res.ok) {
    throw new Error(`Failed to rename session: status ${res.status}`);
  }
}

