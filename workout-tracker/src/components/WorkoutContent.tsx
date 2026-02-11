import React, { useState, useRef, useCallback, useEffect } from 'react';
import { Check, ChevronDown, ChevronUp, Play, CheckCircle, Circle, ExternalLink, Repeat2, Info } from 'lucide-react';
import { updateSetWeight } from '../services/api';

// Types
interface SetData {
  id?: string;
  setNumber: number;
  reps: number;
  kg: string;
  completed: boolean;
}

interface Tip {
  text: string;
}

interface Step {
  text: string;
}

interface AlternativeExercise {
  name: string;
  rir: string;
  restSeconds?: number;
  videoLink: string;
  sets: SetData[];
}

interface ExerciseData {
  id: string;
  name: string;
  badgeColor: string;
  sets: SetData[];
  tips: Tip[];
  steps: Step[];
  rir: string;
  restSeconds?: number;
  videoLink?: string;
  instructionsExpanded?: boolean;
  alternativeExercises?: AlternativeExercise[];
}

interface WorkoutContentProps {
  exercises?: ExerciseData[];
  onComplete?: () => void;
  onExercisesChange?: (exercises: ExerciseData[]) => void;
  isCompleted?: boolean;
  isCompleting?: boolean;
}

// Default exercise data matching the design
const defaultExercises: ExerciseData[] = [
  {
    id: '1',
    name: 'DB Front Squat',
    badgeColor: '#374151',
    instructionsExpanded: false,
    rir: '3-4',
    tips: [
      { text: 'Mantener la espalda recta y el core activado durante todo el ejercicio.' },
      { text: 'Mantener los talones en contacto con el suelo durante todo el movimiento.' },
      { text: 'Evitar que las rodillas se extiendan más allá de los dedos de los pies.' },
    ],
    steps: [
      { text: 'Separar las piernas según el ancho de tus hombros.' },
      { text: 'Colocar los pies ligeramente hacia afuera.' },
      { text: 'Sostener una mancuerna en cada mano sobre los hombros.' },
      { text: 'Bajar de forma controlada doblando las rodillas hasta que los muslos estén paralelo al suelo.' },
      { text: 'Subir de forma controlada hasta regresar a la posición inicial.' },
      { text: 'Repetir el movimiento.' },
    ],
    sets: [
      { setNumber: 1, reps: 12, kg: '10', completed: true },
      { setNumber: 2, reps: 18, kg: '-', completed: false },
      { setNumber: 3, reps: 18, kg: '-', completed: false },
      { setNumber: 4, reps: 18, kg: '-', completed: false },
    ],
    alternativeExercises: [
      {
        name: 'Goblet Squat',
        rir: '3-4',
        restSeconds: 120,
        videoLink: 'https://www.youtube.com/watch?v=example1',
        sets: [
          { setNumber: 1, reps: 12, kg: '-', completed: false },
          { setNumber: 2, reps: 12, kg: '-', completed: false },
          { setNumber: 3, reps: 12, kg: '-', completed: false },
        ],
      },
      {
        name: 'Hack Squat',
        rir: '3-4',
        restSeconds: 120,
        videoLink: 'https://www.youtube.com/watch?v=example2',
        sets: [
          { setNumber: 1, reps: 10, kg: '-', completed: false },
          { setNumber: 2, reps: 10, kg: '-', completed: false },
          { setNumber: 3, reps: 10, kg: '-', completed: false },
        ],
      },
    ],
  },
  {
    id: '2',
    name: 'Lunges',
    badgeColor: '#374151',
    instructionsExpanded: true,
    rir: '2-3',
    tips: [
      { text: 'Mantener la espalda recta y el torso erguido durante todo el movimiento.' },
      { text: 'La rodilla delantera no debe sobrepasar la punta del pie.' },
      { text: 'Bajar hasta que ambas rodillas formen ángulos de 90 grados.' },
    ],
    steps: [
      { text: 'De pie, dar un paso hacia adelante con una pierna.' },
      { text: 'Flexionar ambas rodillas bajando el cuerpo.' },
      { text: 'Empujar con el pie delantero para volver a la posición inicial.' },
      { text: 'Alternar piernas en cada repetición.' },
    ],
    sets: [
      { setNumber: 1, reps: 12, kg: '8', completed: true },
      { setNumber: 2, reps: 18, kg: '-', completed: false },
      { setNumber: 3, reps: 18, kg: '-', completed: false },
      { setNumber: 4, reps: 18, kg: '-', completed: false },
    ],
    alternativeExercises: [
      {
        name: 'Bulgarian Split Squat',
        rir: '2-3',
        restSeconds: 90,
        videoLink: 'https://www.youtube.com/watch?v=example3',
        sets: [
          { setNumber: 1, reps: 10, kg: '-', completed: false },
          { setNumber: 2, reps: 10, kg: '-', completed: false },
          { setNumber: 3, reps: 10, kg: '-', completed: false },
        ],
      },
      {
        name: 'Step Up con mancuernas',
        rir: '2-3',
        restSeconds: 90,
        videoLink: 'https://www.youtube.com/watch?v=example4',
        sets: [
          { setNumber: 1, reps: 12, kg: '-', completed: false },
          { setNumber: 2, reps: 12, kg: '-', completed: false },
          { setNumber: 3, reps: 12, kg: '-', completed: false },
        ],
      },
    ],
  },
  {
    id: '3',
    name: 'Plank Hold',
    badgeColor: '#374151',
    instructionsExpanded: true,
    rir: '1-2',
    tips: [
      { text: 'Mantener el cuerpo en línea recta desde la cabeza hasta los talones.' },
      { text: 'Activar el core y glúteos durante todo el ejercicio.' },
      { text: 'No dejar que las caderas suban o bajen.' },
    ],
    steps: [
      { text: 'Colocarse boca abajo apoyándose en antebrazos y puntas de los pies.' },
      { text: 'Alinear los codos debajo de los hombros.' },
      { text: 'Elevar el cuerpo manteniendo una línea recta.' },
      { text: 'Mantener la posición durante el tiempo indicado.' },
    ],
    sets: [
      { setNumber: 1, reps: 12, kg: '-', completed: true },
      { setNumber: 2, reps: 18, kg: '-', completed: false },
      { setNumber: 3, reps: 18, kg: '-', completed: false },
      { setNumber: 4, reps: 18, kg: '-', completed: false },
    ],
  },
];

