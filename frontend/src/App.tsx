import { useState, useEffect, useRef } from 'react';
import Sidebar from './components/Sidebar';
import ChatArea from './components/ChatArea';
import MemoryPanel from './components/MemoryPanel';
import { UserInfo } from './types';
import { fetchUserInfo, createSession, deleteSession, renameSession } from './api';
import { useSession } from './hooks/useSession';
import { useProfile } from './hooks/useProfile';
import { useChat } from './hooks/useChat';
import ConfirmModal from './components/ConfirmModal';
import RightDrawer from './components/RightDrawer';

export default function App() {
  const [currentSessionID, setCurrentSessionID] = useState<string>('');
  const [sessionToDelete, setSessionToDelete] = useState<string | null>(null);
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
  const [userID, setUserID] = useState<string>('');
  const [selectedModel, setSelectedModel] = useState<string>('deepseek-v4-flash');

  // 状态：深色/明亮模式
  const [isDarkMode, setIsDarkMode] = useState<boolean>(false);

  // User 维度状态：长期记忆画像面板折叠状态，默认折叠。跨会话保持状态
  const [isMemoryCollapsed, setIsMemoryCollapsed] = useState<boolean>(true);

  // Session 维度状态：联网搜索结果面板。切换会话时自动关闭
  const [searchResults, setSearchResults] = useState<{ title: string; url: string; snippet: string }[]>([]);
  const [fetchedUrls, setFetchedUrls] = useState<Set<string>>(new Set());
  const [isSearchPanelOpen, setIsSearchPanelOpen] = useState(false);

  // URL 归一化：提取实际目标URL，去掉尾部斜杠、统一小写 scheme+host
  const normalizeUrl = (url: string): string => {
    try {
      let targetUrl = url.trim();
      if (targetUrl.includes('uddg=')) {
        const u = new URL(targetUrl);
        const uddg = u.searchParams.get('uddg');
        if (uddg) {
          targetUrl = decodeURIComponent(uddg);
        }
      } else if (targetUrl.includes('duckduckgo.com/l/?')) {
        const match = targetUrl.match(/[?&]uddg=([^&]+)/);
        if (match && match[1]) {
          targetUrl = decodeURIComponent(match[1]);
        }
      }

      const u = new URL(targetUrl);
      return (u.origin + u.pathname).replace(/\/+$/, '') + u.search;
    } catch {
      return url.trim().replace(/\/+$/, '');
    }
  };

  const handleOpenSearchResults = (items: { title: string; url: string; snippet: string }[], urls: Set<string>) => {
    // 检查是否重复点击了相同的 sources
    const isSame = isSearchPanelOpen &&
      searchResults.length === items.length &&
      searchResults.every((val, index) => val.url === items[index]?.url);

    if (isSame) {
      setIsSearchPanelOpen(false);
    } else {
      setSearchResults(items);
      // 存储归一化后的 URL 集合，方便匹配
      setFetchedUrls(new Set([...urls].map(normalizeUrl)));
      setIsSearchPanelOpen(true);
      setIsMemoryCollapsed(true);
    }
  };

  // 全局 Tooltip 状态
  const [tooltipText, setTooltipText] = useState<string>('');
  const [tooltipPos, setTooltipPos] = useState<{ x: number; y: number }>({ x: 0, y: 0 });
  const tooltipTimeoutRef = useRef<any>(null);

  const handleShowTooltip = (text: string, e: React.MouseEvent) => {
    if (tooltipTimeoutRef.current) {
      clearTimeout(tooltipTimeoutRef.current);
    }
    const x = e.clientX;
    const y = e.clientY;
    tooltipTimeoutRef.current = setTimeout(() => {
      setTooltipText(text);
      setTooltipPos({ x, y });
    }, 800);
  };

  const handleMoveTooltip = (e: React.MouseEvent) => {
    const x = e.clientX;
    const y = e.clientY;
    setTooltipPos({ x, y });
  };

  const handleHideTooltip = () => {
    if (tooltipTimeoutRef.current) {
      clearTimeout(tooltipTimeoutRef.current);
      tooltipTimeoutRef.current = null;
    }
    setTooltipText('');
  };

  // ── Hooks ──
  const { userProfile, loadProfile, evolveProfile } = useProfile(userID);

  const { sessions, loadSessions } = useSession(userID);

  const {
    messages,
    setMessages,
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
    webSearchEnabled,
    setWebSearchEnabled,
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
    setIsSearchPanelOpen(false); // 切换 Session 时自动收起属于 Session 维度的搜索源抽屉
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

  // 4. 删除指定会话
  const handleDeleteSession = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (isStreaming) {
      alert('正在生成中，无法删除会话');
      return;
    }
    setSessionToDelete(id);
  };

  const confirmDeleteSession = async () => {
    if (!sessionToDelete) return;
    const id = sessionToDelete;
    setSessionToDelete(null);
    try {
      await deleteSession(id);
      
      let nextActiveId = currentSessionID;
      if (id === currentSessionID) {
        const remaining = sessions.filter((s) => s.id !== id);
        if (remaining.length > 0) {
          nextActiveId = remaining[0].id;
        } else {
          nextActiveId = '';
        }
      }
      
      await loadSessions(nextActiveId);
      
      if (nextActiveId) {
        selectSession(nextActiveId);
      } else {
        setCurrentSessionID('');
        setPendingInterrupt(null);
        setMessages([]);
      }
    } catch (err: any) {
      alert('删除会话失败: ' + err.message);
      console.error('删除会话失败:', err);
    }
  };

  const handleRenameSession = async (id: string, newName: string) => {
    try {
      await renameSession(id, newName);
      await loadSessions(currentSessionID);
    } catch (err: any) {
      alert('重命名会话失败: ' + err.message);
      console.error('重命名会话失败:', err);
    }
  };

  return (
    <div className={`portal-container ${isMemoryCollapsed && !isSearchPanelOpen ? 'no-right-panels' : ''}`}>
      <Sidebar
        sessions={sessions}
        currentSessionID={currentSessionID}
        isStreaming={isStreaming}
        pendingInterrupt={pendingInterrupt}
        userInfo={userInfo}
        userID={userID}
        isDarkMode={isDarkMode}
        isMemoryCollapsed={isMemoryCollapsed}
        setIsMemoryCollapsed={(v) => {
          setIsMemoryCollapsed(v);
          if (!v) {
            setIsSearchPanelOpen(false);
          }
        }}
        onSelectSession={selectSession}
        onCreateNewSession={createNewSession}
        onToggleTheme={toggleTheme}
        onDeleteSession={handleDeleteSession}
        onRenameSession={handleRenameSession}
        onShowTooltip={handleShowTooltip}
        onMoveTooltip={handleMoveTooltip}
        onHideTooltip={handleHideTooltip}
      />
      <ChatArea
        messages={messages}
        currentSessionID={currentSessionID}
        currentSessionName={sessions.find(s => s.id === currentSessionID)?.name}
        isStreaming={isStreaming}
        pendingInterrupt={pendingInterrupt}
        selectedModel={selectedModel}
        expandedReasoning={expandedReasoning}
        setExpandedReasoning={setExpandedReasoning}
        webSearchEnabled={webSearchEnabled}
        setWebSearchEnabled={setWebSearchEnabled}
        onSendMessage={(text) => handleSendMessage(text, currentSessionID)}
        onApproveInterrupt={() => handleApproveInterrupt(currentSessionID)}
        onRejectInterrupt={handleRejectInterrupt}
        onCancelChat={() => handleCancelChat(currentSessionID)}
        setSelectedModel={setSelectedModel}
        username={userInfo?.username}
        onShowTooltip={handleShowTooltip}
        onMoveTooltip={handleMoveTooltip}
        onHideTooltip={handleHideTooltip}
        onOpenSearchResults={handleOpenSearchResults}
        isSearchPanelOpen={isSearchPanelOpen}
        searchResults={searchResults}
      />
      <MemoryPanel
        userProfile={userProfile}
        isMemoryCollapsed={isMemoryCollapsed}
        onClose={() => setIsMemoryCollapsed(true)}
      />
      {/* 搜索结果右侧抽屉面板 */}
      <RightDrawer
        isOpen={isSearchPanelOpen}
        onClose={() => setIsSearchPanelOpen(false)}
        title="Web Sources"
        icon={
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 15, height: 15, flexShrink: 0 }}>
            <circle cx="12" cy="12" r="10" />
            <line x1="2" y1="12" x2="22" y2="12" />
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
          </svg>
        }
        className="search-results-panel"
      >
        <div className="search-results-list">
          {fetchedUrls.size > 0 ? (
            <>
              {/* Read Pages 分区头部 */}
              {searchResults.some(item => fetchedUrls.has(normalizeUrl(item.url))) && (() => {
                const count = searchResults.filter(item => fetchedUrls.has(normalizeUrl(item.url))).length;
                return (
                  <div className="search-results-divider-wrapper">
                    <svg
                      className="search-results-divider-icon"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2.5"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z" />
                      <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z" />
                    </svg>
                    <span className="search-results-divider">Read Pages</span>
                    <span className="divider-count-badge read-count">{count}</span>
                  </div>
                );
              })()}
              {/* 被 fetch_webpage 实际抓取的页面置顶高亮 */}
              {searchResults.filter(item => fetchedUrls.has(normalizeUrl(item.url))).map((item, sIdx) => (
                <a key={`fetched-${sIdx}`} href={item.url} target="_blank" rel="noopener noreferrer" className="search-result-item fetched">
                  <div className="search-result-top">
                    <span
                      className="search-result-fetched-icon"
                      onMouseEnter={(e) => handleShowTooltip("Fully read by AI", e)}
                      onMouseMove={handleMoveTooltip}
                      onMouseLeave={handleHideTooltip}
                    >
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" style={{ width: 10, height: 10 }}>
                        <polyline points="20 6 9 17 4 12" />
                      </svg>
                    </span>
                    <span
                      className="search-result-title"
                      onMouseEnter={(e) => handleShowTooltip(item.title, e)}
                      onMouseMove={handleMoveTooltip}
                      onMouseLeave={handleHideTooltip}
                    >
                      {item.title}
                    </span>
                  </div>
                  <span
                    className="search-result-url"
                    onMouseEnter={(e) => handleShowTooltip(item.url, e)}
                    onMouseMove={handleMoveTooltip}
                    onMouseLeave={handleHideTooltip}
                  >
                    {item.url}
                  </span>
                  {item.snippet && <p className="search-result-snippet">{item.snippet}</p>}
                </a>
              ))}

              {/* Other Results 分区头部 */}
              {searchResults.some(item => !fetchedUrls.has(normalizeUrl(item.url))) && (() => {
                const count = searchResults.filter(item => !fetchedUrls.has(normalizeUrl(item.url))).length;
                return (
                  <div className="search-results-divider-wrapper">
                    <svg
                      className="search-results-divider-icon"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2.5"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <circle cx="12" cy="12" r="10" />
                      <path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20" />
                      <path d="M2 12h20" />
                    </svg>
                    <span className="search-results-divider">Other Results</span>
                    <span className="divider-count-badge other-count">{count}</span>
                  </div>
                );
              })()}
              {searchResults.filter(item => !fetchedUrls.has(normalizeUrl(item.url))).map((item, sIdx) => (
                <a key={`search-${sIdx}`} href={item.url} target="_blank" rel="noopener noreferrer" className="search-result-item">
                  <div className="search-result-top">
                    <span className="search-result-index">{sIdx + 1}</span>
                    <span
                      className="search-result-title"
                      onMouseEnter={(e) => handleShowTooltip(item.title, e)}
                      onMouseMove={handleMoveTooltip}
                      onMouseLeave={handleHideTooltip}
                    >
                      {item.title}
                    </span>
                  </div>
                  <span
                    className="search-result-url"
                    onMouseEnter={(e) => handleShowTooltip(item.url, e)}
                    onMouseMove={handleMoveTooltip}
                    onMouseLeave={handleHideTooltip}
                  >
                    {item.url}
                  </span>
                  {item.snippet && <p className="search-result-snippet">{item.snippet}</p>}
                </a>
              ))}
            </>
          ) : (
            <>
              {/* All Results 统一头部 */}
              <div className="search-results-divider-wrapper">
                <svg
                  className="search-results-divider-icon"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20" />
                  <path d="M2 12h20" />
                </svg>
                <span className="search-results-divider">All Results</span>
                <span className="divider-count-badge other-count">{searchResults.length}</span>
              </div>
              {searchResults.map((item, sIdx) => (
                <a key={`search-${sIdx}`} href={item.url} target="_blank" rel="noopener noreferrer" className="search-result-item">
                  <div className="search-result-top">
                    <span className="search-result-index">{sIdx + 1}</span>
                    <span
                      className="search-result-title"
                      onMouseEnter={(e) => handleShowTooltip(item.title, e)}
                      onMouseMove={handleMoveTooltip}
                      onMouseLeave={handleHideTooltip}
                    >
                      {item.title}
                    </span>
                  </div>
                  <span
                    className="search-result-url"
                    onMouseEnter={(e) => handleShowTooltip(item.url, e)}
                    onMouseMove={handleMoveTooltip}
                    onMouseLeave={handleHideTooltip}
                  >
                    {item.url}
                  </span>
                  {item.snippet && <p className="search-result-snippet">{item.snippet}</p>}
                </a>
              ))}
            </>
          )}
        </div>
      </RightDrawer>
      <ConfirmModal
        isOpen={sessionToDelete !== null}
        title="删除会话"
        message="此操作将永久清除该会话的所有历史消息。"
        confirmText="删除"
        cancelText="取消"
        onConfirm={confirmDeleteSession}
        onCancel={() => setSessionToDelete(null)}
      />
      {tooltipText && (
        <div
          className="global-tooltip"
          style={{
            position: 'fixed',
            left: tooltipPos.x + 12,
            top: tooltipPos.y + 12,
            pointerEvents: 'none',
            zIndex: 10000,
          }}
        >
          {tooltipText}
        </div>
      )}
    </div>
  );
}
