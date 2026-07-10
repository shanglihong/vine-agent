import { ReactNode } from 'react';
import ConfirmModal from './ConfirmModal';

interface DeleteConfirmModalProps {
  isOpen: boolean;
  title: string;
  itemName: string;
  itemType: 'session' | 'project';
  onConfirm: () => void;
  onCancel: () => void;
}

export default function DeleteConfirmModal({
  isOpen,
  title,
  itemName,
  itemType,
  onConfirm,
  onCancel,
}: DeleteConfirmModalProps) {
  const getMessage = (): ReactNode => {
    if (itemType === 'session') {
      return (
        <span className="delete-confirm-message-wrapper">
          <span className="delete-confirm-text">You are about to permanently delete this session</span>
          <span className="delete-highlight-name">{itemName}</span>
        </span>
      );
    }
    return (
      <span className="delete-confirm-message-wrapper">
        <span className="delete-confirm-text">You are about to permanently delete this project and all its sessions and messages</span>
        <span className="delete-highlight-name">{itemName}</span>
      </span>
    );
  };

  const warningIcon = (
    <svg
      viewBox="0 0 24 24"
      width="18"
      height="18"
      fill="none"
      stroke="currentColor"
      strokeWidth="2.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      style={{ display: 'block' }}
    >
      <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
      <line x1="12" y1="9" x2="12" y2="13"></line>
      <line x1="12" y1="17" x2="12.01" y2="17"></line>
    </svg>
  );

  return (
    <ConfirmModal
      isOpen={isOpen}
      title={title}
      message={getMessage()}
      confirmText="Delete"
      cancelText="Cancel"
      onConfirm={onConfirm}
      onCancel={onCancel}
      icon={warningIcon}
    />
  );
}
