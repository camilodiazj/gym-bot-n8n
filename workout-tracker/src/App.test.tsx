import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import App from './App'

// Mock fetch globally
const mockFetch = vi.fn()
global.fetch = mockFetch

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Reset window.location.search
    Object.defineProperty(window, 'location', {
      value: { search: '' },
      writable: true,
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('Loading state', () => {
    it('should show loading spinner initially', () => {
      mockFetch.mockImplementation(() => new Promise(() => {})) // Never resolves

      render(<App />)

      expect(screen.getByText('Cargando rutina...')).toBeInTheDocument()
    })
  })

  describe('Error handling', () => {
    it('should show error message when fetch fails', async () => {
      mockFetch.mockRejectedValue(new Error('Network error'))

      render(<App />)

      await waitFor(() => {
        expect(screen.getByText(/Error de conexi.n/)).toBeInTheDocument()
      })
    })

    it('should show expired token error on 401', async () => {
      mockFetch.mockResolvedValue({
        status: 401,
        json: () => Promise.resolve({}),
      })

      render(<App />)

      await waitFor(() => {
        expect(screen.getByText('El link ha expirado. Solicita uno nuevo via WhatsApp.')).toBeInTheDocument()
      })
    })

    it('should show rest day message when no workout scheduled', async () => {
      mockFetch.mockResolvedValue({
        status: 200,
        json: () => Promise.resolve({
          success: false,
          error: { message: 'No workout scheduled for today' },
        }),
      })

      render(<App />)

      await waitFor(() => {
        expect(screen.getByText('Dia de descanso')).toBeInTheDocument()
        expect(screen.getByText('No tienes rutina programada para hoy. Descansa!')).toBeInTheDocument()
      })
    })

    it('should show custom error message from API', async () => {
      mockFetch.mockResolvedValue({
        status: 200,
        json: () => Promise.resolve({
          success: false,
          error: { message: 'Custom error from API' },
        }),
      })

      render(<App />)

      await waitFor(() => {
        expect(screen.getByText('Error')).toBeInTheDocument()
        expect(screen.getByText('Custom error from API')).toBeInTheDocument()
      })
    })

    it('should show default error when API returns success:false without message', async () => {
      mockFetch.mockResolvedValue({
        status: 200,
        json: () => Promise.resolve({
          success: false,
        }),
      })

      render(<App />)

      // When there's no message, it shows "No workout scheduled for today"
      // which triggers the rest day UI
      await waitFor(() => {
        expect(screen.getByText('Dia de descanso')).toBeInTheDocument()
      })
    })
  })

  describe('Successful workout fetch', () => {
    const mockWorkoutData = {
      success: true,
      data: {
        session_id: 'session-123',
        session_name: 'Pierna y Core',
        week: 2,
        day_name: 'Martes',
        exercises: [
          {
            id: 'ex-1',
            name: 'Sentadilla',
            badgeColor: '#374151',
            rir: '2-3',
            sets: [
              { id: 'set-1', setNumber: 1, reps: 12, kg: '50', completed: false },
              { id: 'set-2', setNumber: 2, reps: 12, kg: '50', completed: false },
            ],
            tips: [{ text: 'Mantener espalda recta' }],
            steps: [{ text: 'Bajar controladamente' }],
          },
          {
            id: 'ex-2',
            name: 'Prensa',
            badgeColor: '#374151',
            rir: '3-4',
            sets: [
              { id: 'set-3', setNumber: 1, reps: 15, kg: '80', completed: false },
            ],
            tips: [{ text: 'No bloquear rodillas' }],
            steps: [{ text: 'Empujar con talones' }],
          },
        ],
      },
    }

    it('should render workout content on successful fetch', async () => {
      mockFetch.mockResolvedValue({
        status: 200,
        json: () => Promise.resolve(mockWorkoutData),
      })

      render(<App />)

      await waitFor(() => {
        expect(screen.getByText('Pierna y Core')).toBeInTheDocument()
        expect(screen.getByText('2 ejercicios')).toBeInTheDocument()
        expect(screen.getByText('Sentadilla')).toBeInTheDocument()
        expect(screen.getByText('Prensa')).toBeInTheDocument()
      })
    })

    it('should expand first exercise by default', async () => {
      mockFetch.mockResolvedValue({
        status: 200,
        json: () => Promise.resolve(mockWorkoutData),
      })

      render(<App />)

      await waitFor(() => {
        // First exercise should have its instructions expanded (tips are prefixed with "- ")
        expect(screen.getByText(/- Mantener espalda recta/)).toBeInTheDocument()
      })
    })
  })

  describe('URL parameter handling', () => {
    it('should use code from URL when present', async () => {
      Object.defineProperty(window, 'location', {
        value: { search: '?c=ABC123' },
        writable: true,
      })

      mockFetch.mockResolvedValue({
        status: 200,
        json: () => Promise.resolve({ success: true, data: { exercises: [] } }),
      })

      render(<App />)

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledWith(
          expect.stringContaining('?c=ABC123')
        )
      })
    })

    it('should use dev user_id when no code in URL', async () => {
      Object.defineProperty(window, 'location', {
        value: { search: '' },
        writable: true,
      })

      mockFetch.mockResolvedValue({
        status: 200,
        json: () => Promise.resolve({ success: true, data: { exercises: [] } }),
      })

      render(<App />)

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledWith(
          expect.stringContaining('user_id=')
        )
      })
    })

    it('should URL encode special characters in code', async () => {
      // "+" in URL query is parsed as space by URLSearchParams
      // so "test+code" becomes "test code" and then encodeURIComponent makes it "test%20code"
      Object.defineProperty(window, 'location', {
        value: { search: '?c=test+code' },
        writable: true,
      })

      mockFetch.mockResolvedValue({
        status: 200,
        json: () => Promise.resolve({ success: true, data: { exercises: [] } }),
      })

      render(<App />)

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledWith(
          expect.stringContaining('c=test%20code')
        )
      })
    })
  })

  describe('Complete button', () => {
    it('should show alert when complete button is clicked', async () => {
      const alertMock = vi.spyOn(window, 'alert').mockImplementation(() => {})

      mockFetch.mockResolvedValue({
        status: 200,
        json: () => Promise.resolve({
          success: true,
          data: {
            session_name: 'Test',
            exercises: [
              {
                id: '1',
                name: 'Test Exercise',
                badgeColor: '#374151',
                rir: '2-3',
                sets: [{ setNumber: 1, reps: 10, kg: '20', completed: false }],
                tips: [],
                steps: [],
              },
            ],
          },
        }),
      })

      render(<App />)

      await waitFor(() => {
        expect(screen.getByText('Completar Rutina')).toBeInTheDocument()
      })

      const completeButton = screen.getByText('Completar Rutina')
      completeButton.click()

      expect(alertMock).toHaveBeenCalledWith('Rutina completada!')

      alertMock.mockRestore()
    })
  })
})
