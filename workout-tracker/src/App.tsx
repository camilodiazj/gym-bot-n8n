import { useState, useEffect } from 'react'
import { WorkoutContent } from './components'

// API Configuration
const API_BASE_URL = 'https://workout-api-148665080566.us-central1.run.app/api/v1'
// Fallback user ID for development (only used when no code in URL)
const DEV_USER_ID = '0a220ce8-00e8-4eda-bbf4-112a7fd1e57d'

interface SetData {
  id: string
  setNumber: number
  reps: number
  kg: string
  completed: boolean
}

interface Tip {
  text: string
}

interface Step {
  text: string
}

interface ExerciseData {
  id: string
  name: string
  badgeColor: string
  sets: SetData[]
  tips: Tip[]
  steps: Step[]
  rir: string
  restSeconds?: number
  videoLink?: string
  instructionsExpanded?: boolean
}

interface WorkoutResponse {
  success: boolean
  data: {
    session_id: string
    session_name: string
    week: number
    day_name: string
    exercises: ExerciseData[]
  }
  error?: {
    code: number
    message: string
  }
}

function App() {
  const [exercises, setExercises] = useState<ExerciseData[]>([])
  const [sessionName, setSessionName] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchWorkout = async () => {
      try {
        setLoading(true)
        setError(null)

        // Parse auth from URL (?c=shortcode)
        const params = new URLSearchParams(window.location.search)
        const code = params.get('c')

        // Build the API URL based on auth method
        let apiUrl: string
        if (code) {
          // Production: use short code
          apiUrl = `${API_BASE_URL}/workouts/today?c=${encodeURIComponent(code)}`
        } else {
          // Development fallback: use user_id directly
          apiUrl = `${API_BASE_URL}/workouts/today?user_id=${DEV_USER_ID}`
        }

        const response = await fetch(apiUrl)

        // Handle 401 Unauthorized (expired or invalid token)
        if (response.status === 401) {
          setError('El link ha expirado. Solicita uno nuevo via WhatsApp.')
          setLoading(false)
          return
        }

        const data: WorkoutResponse = await response.json()

        if (data.success && data.data) {
          // Add instructionsExpanded: false to each exercise
          const exercisesWithState = data.data.exercises.map((ex, index) => ({
            ...ex,
            instructionsExpanded: index === 0, // First exercise expanded by default
          }))
          setExercises(exercisesWithState)
          setSessionName(data.data.session_name)
        } else {
          setError(data.error?.message || 'No workout scheduled for today')
        }
      } catch (err) {
        setError('Error de conexión. Intenta de nuevo.')
        console.error('Fetch error:', err)
      } finally {
        setLoading(false)
      }
    }

    fetchWorkout()
  }, [])

  const handleComplete = () => {
    alert('Rutina completada!')
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-white flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-[#22C55E] mx-auto mb-4"></div>
          <p className="text-[#6B7280] font-['DM_Sans']">Cargando rutina...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen bg-white flex items-center justify-center">
        <div className="text-center px-6">
          <div className="w-16 h-16 bg-[#FEE2E2] rounded-full flex items-center justify-center mx-auto mb-4">
            <span className="text-2xl">😴</span>
          </div>
          <h2 className="text-xl font-bold font-['Bricolage_Grotesque'] text-[#1A1A1A] mb-2">
            {error === 'No workout scheduled for today' ? 'Dia de descanso' : 'Error'}
          </h2>
          <p className="text-[#6B7280] font-['DM_Sans']">
            {error === 'No workout scheduled for today'
              ? 'No tienes rutina programada para hoy. Descansa!'
              : error}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-white overflow-auto flex justify-center">
      <div className="w-full max-w-[400px] px-6 py-8">
        {/* Session Header */}
        {sessionName && (
          <div className="mb-6">
            <h1 className="text-2xl font-bold font-['Bricolage_Grotesque'] text-[#1A1A1A]">
              {sessionName}
            </h1>
            <p className="text-[#6B7280] text-sm font-['DM_Sans']">
              {exercises.length} ejercicios
            </p>
          </div>
        )}
        <WorkoutContent exercises={exercises} onComplete={handleComplete} />
      </div>
    </div>
  )
}

export default App
