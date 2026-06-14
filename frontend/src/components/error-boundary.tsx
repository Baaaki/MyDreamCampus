import { Component, type ErrorInfo, type ReactNode } from "react";

type Props = {
  children: ReactNode;
};

type State = {
  error: Error | null;
};

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // eslint-disable-next-line no-console
    console.error("ErrorBoundary caught:", error, info.componentStack);
  }

  reset = () => {
    this.setState({ error: null });
  };

  render() {
    if (!this.state.error) return this.props.children;

    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
        <div className="max-w-md w-full text-center space-y-6">
          <p className="text-6xl font-extrabold text-red-600">Hata</p>
          <div className="space-y-2">
            <h1 className="text-2xl font-bold text-gray-900">Bir şeyler ters gitti</h1>
            <p className="text-sm text-gray-600">
              Sayfayı yüklerken beklenmedik bir hata oluştu. Lütfen tekrar deneyin.
            </p>
            {import.meta.env.DEV && (
              <pre className="text-xs text-left bg-gray-100 p-3 rounded mt-4 overflow-auto max-h-40">
                {this.state.error.message}
              </pre>
            )}
          </div>
          <div className="flex gap-3 justify-center">
            <button
              onClick={this.reset}
              className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700"
            >
              Tekrar dene
            </button>
            <button
              onClick={() => (window.location.href = "/")}
              className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50"
            >
              Ana sayfa
            </button>
          </div>
        </div>
      </div>
    );
  }
}
