import React, { useState, useEffect } from 'react';
import { X, Wrench, CheckCircle, AlertTriangle, Cpu, Hash } from 'lucide-react';
import { AgentRun } from '../types';
import { agentApi } from '../api/agent';

interface RunHistoryModalProps {
  runId: string;
  isOpen: boolean;
  onClose: () => void;
}

export const RunHistoryModal: React.FC<RunHistoryModalProps> = ({ runId, isOpen, onClose }) => {
  const [run, setRun] = useState<AgentRun | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isOpen || !runId) return;
    setIsLoading(true);
    setError(null);
    agentApi.getAgentRun(runId)
      .then(setRun)
      .catch((err) => setError(err.message || 'Failed to load run details'))
      .finally(() => setIsLoading(false));
  }, [runId, isOpen]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
      <div 
        className="w-full max-w-xl max-h-[85vh] bg-[#202020] border border-[#2e2e2e] rounded-xl shadow-2xl flex flex-col overflow-hidden text-[#e6e6e6]"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Modal Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-[#2b2b2b]">
          <div className="flex items-center gap-2">
            <Cpu className="w-4 h-4 text-[#2383e2]" />
            <h3 className="text-sm font-semibold text-[#f0f0f0]">Agent Execution Run</h3>
          </div>
          <button 
            onClick={onClose}
            className="text-[#888888] hover:text-[#e6e6e6] p-1 rounded hover:bg-[#2b2b2b] transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Modal Content */}
        <div className="flex-1 overflow-y-auto p-5 space-y-4">
          {isLoading && (
            <div className="py-8 text-center text-xs text-[#888888]">
              Loading execution trace...
            </div>
          )}

          {error && (
            <div className="p-3 rounded-md bg-[#2b1717] border border-[#4a2222] text-xs text-[#f87171]">
              {error}
            </div>
          )}

          {run && (
            <>
              {/* Summary Stats */}
              <div className="grid grid-cols-3 gap-2.5">
                <div className="p-2.5 rounded-lg bg-[#181818] border border-[#2b2b2b]">
                  <p className="text-[10px] text-[#737373] uppercase tracking-wider">Status</p>
                  <p className="text-xs font-semibold text-[#e6e6e6] capitalize mt-0.5">{run.status}</p>
                </div>
                <div className="p-2.5 rounded-lg bg-[#181818] border border-[#2b2b2b]">
                  <p className="text-[10px] text-[#737373] uppercase tracking-wider">Iterations</p>
                  <p className="text-xs font-semibold text-[#e6e6e6] mt-0.5">{run.iterations} turn(s)</p>
                </div>
                <div className="p-2.5 rounded-lg bg-[#181818] border border-[#2b2b2b]">
                  <p className="text-[10px] text-[#737373] uppercase tracking-wider">Tokens</p>
                  <p className="text-xs font-semibold text-[#e6e6e6] mt-0.5">
                    {run.input_tokens + run.output_tokens} total
                  </p>
                </div>
              </div>

              {/* Tool Calls */}
              <div>
                <h4 className="text-xs font-semibold text-[#a0a0a0] mb-2 flex items-center gap-1.5">
                  <Wrench className="w-3.5 h-3.5" />
                  Tool Calls Executed ({run.tool_calls?.length || 0})
                </h4>

                {!run.tool_calls || run.tool_calls.length === 0 ? (
                  <p className="text-xs text-[#737373] italic">No external tools invoked for this turn.</p>
                ) : (
                  <div className="space-y-2">
                    {run.tool_calls.map((call, idx) => (
                      <div 
                        key={call.id || idx}
                        className="p-3 rounded-lg bg-[#181818] border border-[#2b2b2b] text-xs space-y-2"
                      >
                        <div className="flex items-center justify-between">
                          <span className="font-mono font-medium text-[#2383e2]">
                            {call.tool_name}()
                          </span>
                          {call.is_error ? (
                            <span className="inline-flex items-center gap-1 text-[10px] text-[#f87171]">
                              <AlertTriangle className="w-3 h-3" /> Error
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 text-[10px] text-[#4ade80]">
                              <CheckCircle className="w-3 h-3" /> Success
                            </span>
                          )}
                        </div>

                        {/* Input */}
                        {call.input && (
                          <div>
                            <span className="text-[10px] text-[#666666] uppercase font-mono">Input</span>
                            <pre className="mt-0.5 p-2 rounded bg-[#121212] border border-[#222222] font-mono text-[11px] text-[#a0a0a0] overflow-x-auto">
                              {typeof call.input === 'string' ? call.input : JSON.stringify(call.input, null, 2)}
                            </pre>
                          </div>
                        )}

                        {/* Output */}
                        {call.output && (
                          <div>
                            <span className="text-[10px] text-[#666666] uppercase font-mono">Result</span>
                            <pre className="mt-0.5 p-2 rounded bg-[#121212] border border-[#222222] font-mono text-[11px] text-[#a0a0a0] overflow-x-auto max-h-36">
                              {typeof call.output === 'string' ? call.output : JSON.stringify(call.output, null, 2)}
                            </pre>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </>
          )}
        </div>

        {/* Modal Footer */}
        <div className="p-3 border-t border-[#2b2b2b] bg-[#1a1a1a] flex justify-end">
          <button
            onClick={onClose}
            className="px-3 py-1.5 rounded-md text-xs font-medium bg-[#2b2b2b] hover:bg-[#333333] text-[#e6e6e6] transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
