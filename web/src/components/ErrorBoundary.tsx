import React, { Component, ErrorInfo, ReactNode } from 'react';
import { AlertCircle, RefreshCw } from 'lucide-react';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null,
  };

  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Uncaught error caught by ErrorBoundary:', error, errorInfo);
  }

  public render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-[50vh] flex items-center justify-center p-6 font-sans">
          <div className="notion-card p-6 max-w-md w-full space-y-3 text-center">
            <div className="w-8 h-8 rounded-full bg-[#3d1f22] text-[#e57373] flex items-center justify-center mx-auto">
              <AlertCircle className="w-4 h-4" />
            </div>
            <h2 className="text-sm font-semibold text-[#ebebeb]">Something went wrong</h2>
            <p className="text-xs text-[#888888] leading-relaxed">
              {this.state.error?.message || 'An unexpected rendering error occurred.'}
            </p>
            <button
              onClick={() => {
                this.setState({ hasError: false, error: null });
                window.location.reload();
              }}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-[#2383e2] hover:bg-[#1d72c5] text-white text-xs font-medium rounded transition-colors"
            >
              <RefreshCw className="w-3.5 h-3.5" />
              <span>Reload Workspace</span>
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
