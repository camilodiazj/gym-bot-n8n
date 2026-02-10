import { useState, useEffect } from 'react'
import { WorkoutContent } from './components'
import { CompletionCelebration } from './components/CompletionCelebration'
import { completeWorkout } from './services/api'
import config from './config'

interface SetData {
  id?: string
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

interface AlternativeExercise {
  name: string
  rir: string
  restSeconds?: number
  videoLink: string
  sets: SetData[]
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
  alternativeExercises?: AlternativeExercise[]
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
  const [sessionId, setSessionId] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isCompleted, setIsCompleted] = useState(false)
  const [showCelebration, setShowCelebration] = useState(false)
  const [isCompleting, setIsCompleting] = useState(false)

  useEffect(() => {
    const fetchWorkout = async () => {
      try {
        setLoading(true)
        setError(null)

        // Parse auth from URL (?c=shortcode)
        const params = new URLSearchParams(window.location.search)
        const code = params.get('c')

        // Demo mode: show mock exercises with alternative exercise feature
        if (params.has('demo')) {
          setExercises([
            {
              id: 'demo-1',
              name: 'Sentadilla con barra',
              badgeColor: '#374151',
              rir: '3',
              restSeconds: 120,
              videoLink: 'https://www.youtube.com/watch?v=demo-squat',
              instructionsExpanded: true,
              sets: [
                { setNumber: 1, reps: 8, kg: '-', completed: false },
                { setNumber: 2, reps: 9, kg: '-', completed: false },
                { setNumber: 3, reps: 10, kg: '-', completed: false },
              ],
              tips: [],
              steps: [],
              alternativeExercises: [
                {
                  name: 'Goblet Squat',
                  rir: '3',
                  restSeconds: 120,
                  videoLink: 'https://www.youtube.com/watch?v=example1',
                  sets: [
                    { setNumber: 1, reps: 10, kg: '-', completed: false },
                    { setNumber: 2, reps: 10, kg: '-', completed: false },
                    { setNumber: 3, reps: 10, kg: '-', completed: false },
                  ],
                },
                {
                  name: 'Hack Squat',
                  rir: '2-3',
                  restSeconds: 120,
                  videoLink: 'https://www.youtube.com/watch?v=example2',
                  sets: [
                    { setNumber: 1, reps: 10, kg: '-', completed: false },
                    { setNumber: 2, reps: 10, kg: '-', completed: false },
                    { setNumber: 3, reps: 10, kg: '-', completed: false },
                  ],
                },
              ],
            } as ExerciseData,
            {
              id: 'demo-2',
              name: 'Press banca',
              badgeColor: '#374151',
              rir: '2-3',
              restSeconds: 90,
              videoLink: 'https://www.youtube.com/watch?v=demo-bench',
              instructionsExpanded: false,
              sets: [
                { setNumber: 1, reps: 10, kg: '-', completed: false },
                { setNumber: 2, reps: 10, kg: '-', completed: false },
                { setNumber: 3, reps: 10, kg: '-', completed: false },
              ],
              tips: [],
              steps: [],
              alternativeExercises: [
                {
                  name: 'Press inclinado con mancuernas',
                  rir: '2-3',
                  restSeconds: 90,
                  videoLink: 'https://www.youtube.com/watch?v=example3',
                  sets: [
                    { setNumber: 1, reps: 12, kg: '-', completed: false },
                    { setNumber: 2, reps: 12, kg: '-', completed: false },
                    { setNumber: 3, reps: 12, kg: '-', completed: false },
                  ],
                },
                {
                  name: 'Aperturas con mancuernas',
                  rir: '2-3',
                  restSeconds: 75,
                  videoLink: 'https://www.youtube.com/watch?v=example4',
                  sets: [
                    { setNumber: 1, reps: 15, kg: '-', completed: false },
                    { setNumber: 2, reps: 15, kg: '-', completed: false },
                    { setNumber: 3, reps: 15, kg: '-', completed: false },
                  ],
                },
              ],
            } as ExerciseData,
            {
              id: 'demo-3',
              name: 'Plancha',
              badgeColor: '#374151',
              rir: '1-2',
              videoLink: 'https://www.youtube.com/watch?v=demo-plank',
              instructionsExpanded: false,
              sets: [
                { setNumber: 1, reps: 30, kg: '-', completed: false },
                { setNumber: 2, reps: 30, kg: '-', completed: false },
              ],
              tips: [],
              steps: [],
            } as ExerciseData,
          ])
          setSessionName('Demo - Alternativas')
          setSessionId('demo')
          setLoading(false)
          return
        }

        // Build the API URL based on auth method
        let apiUrl: string
        if (code) {
          // Production: use short code
          apiUrl = `${config.apiBaseUrl}/workouts/today?c=${encodeURIComponent(code)}`
        } else if (config.devUserId) {
          // Development fallback: use user_id directly
          apiUrl = `${config.apiBaseUrl}/workouts/today?user_id=${config.devUserId}`
        } else {
          // No auth method available
          setError('No se encontro codigo de acceso. Usa el link de WhatsApp.')
          setLoading(false)
          return
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
          setSessionId(data.data.session_id)
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

  const handleExercisesChange = (updatedExercises: ExerciseData[]) => {
    setExercises(updatedExercises)
  }

  const handleComplete = async () => {
    if (isCompleting || isCompleted || !sessionId) return

    // Check if all sets are completed
    const totalSets = exercises.reduce((sum, ex) => sum + ex.sets.length, 0)
    const completedSets = exercises.reduce(
      (sum, ex) => sum + ex.sets.filter((s) => s.completed).length,
      0
    )

    if (completedSets < totalSets) {
      const confirmed = window.confirm(
        `Solo completaste ${completedSets} de ${totalSets} sets. ¿Seguro que quieres terminar la rutina?`
      )
      if (!confirmed) return
    }

    setIsCompleting(true)
    try {
      await completeWorkout(sessionId)
      setIsCompleted(true)
      setShowCelebration(true)
    } catch (err) {
      console.error('Failed to complete workout:', err)
      alert('Error al completar la rutina. Intenta de nuevo.')
    } finally {
      setIsCompleting(false)
    }
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
      <div className="w-full max-w-[400px] px-6 pt-8 pb-16">
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
        <WorkoutContent
          exercises={exercises}
          onComplete={handleComplete}
          onExercisesChange={handleExercisesChange}
          isCompleted={isCompleted}
          isCompleting={isCompleting}
        />
      </div>

      {/* Celebration overlay */}
      {showCelebration && (
        <CompletionCelebration onDismiss={() => setShowCelebration(false)} />
      )}
    </div>
  )
}

export default App
