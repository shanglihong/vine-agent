import { useState } from 'react';
import { Session } from '../types';
import { fetchSessions } from '../api';

export function useSession(userID: string) {
  const [sessions, setSessions] = useState<Session[]>([]);

  // 1. 获取会话历史列表（不含自动选中，由调用方决定）
  const loadSessions = async (
    currentSessionID: string,
  ) => {
    if (!userID) return;
    try {
      const data = await fetchSessions(currentSessionID, userID, '');
      setSessions(data);
    } catch (err: any) {
      alert('Failed to load sessions. Network or backend connection error: ' + err.message);
      console.error('加载会话失败:', err);
    }
  };

  return { sessions, loadSessions };
}
