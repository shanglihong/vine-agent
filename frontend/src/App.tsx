import { useState, useEffect } from 'react';
import Sidebar from './components/Sidebar';
import ChatArea from './components/ChatArea';
import MemoryPanel from './components/MemoryPanel';
import { UserInfo } from './types';
import { fetchUserInfo, createSession } from './api';
import { useSession } from './hooks/useSession';
import { useProfile } from './hooks/useProfile';
import { useChat } from './hooks/useChat';

export default function App() {
  const [currentSessionID, setCurrentSessionID] = useState<string>('');
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
  const [userID, setUserID] = useState<string>('');
  const [selectedModel, setSelectedModel] = useState<string>('deepseek-v4-flash');

  // 状态：深色/明亮模式
  const [isDarkMode, setIsDarkMode] = useState<boolean>(false);

  // 状态：长期记忆画像面板折叠状态，默认折叠
  const [isMemoryCollapsed, setIsMemoryCollapsed] = useState<boolean>(true);

  // ── Hooks ──
  const { userProfile, isEvolving, loadProfile, evolveProfile } = useProfile(userID);

  const { sessions, loadSessions } = useSession(userID);

  const {
    messages,
    isStreaming,
    pendingInterrupt,
    setPendingInterrupt,
    expandedReasoning,
    setExpandedReasoning,
    rebuildAndSetMessages,
    handleSendMessage,
    handleApproveInterrupt,
    handleRejectInterrupt,
    handleCancelChat,
  } = useChat({
    userID,
    selectedModel,
    loadSessions: () => loadSessions(currentSessionID),
    evolveProfile,
  });

  // 初始化加载及主题检测
  useEffect(() => {
    const loadUser = async (retries = 5, delay = 1500) => {
      try {
        const u = await fetchUserInfo();
        if (u && u.id) {
          setUserInfo(u);
          setUserID(u.id);
          return;
        }
        throw new Error('User info has no valid ID');
      } catch (err) {
        if (retries > 0) {
          console.warn(`加载用户信息失败，将在 ${delay}ms 后重试... 剩余重试次数: ${retries}`);
          setTimeout(() => loadUser(retries - 1, delay), delay);
        } else {
          console.error('加载用户信息失败，已达到最大重试次数:', err);
        }
      }
    };

    loadUser();

    const savedTheme = localStorage.getItem('theme');
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    if (savedTheme === 'dark' || (!savedTheme && prefersDark)) {
      setIsDarkMode(true);
      document.documentElement.classList.add('dark');
    } else {
      setIsDarkMode(false);
      document.documentElement.classList.remove('dark');
    }
  }, []);

  // 当 userID 就绪后，再加载会话历史和画像
  useEffect(() => {
    if (userID) {
      loadSessions(currentSessionID, (firstId) => selectSession(firstId));
      loadProfile();
    }
  }, [userID]);

  // 切换主题
  const toggleTheme = () => {
    if (isDarkMode) {
      document.documentElement.classList.remove('dark');
      localStorage.setItem('theme', 'light');
      setIsDarkMode(false);
    } else {
      document.documentElement.classList.add('dark');
      localStorage.setItem('theme', 'dark');
      setIsDarkMode(true);
    }
  };

  // 2. 加载指定会话的消息
  const selectSession = async (id: string) => {
    if (isStreaming) return;
    setCurrentSessionID(id);
    setPendingInterrupt(null);
    setExpandedReasoning({}); // 重置折叠状态
    try {
      const data = await rebuildAndSetMessages(id);
      if (data?.status === 'pending_confirmation') {
        const lastMsg = data.messages?.[data.messages.length - 1];
        if (lastMsg && lastMsg.tool_calls) {
          setPendingInterrupt({
            session_id: id,
            pending_tools: lastMsg.tool_calls.map((tc: any) => ({
              id: tc.id,
              function: { name: tc.function.name, arguments: tc.function.arguments },
            })),
          });
        }
      }
    } catch (err: any) {
      alert('读取历史消息失败，网络或后端连接异常: ' + err.message);
      console.error('切换会话失败:', err);
    }
  };

  // 3. 创建全新会话
  const createNewSession = async () => {
    if (isStreaming) return;
    const newSessionID = 'sess_' + Math.random().toString(36).substr(2, 9);
    try {
      await createSession(newSessionID, userID);
      await loadSessions(currentSessionID);
      selectSession(newSessionID);
    } catch (err: any) {
      alert('创建会话失败，网络或后端连接异常: ' + err.message);
      console.error('创建会话失败:', err);
    }
  };

  return (
    <div className="portal-container">
      <Sidebar
        sessions={sessions}
        currentSessionID={currentSessionID}
        isStreaming={isStreaming}
        pendingInterrupt={pendingInterrupt}
        userInfo={userInfo}
        userID={userID}
        isDarkMode={isDarkMode}
        onSelectSession={selectSession}
        onCreateNewSession={createNewSession}
        onToggleTheme={toggleTheme}
      />
      <ChatArea
        messages={messages}
        currentSessionID={currentSessionID}
        isStreaming={isStreaming}
        pendingInterrupt={pendingInterrupt}
        selectedModel={selectedModel}
        expandedReasoning={expandedReasoning}
        setExpandedReasoning={setExpandedReasoning}
        onSendMessage={(text) => handleSendMessage(text, currentSessionID)}
        onApproveInterrupt={() => handleApproveInterrupt(currentSessionID)}
        onRejectInterrupt={handleRejectInterrupt}
        onCancelChat={() => handleCancelChat(currentSessionID)}
        setSelectedModel={setSelectedModel}
        isMemoryCollapsed={isMemoryCollapsed}
        setIsMemoryCollapsed={setIsMemoryCollapsed}
      />
      <MemoryPanel
        userProfile={userProfile}
        isMemoryCollapsed={isMemoryCollapsed}
        isEvolving={isEvolving}
        currentSessionID={currentSessionID}
        onEvolveProfile={() => evolveProfile(currentSessionID)}
      />
    </div>
  );
}
