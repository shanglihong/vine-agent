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
        <span className="interrupt-title">安全授权确认: 检测到敏感工具调用</span>
      </div>
      <p className="interrupt-desc">
        系统检测到智能体发出包含敏感修改或删除动作的工具调用请求。该执行链已自动拦截并挂起，请审查工具参数以授权批准：
      </p>
      <div className="tools-requested-list">
        {pendingInterrupt.pending_tools.map((tool) => (
          <div key={tool.id} className="tool-req-item">
            <div className="tool-req-name">🔧 拟调用工具: {tool.function.name}</div>
            <div className="tool-req-args">
              <strong>参数明细 (JSON):</strong>
              <br />
              {tool.function.arguments}
            </div>
          </div>
        ))}
      </div>
      <div className="approval-btn-group">
        <button className="btn-approve" onClick={onApproveInterrupt}>
          授权执行
        </button>
        <button className="btn-reject" onClick={onRejectInterrupt}>
          拒绝操作
        </button>
      </div>
    </div>
  );
}
