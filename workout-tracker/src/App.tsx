import { WorkoutContent } from './components'

function App() {
  const handleComplete = () => {
    alert('¡Rutina completada! 💪')
  }

  return (
    <div className="min-h-screen bg-white overflow-auto flex justify-center">
      <div className="w-full max-w-[400px] px-6 py-8">
        <WorkoutContent onComplete={handleComplete} />
      </div>
    </div>
  )
}

export default App
