import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { App as AntApp, ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import 'antd/dist/reset.css'

import App from './App'
import './styles/global.css'
import './styles/kanban.css'

dayjs.locale('zh-cn')

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 30_000,
    },
  },
})

async function enableMocking() {
  if (!import.meta.env.DEV || import.meta.env.VITE_ENABLE_MSW !== 'true') {
    return
  }

  const { worker } = await import('./mocks/browser')
  await worker.start({ onUnhandledRequest: 'bypass' })
}

const rootElement = document.getElementById('root')

if (!rootElement) {
  throw new Error('Root element not found')
}

enableMocking().then(() => {
  createRoot(rootElement).render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <ConfigProvider
          locale={zhCN}
          theme={{
            token: {
              borderRadius: 6,
              colorPrimary: '#2563eb',
              fontFamily:
                'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
            },
            components: {
              Layout: {
                headerBg: '#ffffff',
                siderBg: '#111827',
              },
            },
          }}
        >
          <AntApp>
            <App />
          </AntApp>
        </ConfigProvider>
      </QueryClientProvider>
    </StrictMode>,
  )
})
