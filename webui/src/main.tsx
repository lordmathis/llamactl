import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { InstancesProvider } from './contexts/InstancesContext'
import './index.css'
import { AuthProvider } from './contexts/AuthContext'
import { ConfigProvider } from './contexts/ConfigContext'

const rootElement = document.getElementById('root')
if (!rootElement) throw new Error('Failed to find the root element')

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <AuthProvider>
      <ConfigProvider>
        <InstancesProvider>
          <App />
        </InstancesProvider>
      </ConfigProvider>
    </AuthProvider>
  </React.StrictMode>,
)