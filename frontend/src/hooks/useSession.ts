import { useState } from 'react';
import { Session } from '../types';
import { fetchSessions } from '../api';

export function useSession(userID: string) {
  const [sessions, setSessions] = useState<Session[]>([]);

  // 1. 获取会话历史列表（不含自动选中，由调用方决定）
  const loadSessions = async (
    currentSessionID: string,
    onFirstLoad?: (firstSessionId: string) => void,
  ) => {
    if (!userID) return;
    try {
      const data = await fetchSessions(userID);
      setSessions(data);
      // 如果有会话，默认选中第一个
      if (data.length > 0 && !currentSessionID && onFirstLoad) {
        onFirstLoad(data[0].id);
      }
    } catch (err: any) {
      alert('加载会话失败，网络或后端连接异常: ' + err.message);
      console.error('加载会话失败:', err);
    }
  };

  return { sessions, loadSessions };
}
