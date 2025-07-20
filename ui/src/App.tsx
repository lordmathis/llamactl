import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

function App() {
  return (
    <div className="p-8">
      <h1 className="text-3xl font-bold mb-6">Llamactl Dashboard</h1>
      
      <Card className="w-96">
        <CardHeader>
          <CardTitle>Sample Instance</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="mb-4">Status: Running</p>
          <Button>Stop Instance</Button>
        </CardContent>
      </Card>
    </div>
  )
}

export default App