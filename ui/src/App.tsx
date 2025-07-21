import Header from '@/components/Header'
import InstanceList from '@/components/InstanceList'

function App() {
  return (
    <div className="min-h-screen bg-gray-50">
      <Header />
      <main className="container mx-auto max-w-4xl px-4 py-8">
        <InstanceList />
      </main>
    </div>
  )
}

export default App