// Sets Table Component (shared between front and back faces)
const SetsTable: React.FC<{
  sets: SetData[];
  onToggleSet?: (setNumber: number) => void;
  onUpdateReps?: (setNumber: number, reps: number) => void;
  onUpdateKg?: (setNumber: number, kg: string) => void;
  readOnly?: boolean;
}> = ({ sets, onToggleSet, onUpdateReps, onUpdateKg, readOnly = false }) => {
  const [editingReps, setEditingReps] = React.useState<number | null>(null);
  const [editingKg, setEditingKg] = React.useState<number | null>(null);
  const [editValue, setEditValue] = React.useState<string>('');

  return (
    <div className="flex flex-col w-full mt-2">
      {/* Table Header */}
      <div className="grid grid-cols-3 py-3 border-b border-[#E5E7EB] w-full">
        <div className="flex items-center justify-center">
          <span className="text-[#1A1A1A] text-[15px] font-semibold font-['DM_Sans']">Sets</span>
        </div>
        <div className="flex items-center justify-center">
          <span className="text-[#1A1A1A] text-[15px] font-semibold font-['DM_Sans']">Reps</span>
        </div>
        <div className="flex items-center justify-center">
          <span className="text-[#1A1A1A] text-[15px] font-semibold font-['DM_Sans']">Kg</span>
        </div>
      </div>

      {/* Set Rows */}
      {sets.map((set) => (
        <div
          key={set.setNumber}
          onClick={() => !readOnly && onToggleSet?.(set.setNumber)}
          className={`grid grid-cols-3 py-3 border-b border-[#E5E7EB] last:border-b-0 transition-colors ${
            readOnly ? '' : 'cursor-pointer hover:bg-[#E8F5E9]'
          } ${set.completed ? 'bg-[#F0FDF4]' : ''}`}
        >
          <div className="flex items-center justify-center gap-2">
            {set.completed ? (
              <Check className="w-[18px] h-[18px] text-[#22C55E] flex-shrink-0" />
            ) : (
              <Circle className="w-[18px] h-[18px] text-[#D1D5DB] flex-shrink-0" />
            )}
            <span
              className={`text-[15px] font-['DM_Sans'] ${
                set.completed ? 'text-[#22C55E] font-semibold' : 'text-[#1A1A1A] font-medium'
              }`}
            >
              {set.setNumber}
            </span>
          </div>
          <div className="flex items-center justify-center" onClick={(e) => e.stopPropagation()}>
            {!readOnly && editingReps === set.setNumber ? (
              <input
                type="number"
                value={editValue}
                onChange={(e) => setEditValue(e.target.value)}
                onBlur={() => {
                  const newReps = parseInt(editValue);
                  if (!isNaN(newReps) && newReps > 0) onUpdateReps?.(set.setNumber, newReps);
                  setEditingReps(null);
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    const newReps = parseInt(editValue);
                    if (!isNaN(newReps) && newReps > 0) onUpdateReps?.(set.setNumber, newReps);
                    setEditingReps(null);
                  }
                  if (e.key === 'Escape') setEditingReps(null);
                }}
                autoFocus
                className="w-12 text-center text-[15px] font-['DM_Sans'] font-medium bg-white border border-[#22C55E] rounded px-1 py-0.5 outline-none"
              />
            ) : (
              <span
                onClick={() => {
                  if (readOnly) return;
                  setEditingReps(set.setNumber);
                  setEditValue(set.reps.toString());
                }}
                className={`text-[15px] font-['DM_Sans'] ${!readOnly ? 'cursor-pointer hover:underline' : ''} ${
                  set.completed
                    ? 'text-[#22C55E] font-semibold'
                    : 'text-[#1A1A1A] font-medium'
                }`}
              >
                {set.reps}
              </span>
            )}
          </div>
          <div className="flex items-center justify-center" onClick={(e) => e.stopPropagation()}>
            {!readOnly && editingKg === set.setNumber ? (
              <input
                type="text"
                value={editValue}
                onChange={(e) => setEditValue(e.target.value)}
                onBlur={() => {
                  onUpdateKg?.(set.setNumber, editValue || '-');
                  setEditingKg(null);
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    onUpdateKg?.(set.setNumber, editValue || '-');
                    setEditingKg(null);
                  }
                  if (e.key === 'Escape') setEditingKg(null);
                }}
                autoFocus
                className="w-12 text-center text-[15px] font-['DM_Sans'] font-medium bg-white border border-[#22C55E] rounded px-1 py-0.5 outline-none"
              />
            ) : (
              <span
                onClick={() => {
                  if (readOnly) return;
                  setEditingKg(set.setNumber);
                  setEditValue(set.kg === '-' ? '' : set.kg);
                }}
                className={`text-[15px] font-['DM_Sans'] ${!readOnly ? 'cursor-pointer hover:underline' : ''} ${
                  set.completed
                    ? 'text-[#22C55E] font-semibold'
                    : set.kg === '-'
                    ? 'text-[#9CA3AF] font-medium'
                    : 'text-[#1A1A1A] font-medium'
                }`}
              >
                {set.kg}
              </span>
            )}
          </div>
        </div>
      ))}
    </div>
  );
};

