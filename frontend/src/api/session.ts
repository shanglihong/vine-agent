import { Session } from '../types';
import { request } from './client';

export async function fetchSessions(sessionId: string, userId: string, projectId: string): Promise<Session[]> {
  const data = await request<Session[]>(`/api/sessions?user_id=${userId}&session_id=${sessionId}&project_id=${projectId}`);
  return data;
}

export async function fetchSessionMessages(sessionId: string): Promise<{ messages: any[]; status?: string }> {
  const data = await request<{ messages: any[]; status?: string }>(`/api/sessions/${sessionId}/messages`);
  return data;
}

export async function createSession(sessionId: string, userId: string, projectId?: string): Promise<void> {
  const body: any = { session_id: sessionId, user_id: userId };
  if (projectId && projectId !== 'all' && projectId !== 'unclassified') {
    body.project_id = projectId;
  }
  await request<void>('/api/sessions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}

export async function sendChatMessage(sessionId: string, userId: string, message: string, model: string, tools?: string[]): Promise<Response> {
  const data = await request<Response>(`/api/sessions/${sessionId}/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, message, model, tools }),
  });
  return data;
}

export async function confirmInterrupt(sessionId: string, userId: string, confirmedToolCallIds: string[]): Promise<Response> {
  const data = await request<Response>(`/api/sessions/${sessionId}/confirm`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, confirmed_tool_call_ids: confirmedToolCallIds }),
  });
  return data;
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
  await request<void>(`/api/sessions/${sessionId}`, {
    method: 'DELETE',
  });
}

export async function renameSession(sessionId: string, name: string): Promise<void> {
  await request<void>(`/api/sessions/${sessionId}/rename`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
}

