import React, { useState, useRef, useEffect } from 'react';

interface RightDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  icon?: React.ReactNode;
  headerActions?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  defaultWidth?: number;
  minWidth?: number;
  maxWidth?: number;
}

export default function RightDrawer({
  isOpen,
  onClose,
  title,
  icon,
  headerActions,
  children,
  className = '',
}: RightDrawerProps) {
  const getScreenConfig = () => {
    if (typeof window !== 'undefined' && window.innerWidth >= 1450) {
      return {
        defaultWidth: 700,
        minWidth: 640,
        maxWidth: 1000,
      };
    }
    return {
      defaultWidth: 300,
      minWidth: 300,
      maxWidth: 550,
    };
  };

  const config = getScreenConfig();
  const [width, setWidth] = useState(config.defaultWidth);
  const [isResizing, setIsResizing] = useState(false);
  const isResizingRef = useRef(false);

  // 当 isOpen 变为 true 时，重置宽度为当前视口对应的默认宽度，避免 React HMR 或页面缓存状态错位
  useEffect(() => {
    if (isOpen) {
      const currentConfig = getScreenConfig();
      setWidth(currentConfig.defaultWidth);
    }
  }, [isOpen]);

  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault();
    isResizingRef.current = true;
    setIsResizing(true);
    document.body.style.cursor = 'ew-resize';
    document.body.style.userSelect = 'none';
  };

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizingRef.current) return;
      // 抽屉在右侧，向左拉伸：新宽度 = 浏览器窗口宽度 - 鼠标当前 X 坐标
      const newWidth = window.innerWidth - e.clientX;
      const currentConfig = getScreenConfig();
      if (newWidth >= currentConfig.minWidth && newWidth <= currentConfig.maxWidth) {
        setWidth(newWidth);
      }
    };

    const handleMouseUp = () => {
      if (isResizingRef.current) {
        isResizingRef.current = false;
        setIsResizing(false);
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      }
    };

    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);
    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, []);

  return (
    <aside
      className={`right-drawer-panel ${className} ${isOpen ? '' : 'collapsed'} ${isResizing ? 'resizing' : ''}`}
      style={{ width: isOpen ? width : 0 }}
    >
      {/* 拖拽手柄 */}
      {isOpen && (
        <div
          className="drawer-resize-handle"
          onMouseDown={handleMouseDown}
        />
      )}
      <header className="right-drawer-header">
        <div className="right-drawer-header-title">
          {icon}
          <span>{title}</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          {headerActions}
          <button
            className="right-drawer-header-btn right-drawer-close"
            onClick={onClose}
          >
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
              style={{ width: 14, height: 14 }}
            >
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
      </header>
      <div className="right-drawer-content">
        {children}
      </div>
    </aside>
  );
}
