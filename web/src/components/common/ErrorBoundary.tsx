import { Component, type ErrorInfo, type ReactNode } from 'react'
import { Alert, Button } from 'antd'

interface ErrorBoundaryProps {
  children: ReactNode
}

interface ErrorBoundaryState {
  error: Error | null
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Route crashed', error, info)
  }

  render() {
    if (this.state.error) {
      return (
        <div className="route-error">
          <Alert
            action={
              <Button onClick={() => window.location.reload()} type="primary">
                刷新页面
              </Button>
            }
            description={this.state.error.message}
            message="页面加载失败"
            showIcon
            type="error"
          />
        </div>
      )
    }

    return this.props.children
  }
}
