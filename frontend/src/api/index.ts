import { Session, Profile, UserInfo } from '../types';

export async function fetchUserInfo(): Promise<UserInfo> {
  const res = await fetch('/api/user');
  if (!res.ok) {
    throw new Error(`Failed to fetch user info: status ${res.status}`);
  }
  return res.json();
}

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

export async function createSession(sessionId: string, userId: string): Promise<void> {
  const res = await fetch('/api/sessions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ session_id: sessionId, user_id: userId }),
  });
  if (!res.ok) {
    throw new Error(`Failed to create session: status ${res.status}`);
  }
}

export async function fetchUserProfile(userId: string): Promise<Profile> {
  const res = await fetch(`/api/users/${userId}/profile`);
  if (!res.ok) {
    throw new Error(`Failed to fetch profile: status ${res.status}`);
  }
  const data = await res.json();
  return {
    user_id: data.user_id,
    preferences: data.preferences || [],
    facts: data.facts || [],
  };
}

export async function evolveUserProfile(userId: string, sessionId: string): Promise<Profile> {
  const res = await fetch(`/api/users/${userId}/evolve?session_id=${sessionId}`, {
    method: 'POST',
  });
  if (!res.ok) {
    throw new Error(`Failed to evolve profile: status ${res.status}`);
  }
  const data = await res.json();
  return {
    user_id: data.user_id,
    preferences: data.preferences || [],
    facts: data.facts || [],
  };
}

export async function sendChatMessage(sessionId: string, userId: string, message: string, model: string): Promise<Response> {
  const res = await fetch(`/api/sessions/${sessionId}/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, message, model }),
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
