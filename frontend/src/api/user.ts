import { Profile, UserInfo } from '../types';
import { request } from './client';

export async function fetchUserInfo(): Promise<UserInfo> {
  const data = await request<UserInfo>('/api/user');
  return data;
}

export async function fetchUserProfile(userId: string): Promise<Profile> {
  const data = await request<Profile>(`/api/users/${userId}/profile`);
  return {
    user_id: data.user_id,
    preferences: data.preferences || [],
    facts: data.facts || [],
  };
}

export async function evolveUserProfile(userId: string, sessionId: string): Promise<Profile> {
  const data = await request<Profile>(`/api/users/${userId}/evolve?session_id=${sessionId}`, {
    method: 'POST',
  });
  return {
    user_id: data.user_id,
    preferences: data.preferences || [],
    facts: data.facts || [],
  };
}
