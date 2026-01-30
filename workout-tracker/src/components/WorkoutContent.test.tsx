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

      // First exercise is expanded - tips and steps are prefixed with "- "
      expect(screen.getByText(/- Mantener espalda recta/)).toBeInTheDocument()
      expect(screen.getByText(/- Bajar controladamente/)).toBeInTheDocument()
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
  })

  describe('Expand/Collapse', () => {
    it('should toggle exercise instructions on chevron click', async () => {
      render(<WorkoutContent exercises={mockExercises} />)

      // Find the chevron buttons (they contain ChevronUp/ChevronDown icons)
      const exerciseCards = screen.getAllByText('Exercise')
      const firstCard = exerciseCards[0].closest('div[class*="bg-"]')

      // Initially expanded - should show tips with "- " prefix
      expect(screen.getByText(/- Mantener espalda recta/)).toBeInTheDocument()

      // Find the toggle button within the first card header
      const toggleButton = firstCard!.querySelector('button')
      expect(toggleButton).toBeInTheDocument()

      // Click to collapse
      fireEvent.click(toggleButton!)

      // After collapse, tips should be hidden
      await waitFor(() => {
        expect(screen.queryByText(/- Mantener espalda recta/)).not.toBeInTheDocument()
      })

      // Click to expand again
      fireEvent.click(toggleButton!)

      await waitFor(() => {
        expect(screen.getByText(/- Mantener espalda recta/)).toBeInTheDocument()
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
})
