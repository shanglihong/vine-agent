import { Project, Session } from '../types';
import { request } from './client';

export async function fetchProjects(userId: string): Promise<Project[]> {
  const data = await request<Project[]>(`/api/projects?user_id=${userId}`);
  return data;
}

export async function createProject(
  userId: string,
  name: string,
  description = '',
  metadata: Record<string, string> = {}
): Promise<{ id: string; status: string }> {
  const data = await request<{ id: string; status: string }>('/api/projects', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, name, description, metadata }),
  });
  return data;
}

export async function updateProject(
  projectId: string,
  name: string,
  description = '',
  metadata: Record<string, string> = {}
): Promise<void> {
  await request<void>(`/api/projects/${projectId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, metadata }),
  });
}

export async function deleteProject(projectId: string): Promise<void> {
  await request<void>(`/api/projects/${projectId}`, {
    method: 'DELETE',
  });
}

export async function fetchProjectSessions(projectId: string): Promise<Session[]> {
  const data = await request<Session[]>(`/api/projects/${projectId}/sessions`);
  return data;
}
