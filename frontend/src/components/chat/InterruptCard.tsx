import { PendingTool } from '../../types';

interface InterruptCardProps {
  pendingInterrupt: {
    session_id: string;
    pending_tools: PendingTool[];
  };
  onApproveInterrupt: () => void;
  onRejectInterrupt: () => void;
}

export default function InterruptCard({
  pendingInterrupt,
  onApproveInterrupt,
  onRejectInterrupt,
}: InterruptCardProps) {
  return (
    <div className="interrupt-approval-card">
      <div className="interrupt-header">
        <svg className="warning-icon-svg" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z" />
        </svg>
        <span className="interrupt-title">Security Authorization Confirm: Sensitive Tool Call Detected</span>
      </div>
      <p className="interrupt-desc">
        The system detected tool execution requests containing sensitive modify/delete actions. The execution flow has been intercepted and suspended. Please review the parameters for authorization:
      </p>
      <div className="tools-requested-list">
        {pendingInterrupt.pending_tools.map((tool) => (
          <div key={tool.id} className="tool-req-item">
            <div className="tool-req-name">🔧 Intended Tool: {tool.function.name}</div>
            <div className="tool-req-args">
              <strong>Parameters (JSON):</strong>
              <br />
              {tool.function.arguments}
            </div>
          </div>
        ))}
      </div>
      <div className="approval-btn-group">
        <button className="btn-approve" onClick={onApproveInterrupt}>
          Approve
        </button>
        <button className="btn-reject" onClick={onRejectInterrupt}>
          Reject
        </button>
      </div>
    </div>
  );
}
