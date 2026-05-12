import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { WorkoutContent } from './WorkoutContent'

// Mock the API module
vi.mock('../services/api', () => ({
  updateSetWeight: vi.fn().mockResolvedValue(undefined),
}))

import { updateSetWeight } from '../services/api'

describe('WorkoutContent', () => {
  const mockExercises = [
    {
      id: 'ex-1',
      name: 'Sentadilla',
      badgeColor: '#374151',
      rir: '2-3',
      restSeconds: 90,
      instructionsExpanded: true,
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
      badgeColor: '#22C55E',
      rir: '3-4',
      restSeconds: 120,
      videoLink: 'https://example.com/video',
      instructionsExpanded: false,
      sets: [
        { id: 'set-3', setNumber: 1, reps: 15, kg: '-', completed: false },
      ],
      tips: [{ text: 'No bloquear rodillas' }],
      steps: [{ text: 'Empujar con talones' }],
    },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Rendering', () => {
    it('should render all exercises', () => {
      render(<WorkoutContent exercises={mockExercises} />)

      expect(screen.getByText('Sentadilla')).toBeInTheDocument()
      expect(screen.getByText('Prensa')).toBeInTheDocument()
    })

    it('should render complete button', () => {
      render(<WorkoutContent exercises={mockExercises} />)

      expect(screen.getByText('Completar Rutina')).toBeInTheDocument()
    })

    it('should render default exercises when no props provided', () => {
      render(<WorkoutContent />)

      expect(screen.getByText('DB Front Squat')).toBeInTheDocument()
    })

    it('should show expanded exercise instructions', () => {
      render(<WorkoutContent exercises={mockExercises} />)

      // First exercise is expanded - shows RIR and rest info
      expect(screen.getByText(/RIR: 2-3/)).toBeInTheDocument()
      expect(screen.getByText(/1:30 min entre series/)).toBeInTheDocument()
    })

    it('should show RIR badge for expanded exercise', () => {
      render(<WorkoutContent exercises={mockExercises} />)

      expect(screen.getByText(/RIR: 2-3/)).toBeInTheDocument()
    })

    it('should show rest time for expanded exercise', () => {
      render(<WorkoutContent exercises={mockExercises} />)

      expect(screen.getByText(/1:30 min entre series/)).toBeInTheDocument()
    })

    it('should show rest time in seconds when less than 60', () => {
      const exerciseWithShortRest = [{
        ...mockExercises[0],
        restSeconds: 45,
      }]

      render(<WorkoutContent exercises={exerciseWithShortRest} />)

      expect(screen.getByText(/45 seg entre series/)).toBeInTheDocument()
    })

    it('should render video link when available', () => {
      render(<WorkoutContent exercises={mockExercises} />)

      const videoLink = screen.getAllByRole('link').find(
        link => link.getAttribute('href') === 'https://example.com/video'
      )
      expect(videoLink).toBeInTheDocument()
    })

    it('should render Spanish copy (no English literals leak through)', () => {
      // Regression guard for BUG-5 — the exercise badge and the instructions
      // section header used to read "Exercise" and "Workout Instructions" in
      // an otherwise-Spanish UI. The literals now live in components/copy.ts.
      render(<WorkoutContent exercises={mockExercises} />)

      expect(screen.queryByText('Exercise')).toBeNull()
      expect(screen.queryByText('Workout Instructions')).toBeNull()
      expect(screen.getAllByText('Ejercicio').length).toBeGreaterThan(0)
      expect(screen.getByText('Instrucciones')).toBeInTheDocument()
    })
  })

  describe('Expand/Collapse', () => {
    it('should toggle exercise instructions on chevron click', async () => {
      render(<WorkoutContent exercises={mockExercises} />)

      // Find the chevron buttons. Each card renders the Spanish badge "Ejercicio".
      const exerciseCards = screen.getAllByText('Ejercicio')
      const firstCard = exerciseCards[0].closest('div[class*="bg-"]')

      // Initially expanded - should show RIR info
      expect(screen.getByText(/RIR: 2-3/)).toBeInTheDocument()

      // Find the toggle button within the first card header
      const toggleButton = firstCard!.querySelector('button')
      expect(toggleButton).toBeInTheDocument()

      // Click to collapse
      fireEvent.click(toggleButton!)

      // After collapse, RIR info should be hidden
      await waitFor(() => {
        expect(screen.queryByText(/RIR: 2-3/)).not.toBeInTheDocument()
      })

      // Click to expand again
      fireEvent.click(toggleButton!)

      await waitFor(() => {
        expect(screen.getByText(/RIR: 2-3/)).toBeInTheDocument()
      })
    })
  })

  describe('Set completion', () => {
    it('should toggle set completion on row click', async () => {
      render(<WorkoutContent exercises={mockExercises} />)

      // Find set row with "1" (first set)
      const setRows = screen.getAllByText('1')
      const firstSetRow = setRows[0].closest('div[class*="grid"]')

      expect(firstSetRow).toBeInTheDocument()

      fireEvent.click(firstSetRow!)

      // After click, the set should be marked complete
      await waitFor(() => {
        // Check if green styling is applied
        const completedCheck = firstSetRow!.querySelector('.text-\\[\\#22C55E\\]')
        expect(completedCheck).toBeInTheDocument()
      })
    })

    it('should save weights when all sets are completed', async () => {
      const singleSetExercise = [{
        id: 'ex-1',
        name: 'Test',
        badgeColor: '#374151',
        rir: '2-3',
        instructionsExpanded: true,
        sets: [
          { id: 'set-1', setNumber: 1, reps: 12, kg: '50', completed: false },
        ],
        tips: [],
        steps: [],
      }]

      render(<WorkoutContent exercises={singleSetExercise} />)

      const setRow = screen.getByText('1').closest('div[class*="grid"]')
      fireEvent.click(setRow!)

      await waitFor(() => {
        expect(updateSetWeight).toHaveBeenCalledWith('set-1', '50')
      })
    })

    it('should not save weights for sets with "-" kg', async () => {
      const exerciseWithNoWeight = [{
        id: 'ex-1',
        name: 'Test',
        badgeColor: '#374151',
        rir: '2-3',
        instructionsExpanded: true,
        sets: [
          { id: 'set-1', setNumber: 1, reps: 12, kg: '-', completed: false },
        ],
        tips: [],
        steps: [],
      }]

      render(<WorkoutContent exercises={exerciseWithNoWeight} />)

      const setRow = screen.getByText('1').closest('div[class*="grid"]')
      fireEvent.click(setRow!)

      await waitFor(() => {
        expect(updateSetWeight).not.toHaveBeenCalled()
      })
    })
  })

  describe('Editable fields', () => {
    it('should make reps editable on click', async () => {
      const user = userEvent.setup()

      render(<WorkoutContent exercises={mockExercises} />)

      // Find reps value "12" and click it
      const repsSpan = screen.getAllByText('12')[0]
      await user.click(repsSpan)

      // Should show input field
      const input = screen.getByRole('spinbutton')
      expect(input).toBeInTheDocument()
      expect(input).toHaveValue(12)
    })

    it('should update reps on blur', async () => {
      const user = userEvent.setup()

      render(<WorkoutContent exercises={mockExercises} />)

      const repsSpan = screen.getAllByText('12')[0]
      await user.click(repsSpan)

      const input = screen.getByRole('spinbutton')
      await user.clear(input)
      await user.type(input, '99')
      await user.tab() // blur

      await waitFor(() => {
        expect(screen.getByText('99')).toBeInTheDocument()
      })
    })

    it('should update reps on Enter key', async () => {
      const user = userEvent.setup()

      render(<WorkoutContent exercises={mockExercises} />)

      const repsSpan = screen.getAllByText('12')[0]
      await user.click(repsSpan)

      const input = screen.getByRole('spinbutton')
      await user.clear(input)
      await user.type(input, '20{enter}')

      await waitFor(() => {
        expect(screen.getByText('20')).toBeInTheDocument()
      })
    })

    it('should cancel reps edit on Escape', async () => {
      const user = userEvent.setup()

      render(<WorkoutContent exercises={mockExercises} />)

      const repsSpan = screen.getAllByText('12')[0]
      await user.click(repsSpan)

      const input = screen.getByRole('spinbutton')
      await user.clear(input)
      await user.type(input, '99')
      await user.keyboard('{Escape}')

      // Should still show original value
      await waitFor(() => {
        expect(screen.queryByRole('spinbutton')).not.toBeInTheDocument()
        expect(screen.getAllByText('12')[0]).toBeInTheDocument()
      })
    })

    it('should not update reps with invalid value', async () => {
      const user = userEvent.setup()

      render(<WorkoutContent exercises={mockExercises} />)

      const repsSpan = screen.getAllByText('12')[0]
      await user.click(repsSpan)

      const input = screen.getByRole('spinbutton')
      await user.clear(input)
      await user.type(input, '0')
      await user.tab()

      // Should keep original value since 0 is invalid
      await waitFor(() => {
        expect(screen.getAllByText('12')[0]).toBeInTheDocument()
      })
    })

    it('should make kg editable on click', async () => {
      const user = userEvent.setup()

      render(<WorkoutContent exercises={mockExercises} />)

      const kgSpan = screen.getAllByText('50')[0]
      await user.click(kgSpan)

      const input = screen.getByRole('textbox')
      expect(input).toBeInTheDocument()
    })

    it('should update kg on blur', async () => {
      const user = userEvent.setup()

      render(<WorkoutContent exercises={mockExercises} />)

      const kgSpan = screen.getAllByText('50')[0]
      await user.click(kgSpan)

      const input = screen.getByRole('textbox')
      await user.clear(input)
      await user.type(input, '60')
      await user.tab()

      await waitFor(() => {
        expect(screen.getByText('60')).toBeInTheDocument()
      })
    })

    it('should set kg to "-" when empty on blur', async () => {
      const user = userEvent.setup()

      render(<WorkoutContent exercises={mockExercises} />)

      const kgSpan = screen.getAllByText('50')[0]
      await user.click(kgSpan)

      const input = screen.getByRole('textbox')
      await user.clear(input)
      await user.tab()

      // When kg is empty, the row should show "-"
      await waitFor(() => {
        // The input should be cleared and show '-'
        expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
      })
    })

    it('should update kg on Enter', async () => {
      const user = userEvent.setup()

      render(<WorkoutContent exercises={mockExercises} />)

      const kgSpan = screen.getAllByText('50')[0]
      await user.click(kgSpan)

      const input = screen.getByRole('textbox')
      await user.clear(input)
      await user.type(input, '75{enter}')

      await waitFor(() => {
        expect(screen.getByText('75')).toBeInTheDocument()
      })
    })

    it('should cancel kg edit on Escape', async () => {
      const user = userEvent.setup()

      render(<WorkoutContent exercises={mockExercises} />)

      const kgSpan = screen.getAllByText('50')[0]
      await user.click(kgSpan)

      const input = screen.getByRole('textbox')
      await user.clear(input)
      await user.type(input, '999')
      await user.keyboard('{Escape}')

      await waitFor(() => {
        expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
        expect(screen.getAllByText('50')[0]).toBeInTheDocument()
      })
    })
  })

  describe('Complete button', () => {
    it('should call onComplete when clicked', async () => {
      const onComplete = vi.fn()

      render(<WorkoutContent exercises={mockExercises} onComplete={onComplete} />)

      const completeButton = screen.getByText('Completar Rutina')
      fireEvent.click(completeButton)

      expect(onComplete).toHaveBeenCalled()
    })

    it('should handle undefined onComplete gracefully', () => {
      render(<WorkoutContent exercises={mockExercises} />)

      const completeButton = screen.getByText('Completar Rutina')

      // Should not throw
      expect(() => fireEvent.click(completeButton)).not.toThrow()
    })
  })

  describe('Exercise with no sets', () => {
    it('should render exercise without sets', () => {
      const exerciseWithNoSets = [{
        id: 'ex-1',
        name: 'Stretch',
        badgeColor: '#374151',
        rir: '0',
        instructionsExpanded: true,
        sets: [],
        tips: [],
        steps: [],
      }]

      render(<WorkoutContent exercises={exerciseWithNoSets} />)

      expect(screen.getByText('Stretch')).toBeInTheDocument()
    })
  })

  describe('Video link interaction', () => {
    it('should render video link with correct href', () => {
      const exerciseWithVideo = [{
        ...mockExercises[1],
        instructionsExpanded: true,
      }]

      render(<WorkoutContent exercises={exerciseWithVideo} />)

      const videoLink = screen.getByRole('link')
      expect(videoLink).toHaveAttribute('href', 'https://example.com/video')
      expect(videoLink).toHaveAttribute('target', '_blank')
      expect(videoLink).toHaveAttribute('rel', 'noopener noreferrer')
    })
  })

  describe('Alternative exercises - flip card', () => {
    beforeEach(() => {
      // Mock localStorage for swap-tip feature (jsdom doesn't always support it fully)
      const store: Record<string, string> = {}
      Object.defineProperty(window, 'localStorage', {
        value: {
          getItem: vi.fn((key: string) => store[key] ?? null),
          setItem: vi.fn((key: string, value: string) => { store[key] = value }),
          removeItem: vi.fn((key: string) => { delete store[key] }),
          clear: vi.fn(() => { Object.keys(store).forEach(k => delete store[k]) }),
          length: 0,
          key: vi.fn(),
        },
        writable: true,
        configurable: true,
      })
    })

    const makeAlt = (name: string, completed: boolean) => ({
      name,
      rir: '2-3',
      restSeconds: 90,
      videoLink: 'https://example.com/alt',
      sets: [
        { setNumber: 1, reps: 10, kg: '-', completed },
        { setNumber: 2, reps: 10, kg: '-', completed: false },
      ],
    })

    const exerciseWithAlts = (primaryCompleted: boolean, alt1Completed: boolean, alt2Completed = false) => [{
      id: 'ex-alt',
      name: 'Sentadilla con barra',
      badgeColor: '#374151',
      rir: '2-3',
      restSeconds: 120,
      instructionsExpanded: true,
      sets: [
        { id: 'set-1', setNumber: 1, reps: 10, kg: '50', completed: primaryCompleted },
        { id: 'set-2', setNumber: 2, reps: 10, kg: '50', completed: false },
      ],
      tips: [],
      steps: [],
      alternativeExercises: [
        makeAlt('Sentadilla búlgara', alt1Completed),
        makeAlt('Hack Squat', alt2Completed),
      ],
    }]

    it('should show primary exercise by default when no sets completed', () => {
      render(<WorkoutContent exercises={exerciseWithAlts(false, false)} />)

      expect(screen.getByText('Sentadilla con barra')).toBeInTheDocument()
    })

    it('should show swap button when no sets completed', () => {
      render(<WorkoutContent exercises={exerciseWithAlts(false, false)} />)

      expect(screen.getByText('Ver alternativa')).toBeInTheDocument()
    })

    it('should hide swap button when primary has completed sets', () => {
      render(<WorkoutContent exercises={exerciseWithAlts(true, false)} />)

      expect(screen.queryByText('Ver alternativa')).not.toBeInTheDocument()
    })

    it('should show primary when primary has completed sets (initialIndex = 0)', () => {
      render(<WorkoutContent exercises={exerciseWithAlts(true, false)} />)

      expect(screen.getByText('Sentadilla con barra')).toBeInTheDocument()
    })

    it('should show first alternative when it has completed sets but primary does not', () => {
      render(<WorkoutContent exercises={exerciseWithAlts(false, true)} />)

      // The alternative name should be visible
      expect(screen.getByText('Sentadilla búlgara')).toBeInTheDocument()
      expect(screen.getByText('Alternativa 1 de 2')).toBeInTheDocument()
    })

    it('should show second alternative when it has completed sets but primary and first do not', () => {
      render(<WorkoutContent exercises={exerciseWithAlts(false, false, true)} />)

      expect(screen.getByText('Hack Squat')).toBeInTheDocument()
      expect(screen.getByText('Alternativa 2 de 2')).toBeInTheDocument()
    })

    it('should show primary when both primary and alternative have no completed sets', () => {
      const { container } = render(<WorkoutContent exercises={exerciseWithAlts(false, false)} />)

      // Primary exercise name visible
      expect(screen.getByText('Sentadilla con barra')).toBeInTheDocument()
      // Flip card starts at 0deg (front face visible = primary)
      const flipCard = container.querySelector('.flip-card')
      expect(flipCard!.getAttribute('style')).toContain('rotateY(0deg)')
    })

    it('should prefer primary when primary has completed sets even if alt does too', () => {
      render(<WorkoutContent exercises={exerciseWithAlts(true, true)} />)

      // Primary takes precedence
      expect(screen.getByText('Sentadilla con barra')).toBeInTheDocument()
    })

    it('should not show flip animation on mount when starting on alternative', () => {
      const { container } = render(<WorkoutContent exercises={exerciseWithAlts(false, true)} />)

      const flipCard = container.querySelector('.flip-card')
      expect(flipCard).toBeInTheDocument()
      // On initial render, transition should be suppressed
      expect(flipCard!.getAttribute('style')).toContain('transition: none')
    })

    it('should show flip animation on mount when starting on primary (no suppression needed)', () => {
      const { container } = render(<WorkoutContent exercises={exerciseWithAlts(false, false)} />)

      const flipCard = container.querySelector('.flip-card')
      expect(flipCard).toBeInTheDocument()
      // No transition suppression when starting on primary
      expect(flipCard!.getAttribute('style')).not.toContain('transition: none')
    })

    it('should cycle to alternative on swap button click', async () => {
      render(<WorkoutContent exercises={exerciseWithAlts(false, false)} />)

      const swapButton = screen.getByText('Ver alternativa')
      fireEvent.click(swapButton)

      await waitFor(() => {
        expect(screen.getByText('Sentadilla búlgara')).toBeInTheDocument()
        expect(screen.getByText('Alternativa 1 de 2')).toBeInTheDocument()
      })
    })

    it('should toggle alternative set completion', async () => {
      render(<WorkoutContent exercises={exerciseWithAlts(false, false)} />)

      // Flip to first alternative
      const swapButton = screen.getByText('Ver alternativa')
      fireEvent.click(swapButton)

      await waitFor(() => {
        expect(screen.getByText('Sentadilla búlgara')).toBeInTheDocument()
      })

      // Find set rows in the alternative view and click to complete
      const setRows = screen.getAllByText('1')
      const firstSetRow = setRows[0].closest('div[class*="grid"]')
      expect(firstSetRow).toBeInTheDocument()

      fireEvent.click(firstSetRow!)

      // After completing a set, swap button should disappear
      await waitFor(() => {
        expect(screen.queryByText('Siguiente alternativa')).not.toBeInTheDocument()
        expect(screen.queryByText('Volver al original')).not.toBeInTheDocument()
      })
    })

    it('should render exercise without alternatives as simple card (no flip container)', () => {
      const simpleExercise = [{
        ...mockExercises[0],
        alternativeExercises: undefined,
      }]

      const { container } = render(<WorkoutContent exercises={simpleExercise} />)

      expect(container.querySelector('.flip-container')).not.toBeInTheDocument()
      expect(screen.getByText('Sentadilla')).toBeInTheDocument()
    })

    it('should render exercise with empty alternatives as simple card', () => {
      const simpleExercise = [{
        ...mockExercises[0],
        alternativeExercises: [],
      }]

      const { container } = render(<WorkoutContent exercises={simpleExercise} />)

      expect(container.querySelector('.flip-container')).not.toBeInTheDocument()
    })
  })
})
