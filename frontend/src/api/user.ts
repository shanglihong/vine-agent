import { Profile, UserInfo } from '../types';

export async function fetchUserInfo(): Promise<UserInfo> {
  const res = await fetch('/api/user');
  if (!res.ok) {
    throw new Error(`Failed to fetch user info: status ${res.status}`);
  }
  return res.json();
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
