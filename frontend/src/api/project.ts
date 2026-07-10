import { Project, Session } from '../types';

export async function fetchProjects(userId: string): Promise<Project[]> {
  const res = await fetch(`/api/projects?user_id=${userId}`);
  if (!res.ok) {
    throw new Error(`Failed to fetch projects: status ${res.status}`);
  }
  return res.json();
}

export async function createProject(
  userId: string,
  name: string,
  description = '',
  metadata: Record<string, string> = {}
): Promise<{ id: string; status: string }> {
  const res = await fetch('/api/projects', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, name, description, metadata }),
  });
  if (!res.ok) {
    throw new Error(`Failed to create project: status ${res.status}`);
  }
  return res.json();
}

export async function updateProject(
  projectId: string,
  name: string,
  description = '',
  metadata: Record<string, string> = {}
): Promise<void> {
  const res = await fetch(`/api/projects/${projectId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, metadata }),
  });
  if (!res.ok) {
    throw new Error(`Failed to update project: status ${res.status}`);
  }
}

export async function deleteProject(projectId: string): Promise<void> {
  const res = await fetch(`/api/projects/${projectId}`, {
    method: 'DELETE',
  });
  if (!res.ok) {
    throw new Error(`Failed to delete project: status ${res.status}`);
  }
}

export async function fetchProjectSessions(projectId: string): Promise<Session[]> {
  const res = await fetch(`/api/projects/${projectId}/sessions`);
  if (!res.ok) {
    throw new Error(`Failed to fetch project sessions: status ${res.status}`);
  }
  return res.json();
}