// Format rest time helper
const formatRestTime = (seconds: number): string => {
  if (seconds >= 60) {
    return `${Math.floor(seconds / 60)}:${(seconds % 60).toString().padStart(2, '0')} min`;
  }
  return `${seconds} seg`;
};

// Exercise Card Component with cycling flip
const ExerciseCard: React.FC<{
  exercise: ExerciseData;
  onToggleInstructions: () => void;
  onToggleSet: (setNumber: number) => void;
  onUpdateReps: (setNumber: number, reps: number) => void;
  onUpdateKg: (setNumber: number, kg: string) => void;
  onToggleAltSet: (altIndex: number, setNumber: number) => void;
  onUpdateAltReps: (altIndex: number, setNumber: number, reps: number) => void;
  onUpdateAltKg: (altIndex: number, setNumber: number, kg: string) => void;
}> = ({ exercise, onToggleInstructions, onToggleSet, onUpdateReps, onUpdateKg, onToggleAltSet, onUpdateAltReps, onUpdateAltKg }) => {
  const alts = exercise.alternativeExercises || [];
  const hasAlternatives = alts.length > 0;
  const totalViews = 1 + alts.length; // original + alternatives

  // Auto-detect which exercise the user was working on (survives page reload).
  // If an alternative has completed sets but the primary doesn't, start on that alternative.
  // Only one exercise can have completed sets due to the anySetCompleted lock.
  // Deps: [exercise.id] is intentional — initialIndex only feeds useState (which ignores
  // subsequent changes), so recomputing on set changes would be wasted work.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const initialIndex = React.useMemo(() => {
    const primaryHasCompleted = exercise.sets.some((s) => s.completed);
    if (primaryHasCompleted || !hasAlternatives) return 0;
    const altIdx = alts.findIndex((alt) => alt.sets.some((s) => s.completed));
    return altIdx >= 0 ? altIdx + 1 : 0; // +1 because index 0 = primary
  }, [exercise.id]);

  const [flipCount, setFlipCount] = React.useState(initialIndex);
  const [isFlipping, setIsFlipping] = React.useState(false);
  const [showSwapTip, setShowSwapTip] = React.useState(false);

  // Suppress the CSS transition on mount when starting on an alternative
  // so the card doesn't animate a flip when the page loads.
  const [skipTransition, setSkipTransition] = React.useState(initialIndex > 0);
  React.useEffect(() => {
    if (initialIndex > 0) {
      const frameId = requestAnimationFrame(() => setSkipTransition(false));
      return () => cancelAnimationFrame(frameId);
    }
  }, [initialIndex]);

  // Lock swap once any set is completed (user committed to an exercise)
  const anySetCompleted = exercise.sets.some((s) => s.completed)
    || alts.some((alt) => alt.sets.some((s) => s.completed));
  const canSwap = hasAlternatives && !anySetCompleted;

  // Which view index is currently visible
  const currentIndex = flipCount % totalViews;
  // Is the front or back CSS face currently showing?
  const isFrontVisible = flipCount % 2 === 0;

  // Assign content indices to each CSS face
  // The visible face shows currentIndex, the hidden face pre-loads the next one
  const frontIndex = isFrontVisible ? currentIndex : (currentIndex + 1) % totalViews;
  const backIndex = !isFrontVisible ? currentIndex : (currentIndex + 1) % totalViews;

  const handleNextView = () => {
    if (isFlipping) return;

    // Show tooltip on first tap (if not seen before)
    const hasSeenTip = localStorage.getItem('gymbot-swap-tip-seen');
    if (!hasSeenTip && currentIndex === 0) {
      setShowSwapTip(true);
      localStorage.setItem('gymbot-swap-tip-seen', 'true');
      setTimeout(() => setShowSwapTip(false), 4000);
    }

    setIsFlipping(true);
    setFlipCount((prev) => prev + 1);
    setTimeout(() => setIsFlipping(false), 600);
  };

  // Get swap button label based on current view
  const getSwapLabel = (viewIdx: number): string => {
    if (viewIdx === 0) return 'Ver alternativa';
    if ((viewIdx + 1) % totalViews === 0) return 'Volver al original';
    return 'Siguiente alternativa';
  };

  // Render an alternative face
  const renderAlternativeFace = (alt: AlternativeExercise, isBack: boolean, viewIdx: number, isVisible: boolean) => (
    <div
      className={`flip-face ${isBack ? 'flip-back' : ''} bg-[#E8EDF3] border-l-4 border-[#6366F1] rounded-[20px] flex flex-col gap-4 w-full`}
      style={{
        padding: '24px',
        position: isVisible ? 'relative' : 'absolute',
      }}
    >
      {/* Header */}
      <div className="flex items-center justify-between w-full">
        <div className="flex items-center gap-3">
          <a
            href={alt.videoLink}
            target="_blank"
            rel="noopener noreferrer"
            className="relative w-11 h-11 min-w-[44px] min-h-[44px] rounded-xl flex items-center justify-center flex-shrink-0 cursor-pointer group transition-all duration-200 hover:scale-105 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6366F1] focus-visible:ring-offset-2"
            style={{ backgroundColor: '#6366F1' }}
            onClick={(e) => e.stopPropagation()}
          >
            <div
              className="absolute inset-0 rounded-xl opacity-0 group-hover:opacity-100 transition-opacity duration-200"
              style={{ boxShadow: '0 0 0 3px #6366F140, 0 0 12px #6366F160' }}
            />
            <Play className="w-4 h-4 text-white flex-shrink-0" fill="white" />
            <div className="absolute -top-1 -right-1 w-4 h-4 bg-white rounded-full flex items-center justify-center shadow-sm">
              <ExternalLink className="w-2.5 h-2.5 text-gray-600" />
            </div>
          </a>
          <div className="flex flex-col gap-0.5">
            <span className="text-[#6366F1] text-xs font-medium font-['DM_Sans']">
              Alternativa {viewIdx} de {alts.length}
            </span>
            <span className="text-[#1A1A1A] text-lg font-bold font-['Bricolage_Grotesque']">
              {alt.name}
            </span>
            {canSwap && (
              <div className="relative">
                <button
                  onClick={(e) => { e.stopPropagation(); handleNextView(); }}
                  className="flex items-center gap-1.5 mt-1 px-2 py-2.5 -ml-2 rounded-lg text-xs text-[#6B7280] hover:text-[#374151] hover:bg-[#E5E7EB]/50 active:bg-[#E5E7EB] transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6366F1] focus-visible:ring-offset-2"
                  aria-label={getSwapLabel(viewIdx)}
                >
                  <Repeat2 className="w-3.5 h-3.5" />
                  <span className="font-['DM_Sans'] font-medium">{getSwapLabel(viewIdx)}</span>
                </button>
                {showSwapTip && isVisible && (
                  <div className="absolute top-full left-0 mt-2 px-3 py-2 bg-[#1F2937] text-white text-xs rounded-lg shadow-lg flex items-center gap-2 max-w-[240px] z-10 animate-fadeIn">
                    <Info className="w-3.5 h-3.5 flex-shrink-0" />
                    <span className="font-['DM_Sans']">Al completar un set, te comprometes a este ejercicio</span>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
        {/* Chevron toggle */}
        <button onClick={onToggleInstructions} className="p-3 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6366F1] focus-visible:ring-offset-2 rounded-lg">
          {exercise.instructionsExpanded ? (
            <ChevronUp className="w-5 h-5 text-[#6B7280]" />
          ) : (
            <ChevronDown className="w-5 h-5 text-[#6B7280]" />
          )}
        </button>
      </div>

      {/* Instructions (Collapsible) */}
      {exercise.instructionsExpanded && (
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2 bg-[#FEF3C7] rounded-lg px-3 py-2">
            <span className="text-[#92400E] text-sm font-semibold font-['DM_Sans']">Esfuerzo:</span>
            <span className="text-[#92400E] text-sm font-normal font-['DM_Sans']">
              RIR: {alt.rir} (Deja {alt.rir} reps en reserva)
            </span>
          </div>
          {alt.restSeconds && alt.restSeconds > 0 && (
            <div className="flex items-center gap-2 bg-[#DBEAFE] rounded-lg px-3 py-2">
              <span className="text-[#1E40AF] text-sm font-semibold font-['DM_Sans']">Descanso:</span>
              <span className="text-[#1E40AF] text-sm font-normal font-['DM_Sans']">
                {formatRestTime(alt.restSeconds)} entre series
              </span>
            </div>
          )}
        </div>
      )}

      {/* Sets Table */}
      <SetsTable
        sets={alt.sets}
        onToggleSet={(setNumber) => onToggleAltSet(viewIdx - 1, setNumber)}
        onUpdateReps={(setNumber, reps) => onUpdateAltReps(viewIdx - 1, setNumber, reps)}
        onUpdateKg={(setNumber, kg) => onUpdateAltKg(viewIdx - 1, setNumber, kg)}
      />
    </div>
  );

  // Render the original exercise face
  const renderOriginalFace = (isBack: boolean, isVisible: boolean) => (
    <div
      className={`flip-face ${isBack ? 'flip-back' : ''} bg-[#EBEDF0] rounded-[20px] flex flex-col gap-4 w-full`}
      style={{
        padding: '24px',
        position: isVisible ? 'relative' : 'absolute',
      }}
    >
      {/* Exercise Header */}
      <div className="flex items-center justify-between w-full">
        <div className="flex items-center gap-3">
          {/* Exercise Badge */}
          {exercise.videoLink ? (
            <a
              href={exercise.videoLink}
              target="_blank"
              rel="noopener noreferrer"
              className="relative w-11 h-11 min-w-[44px] min-h-[44px] rounded-xl flex items-center justify-center flex-shrink-0 cursor-pointer group transition-all duration-200 hover:scale-105 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6366F1] focus-visible:ring-offset-2"
              style={{ backgroundColor: exercise.badgeColor }}
              onClick={(e) => e.stopPropagation()}
            >
              <div
                className="absolute inset-0 rounded-xl opacity-0 group-hover:opacity-100 transition-opacity duration-200"
                style={{
                  boxShadow: `0 0 0 3px ${exercise.badgeColor}40, 0 0 12px ${exercise.badgeColor}60`,
                }}
              />
              <Play className="w-4 h-4 text-white flex-shrink-0" fill="white" />
              <div className="absolute -top-1 -right-1 w-4 h-4 bg-white rounded-full flex items-center justify-center shadow-sm">
                <ExternalLink className="w-2.5 h-2.5 text-gray-600" />
              </div>
            </a>
          ) : (
            <div
              className="w-11 h-11 min-w-[44px] min-h-[44px] rounded-xl flex items-center justify-center flex-shrink-0"
              style={{ backgroundColor: exercise.badgeColor }}
            >
              <Play className="w-4 h-4 text-white flex-shrink-0" fill="white" />
            </div>
          )}
          {/* Exercise Info */}
          <div className="flex flex-col gap-0.5">
            <span className="text-[#9CA3AF] text-xs font-medium font-['DM_Sans']">Exercise</span>
            <span className="text-[#1A1A1A] text-lg font-bold font-['Bricolage_Grotesque']">
              {exercise.name}
            </span>
            {canSwap && (
              <div className="relative">
                <button
                  onClick={(e) => { e.stopPropagation(); handleNextView(); }}
                  className="flex items-center gap-1.5 mt-1 px-2 py-2.5 -ml-2 rounded-lg text-xs text-[#6B7280] hover:text-[#374151] hover:bg-[#E5E7EB]/50 active:bg-[#E5E7EB] transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6366F1] focus-visible:ring-offset-2"
                  aria-label="Ver ejercicio alternativo"
                >
                  <Repeat2 className="w-3.5 h-3.5" />
                  <span className="font-['DM_Sans'] font-medium">Ver alternativa</span>
                </button>
                {showSwapTip && isVisible && (
                  <div className="absolute top-full left-0 mt-2 px-3 py-2 bg-[#1F2937] text-white text-xs rounded-lg shadow-lg flex items-center gap-2 max-w-[240px] z-10 animate-fadeIn">
                    <Info className="w-3.5 h-3.5 flex-shrink-0" />
                    <span className="font-['DM_Sans']">Al completar un set, te comprometes a este ejercicio</span>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
        {/* Chevron toggle */}
        <button onClick={onToggleInstructions} className="p-3 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6366F1] focus-visible:ring-offset-2 rounded-lg">
          {exercise.instructionsExpanded ? (
            <ChevronUp className="w-5 h-5 text-[#6B7280]" />
          ) : (
            <ChevronDown className="w-5 h-5 text-[#6B7280]" />
          )}
        </button>
      </div>

      {/* Instructions Section (Collapsible) */}
      {exercise.instructionsExpanded && (
        <div className="flex flex-col gap-3 w-full">
          <span className="text-[#FF6B6B] text-sm font-semibold font-['DM_Sans']">
            Workout Instructions
          </span>
          <div className="flex flex-col gap-2">
            <div className="flex items-center gap-2 bg-[#FEF3C7] rounded-lg px-3 py-2">
              <span className="text-[#92400E] text-sm font-semibold font-['DM_Sans']">Esfuerzo:</span>
              <span className="text-[#92400E] text-sm font-normal font-['DM_Sans']">
                RIR: {exercise.rir} (Deja {exercise.rir} reps en reserva)
              </span>
            </div>
            {exercise.restSeconds && exercise.restSeconds > 0 && (
              <div className="flex items-center gap-2 bg-[#DBEAFE] rounded-lg px-3 py-2">
                <span className="text-[#1E40AF] text-sm font-semibold font-['DM_Sans']">Descanso:</span>
                <span className="text-[#1E40AF] text-sm font-normal font-['DM_Sans']">
                  {formatRestTime(exercise.restSeconds)} entre series
                </span>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Sets Table */}
      <SetsTable
        sets={exercise.sets}
        onToggleSet={onToggleSet}
        onUpdateReps={onUpdateReps}
        onUpdateKg={onUpdateKg}
      />
    </div>
  );

  // Render a face based on view index
  const renderFace = (viewIndex: number, isBack: boolean, isVisible: boolean) => {
    if (viewIndex === 0) return renderOriginalFace(isBack, isVisible);
    return renderAlternativeFace(alts[viewIndex - 1], isBack, viewIndex, isVisible);
  };

  // No alternatives — simple card, no flip
  if (!hasAlternatives) {
    return renderOriginalFace(false, true);
  }

  return (
    <div className="flip-container w-full">
      <div
        className="flip-card w-full"
        style={{
          transform: `rotateY(${flipCount * 180}deg)`,
          ...(skipTransition ? { transition: 'none' } : {}),
        }}
      >
        {renderFace(frontIndex, false, isFrontVisible)}
        {renderFace(backIndex, true, !isFrontVisible)}
      </div>
    </div>
  );
};

// Main Content Wrapper Component
export const WorkoutContent: React.FC<WorkoutContentProps> = ({
  exercises = defaultExercises,
  onComplete,
  onExercisesChange,
  isCompleted = false,
  isCompleting = false,
}) => {
  const [exerciseList, setExerciseList] = useState<ExerciseData[]>(exercises);
  const exerciseRefs = useRef<{ [key: string]: HTMLDivElement | null }>({});

  // Notify parent when exercises change
  useEffect(() => {
    onExercisesChange?.(exerciseList);
  }, [exerciseList, onExercisesChange]);

  const handleToggleInstructions = (exerciseId: string) => {
    setExerciseList((prev) =>
      prev.map((ex) =>
        ex.id === exerciseId
          ? { ...ex, instructionsExpanded: !ex.instructionsExpanded }
          : ex
      )
    );
  };

  // Save weights for a list of sets to the server
  const saveSetWeights = useCallback(async (sets: SetData[], fallbackPrefix: string) => {
    for (const set of sets) {
      if (set.kg && set.kg !== '-') {
        const setId = set.id || `${fallbackPrefix}-${set.setNumber}`;
        try {
          await updateSetWeight(setId, set.kg);
          console.log(`Weight saved for set ${setId}: ${set.kg}`);
        } catch (error) {
          console.error(`Failed to save weight for set ${setId}:`, error);
        }
      }
    }
  }, []);

  const handleToggleSet = (exerciseId: string, setNumber: number) => {
    setExerciseList((prev) => {
      let newList = prev.map((ex) =>
        ex.id === exerciseId
          ? {
              ...ex,
              sets: ex.sets.map((set) =>
                set.setNumber === setNumber
                  ? { ...set, completed: !set.completed }
                  : set
              ),
            }
          : ex
      );

      // Check if all sets of this exercise are now completed
      const currentExercise = newList.find((ex) => ex.id === exerciseId);
      const currentIndex = newList.findIndex((ex) => ex.id === exerciseId);

      if (currentExercise && currentExercise.sets.every((set) => set.completed)) {
        // Save all weights for this exercise to the server
        saveSetWeights(currentExercise.sets, currentExercise.id);

        // Collapse the completed exercise and expand the next one
        const nextExercise = newList[currentIndex + 1];
        newList = newList.map((ex) =>
          ex.id === exerciseId
            ? { ...ex, instructionsExpanded: false }
            : nextExercise && ex.id === nextExercise.id
            ? { ...ex, instructionsExpanded: true }
            : ex
        );

        // Scroll to next exercise
        if (nextExercise) {
          setTimeout(() => {
            const nextRef = exerciseRefs.current[nextExercise.id];
            if (nextRef) {
              nextRef.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }
          }, 300);
        }
      }

      return newList;
    });
  };

  const handleUpdateReps = (exerciseId: string, setNumber: number, reps: number) => {
    setExerciseList((prev) =>
      prev.map((ex) =>
        ex.id === exerciseId
          ? {
              ...ex,
              sets: ex.sets.map((set) =>
                set.setNumber === setNumber
                  ? { ...set, reps }
                  : set
              ),
            }
          : ex
      )
    );
  };

  const handleUpdateKg = (exerciseId: string, setNumber: number, kg: string) => {
    setExerciseList((prev) =>
      prev.map((ex) =>
        ex.id === exerciseId
          ? {
              ...ex,
              sets: ex.sets.map((s) =>
                s.setNumber === setNumber
                  ? { ...s, kg }
                  : s
              ),
            }
          : ex
      )
    );
  };

  const handleToggleAltSet = (exerciseId: string, altIndex: number, setNumber: number) => {
    setExerciseList((prev) => {
      let newList = prev.map((ex) =>
        ex.id === exerciseId && ex.alternativeExercises
          ? {
              ...ex,
              alternativeExercises: ex.alternativeExercises.map((alt, i) =>
                i === altIndex
                  ? { ...alt, sets: alt.sets.map((s) => s.setNumber === setNumber ? { ...s, completed: !s.completed } : s) }
                  : alt
              ),
            }
          : ex
      );

      // Check if all sets of the alternative are now completed
      const currentExercise = newList.find((ex) => ex.id === exerciseId);
      const currentIdx = newList.findIndex((ex) => ex.id === exerciseId);
      const currentAlt = currentExercise?.alternativeExercises?.[altIndex];

      if (currentAlt && currentAlt.sets.every((s) => s.completed)) {
        // Save all weights for this alternative to the server
        saveSetWeights(currentAlt.sets, currentExercise?.id || exerciseId);

        // Collapse current exercise and expand next
        const nextExercise = newList[currentIdx + 1];
        newList = newList.map((ex) =>
          ex.id === exerciseId
            ? { ...ex, instructionsExpanded: false }
            : nextExercise && ex.id === nextExercise.id
            ? { ...ex, instructionsExpanded: true }
            : ex
        );

        // Scroll to next exercise
        if (nextExercise) {
          setTimeout(() => {
            const nextRef = exerciseRefs.current[nextExercise.id];
            if (nextRef) {
              nextRef.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }
          }, 300);
        }
      }

      return newList;
    });
  };

  const handleUpdateAltReps = (exerciseId: string, altIndex: number, setNumber: number, reps: number) => {
    setExerciseList((prev) =>
      prev.map((ex) =>
        ex.id === exerciseId && ex.alternativeExercises
          ? {
              ...ex,
              alternativeExercises: ex.alternativeExercises.map((alt, i) =>
                i === altIndex
                  ? { ...alt, sets: alt.sets.map((s) => s.setNumber === setNumber ? { ...s, reps } : s) }
                  : alt
              ),
            }
          : ex
      )
    );
  };

  const handleUpdateAltKg = (exerciseId: string, altIndex: number, setNumber: number, kg: string) => {
    setExerciseList((prev) =>
      prev.map((ex) =>
        ex.id === exerciseId && ex.alternativeExercises
          ? {
              ...ex,
              alternativeExercises: ex.alternativeExercises.map((alt, i) =>
                i === altIndex
                  ? { ...alt, sets: alt.sets.map((s) => s.setNumber === setNumber ? { ...s, kg } : s) }
                  : alt
              ),
            }
          : ex
      )
    );
  };

  return (
    <div className="flex flex-col gap-6 w-full pb-8">
      {/* Exercise Cards */}
      <div className="flex flex-col gap-6 w-full">
        {exerciseList.map((exercise) => (
          <div
            key={exercise.id}
            ref={(el) => { exerciseRefs.current[exercise.id] = el; }}
          >
            <ExerciseCard
              exercise={exercise}
              onToggleInstructions={() => handleToggleInstructions(exercise.id)}
              onToggleSet={(setNumber) => handleToggleSet(exercise.id, setNumber)}
              onUpdateReps={(setNumber, reps) => handleUpdateReps(exercise.id, setNumber, reps)}
              onUpdateKg={(setNumber, kg) => handleUpdateKg(exercise.id, setNumber, kg)}
              onToggleAltSet={(altIndex, setNumber) => handleToggleAltSet(exercise.id, altIndex, setNumber)}
              onUpdateAltReps={(altIndex, setNumber, reps) => handleUpdateAltReps(exercise.id, altIndex, setNumber, reps)}
              onUpdateAltKg={(altIndex, setNumber, kg) => handleUpdateAltKg(exercise.id, altIndex, setNumber, kg)}
            />
          </div>
        ))}
      </div>

      {/* Complete Button */}
      <button
        onClick={onComplete}
        disabled={isCompleting || isCompleted}
        className={`flex items-center justify-center gap-2 w-full h-[52px] rounded-[26px] transition-colors ${
          isCompleted
            ? 'bg-[#86EFAC] cursor-default'
            : isCompleting
            ? 'bg-[#22C55E] opacity-70 cursor-wait'
            : 'bg-[#22C55E] hover:bg-[#16A34A]'
        }`}
      >
        {isCompleting ? (
          <>
            <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
            <span className="text-white text-base font-bold font-['DM_Sans']">
              Completando...
            </span>
          </>
        ) : isCompleted ? (
          <>
            <CheckCircle className="w-[22px] h-[22px] text-white" />
            <span className="text-white text-base font-bold font-['DM_Sans']">
              Rutina Completada
            </span>
          </>
        ) : (
          <>
            <CheckCircle className="w-[22px] h-[22px] text-white" />
            <span className="text-white text-base font-bold font-['DM_Sans']">
              Completar Rutina
            </span>
          </>
        )}
      </button>
    </div>
  );
};

export default WorkoutContent;
