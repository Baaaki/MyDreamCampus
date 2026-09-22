import { Component, type ErrorInfo, type ReactNode } from "react"

type Props = {
  children: ReactNode
}

type State = {
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // eslint-disable-next-line no-console
    console.error("ErrorBoundary caught:", error, info.componentStack)
  }

  reset = () => {
    this.setState({ error: null })
  }

  render() {
    if (!this.state.error) return this.props.children

    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4">
        <div className="w-full max-w-md space-y-6 text-center">
          <p className="text-6xl font-extrabold text-red-600">Hata</p>
          <div className="space-y-2">
            <h1 className="text-2xl font-bold text-gray-900">
              Bir şeyler ters gitti
            </h1>
            <p className="text-sm text-gray-600">
              Sayfayı yüklerken beklenmedik bir hata oluştu. Lütfen tekrar
              deneyin.
            </p>
            {import.meta.env.DEV && (
              <pre className="mt-4 max-h-40 overflow-auto rounded bg-gray-100 p-3 text-left text-xs">
                {this.state.error.message}
              </pre>
            )}
          </div>
          <div className="flex justify-center gap-3">
            <button
              onClick={this.reset}
              className="rounded-md bg-indigo-600 px-4 py-2 text-white hover:bg-indigo-700"
            >
              Tekrar dene
            </button>
            <button
              onClick={() => (window.location.href = "/")}
              className="rounded-md border border-gray-300 px-4 py-2 text-gray-700 hover:bg-gray-50"
            >
              Ana sayfa
            </button>
          </div>
        </div>
      </div>
    )
  }
}
