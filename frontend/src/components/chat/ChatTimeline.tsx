import React, { useState, useEffect } from 'react';
import { Message } from '../../types';

interface ChatTimelineProps {
  messages: Message[];
  scrollContainerRef: React.RefObject<HTMLDivElement>;
  onNodeClick: (index: number) => void;
  onShowTooltip: (text: string, e: React.MouseEvent) => void;
  onMoveTooltip: (e: React.MouseEvent) => void;
  onHideTooltip: () => void;
}

export default function ChatTimeline({
  messages,
  scrollContainerRef,
  onNodeClick,
  onShowTooltip,
  onMoveTooltip,
  onHideTooltip,
}: ChatTimelineProps) {
  const [activeNodeIdx, setActiveNodeIdx] = useState<number | null>(null);

  // 筛选出所有用户发送和助理回复的消息
  const chatNodes = messages
    .map((msg, index) => ({ msg, index }))
    .filter(({ msg, index }) => {
      if (msg.role === 'user') return true;
      if (msg.role === 'assistant') {
        // 保留有内容、有推理，或者是当前正在响应的最后一条消息
        return (
          (msg.content && msg.content.trim().length > 0) ||
          (msg.reasoning_content && msg.reasoning_content.trim().length > 0) ||
          index === messages.length - 1
        );
      }
      return false;
    });

  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const handleScroll = () => {
      const msgElements = container.querySelectorAll('.message-wrapper');
      let currentActiveIdx: number | null = null;
      let minDistance = Infinity;

      msgElements.forEach((el) => {
        const rect = el.getBoundingClientRect();
        const containerRect = container.getBoundingClientRect();
        // 计算该元素距离容器视口顶部的绝对偏差
        const distance = Math.abs(rect.top - containerRect.top);
        if (distance < minDistance) {
          minDistance = distance;
          const idAttr = el.getAttribute('id');
          if (idAttr) {
            const match = idAttr.match(/msg-(\d+)/);
            if (match) {
              currentActiveIdx = parseInt(match[1], 10);
            }
          }
        }
      });

      if (currentActiveIdx !== null) {
        setActiveNodeIdx(currentActiveIdx);
      }
    };

    container.addEventListener('scroll', handleScroll, { passive: true });
    const timer = setTimeout(handleScroll, 100);

    return () => {
      container.removeEventListener('scroll', handleScroll);
      clearTimeout(timer);
    };
  }, [messages, scrollContainerRef]);

  // 如果总有效节点数少于 2，则不需要显示时间轴
  if (chatNodes.length < 2) {
    return null;
  }

  // 辅助变量：动态计算用户问题轮次
  let userRound = 0;

  return (
    <div className="chat-timeline">
      {/* 背景连线 */}
      <div className="timeline-track" />

      {/* 各个节点 */}
      {chatNodes.map(({ msg, index }) => {
        const isUser = msg.role === 'user';
        if (isUser) {
          userRound++;
        }

        const isActive = activeNodeIdx === index;
        const rawContent = msg.content || '';

        let nodeTitle = '';
        if (isUser) {
          nodeTitle = `Round ${userRound} Question: ${rawContent}`;
        } else {
          const rawAnswer = rawContent || msg.reasoning_content || '';
          const maxLen = 70;
          const previewText = rawAnswer.length > maxLen ? rawAnswer.slice(0, maxLen).trim() + '...' : rawAnswer;
          nodeTitle = `Round ${userRound} Answer: ${previewText}`;
        }

        return (
          <div
            key={index}
            className={`timeline-node-container ${isUser ? 'user' : 'assistant'} ${isActive ? 'active' : ''}`}
            onClick={() => {
              onNodeClick(index);
              onHideTooltip();
            }}
            onMouseEnter={(e) => onShowTooltip(nodeTitle, e)}
            onMouseMove={onMoveTooltip}
            onMouseLeave={onHideTooltip}
          >
            <div className="timeline-node-circle">
              {isUser ? (
                <svg className="node-icon" viewBox="0 0 24 24" width="10" height="10" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                  <circle cx="12" cy="7" r="4" />
                </svg>
              ) : (
                <svg className="node-icon" viewBox="0 0 24 24" width="10" height="10" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M12 2v4M12 18v4M4 12H2M22 12h-4M12 6c0 3.3-2.7 6-6 6 3.3 0 6 2.7 6 6 0-3.3 2.7-6 6-6-3.3 0-6-2.7-6-6Z" />
                </svg>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
