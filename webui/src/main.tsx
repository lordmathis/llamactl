import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { InstancesProvider } from './contexts/InstancesContext'
import './index.css'
import { AuthProvider } from './contexts/AuthContext'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AuthProvider>
      <InstancesProvider>
        <App />
      </InstancesProvider>
    </AuthProvider>
  </React.StrictMode>,
)