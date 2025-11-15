import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { InstancesProvider } from './contexts/InstancesContext'
import './index.css'
import { AuthProvider } from './contexts/AuthContext'
import { ConfigProvider } from './contexts/ConfigContext'

ReactDOM.createRoot(document.getElementById('root')!).render(
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