import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { InstancesProvider } from './contexts/InstancesContext'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <InstancesProvider>
      <App />
    </InstancesProvider>
  </React.StrictMode>,
)