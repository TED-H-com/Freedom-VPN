import React from 'react';

interface ErrorBoundaryState {
  hasError: boolean;
  errorMessage: string;
}

export class ErrorBoundary extends React.Component<
  { children: React.ReactNode },
  ErrorBoundaryState
> {
  constructor(props: { children: React.ReactNode }) {
    super(props);
    this.state = { hasError: false, errorMessage: '' };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return {
      hasError: true,
      errorMessage: error.message || 'Unexpected error',
    };
  }

  override render(): React.ReactNode {
    if (this.state.hasError) {
      return (
        <div className="card alert-danger" role="alert" style={{ margin: '24px' }}>
          <h2>畫面發生非預期錯誤 / Unexpected Error</h2>
          <p>{this.state.errorMessage}</p>
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => this.setState({ hasError: false, errorMessage: '' })}
          >
            重新載入狀態 / Reset
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
