import React, { useState, useRef } from 'react';
import { Check, ChevronDown, ChevronUp, MoreVertical, Play, CheckCircle, Circle } from 'lucide-react';

// Types
interface SetData {
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

interface ExerciseData {
  id: string;
  name: string;
  badgeColor: string;
  sets: SetData[];
  tips: Tip[];
  steps: Step[];
  instructionsExpanded?: boolean;
}

interface WorkoutContentProps {
  exercises?: ExerciseData[];
  onComplete?: () => void;
}

// Default exercise data matching the design
const defaultExercises: ExerciseData[] = [
  {
    id: '1',
    name: 'DB Front Squat',
    badgeColor: '#22C55E',
    instructionsExpanded: false,
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
  },
  {
    id: '2',
    name: 'Lunges',
    badgeColor: '#3B82F6',
    instructionsExpanded: true,
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
  },
  {
    id: '3',
    name: 'Plank Hold',
    badgeColor: '#8B5CF6',
    instructionsExpanded: true,
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

// Exercise Card Component
const ExerciseCard: React.FC<{
  exercise: ExerciseData;
  onToggleInstructions: () => void;
  onToggleSet: (setNumber: number) => void;
  onUpdateReps: (setNumber: number, reps: number) => void;
  onUpdateKg: (setNumber: number, kg: string) => void;
}> = ({ exercise, onToggleInstructions, onToggleSet, onUpdateReps, onUpdateKg }) => {
  const [editingReps, setEditingReps] = React.useState<number | null>(null);
  const [editingKg, setEditingKg] = React.useState<number | null>(null);
  const [editValue, setEditValue] = React.useState<string>('');
  const totalSets = exercise.sets.length;

  return (
    <div className="bg-[#F6F7F8] rounded-[20px] flex flex-col gap-4 w-full" style={{ padding: '24px' }}>
      {/* Exercise Header */}
      <div className="flex items-center justify-between w-full">
        <div className="flex items-center gap-3">
          {/* Exercise Badge */}
          <div
            className="w-9 h-9 rounded-xl flex items-center justify-center"
            style={{ backgroundColor: exercise.badgeColor }}
          >
            <Play className="w-4 h-4 text-white" fill="white" />
          </div>
          {/* Exercise Info */}
          <div className="flex flex-col gap-0.5">
            <span className="text-[#9CA3AF] text-xs font-medium font-['DM_Sans']">
              Exercise
            </span>
            <span className="text-[#1A1A1A] text-lg font-bold font-['Bricolage_Grotesque']">
              {exercise.name}
            </span>
          </div>
        </div>
        {/* Exercise Controls */}
        <div className="flex items-center gap-1">
          <button onClick={onToggleInstructions} className="p-1">
            {exercise.instructionsExpanded ? (
              <ChevronUp className="w-5 h-5 text-[#6B7280]" />
            ) : (
              <ChevronDown className="w-5 h-5 text-[#6B7280]" />
            )}
          </button>
          <button className="p-1">
            <MoreVertical className="w-5 h-5 text-[#9CA3AF]" />
          </button>
        </div>
      </div>

      {/* Instructions Section (Collapsible) */}
      {exercise.instructionsExpanded && (
        <div className="flex flex-col gap-3 w-full">
          {/* Instructions Header */}
          <span className="text-[#FF6B6B] text-sm font-semibold font-['DM_Sans']">
            Workout Instructions
          </span>

          {/* Tips Section */}
          <div className="flex flex-col gap-2 w-full">
            <span className="text-[#1A1A1A] text-sm font-semibold font-['DM_Sans']">
              Ten en cuenta:
            </span>
            <div className="flex flex-col gap-1 w-full">
              {exercise.tips.map((tip, index) => (
                <p
                  key={index}
                  className="text-[#6B7280] text-[13px] font-normal font-['DM_Sans'] leading-[1.5]"
                >
                  - {tip.text}
                </p>
              ))}
            </div>
          </div>

          {/* Steps Section */}
          <div className="flex flex-col gap-2 w-full">
            <span className="text-[#1A1A1A] text-sm font-semibold font-['DM_Sans']">
              Sigue el paso a paso:
            </span>
            <div className="flex flex-col gap-1 w-full">
              {exercise.steps.map((step, index) => (
                <p
                  key={index}
                  className="text-[#6B7280] text-[13px] font-normal font-['DM_Sans'] leading-[1.5]"
                >
                  - {step.text}
                </p>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Sets Table */}
      <div className="flex flex-col w-full mt-2">
        {/* Table Header */}
        <div className="grid grid-cols-3 py-3 border-b border-[#E5E7EB] w-full">
          <div className="flex items-center justify-center gap-1">
            <span className="text-[#1A1A1A] text-[15px] font-semibold font-['DM_Sans'] whitespace-nowrap">
              {totalSets} Sets
            </span>
            <ChevronDown className="w-3.5 h-3.5 text-[#6B7280] flex-shrink-0" />
          </div>
          <div className="flex items-center justify-center gap-1">
            <span className="text-[#1A1A1A] text-[15px] font-semibold font-['DM_Sans']">
              Reps
            </span>
            <ChevronDown className="w-3.5 h-3.5 text-[#6B7280] flex-shrink-0" />
          </div>
          <div className="flex items-center justify-center">
            <span className="text-[#1A1A1A] text-[15px] font-semibold font-['DM_Sans']">
              Kg
            </span>
          </div>
        </div>

        {/* Set Rows */}
        {exercise.sets.map((set) => (
          <div
            key={set.setNumber}
            onClick={() => onToggleSet(set.setNumber)}
            className={`grid grid-cols-3 py-3 border-b border-[#E5E7EB] last:border-b-0 cursor-pointer hover:bg-[#E8F5E9] transition-colors ${
              set.completed ? 'bg-[#F0FDF4]' : ''
            }`}
          >
            <div className="flex items-center justify-center gap-2">
              {set.completed ? (
                <Check className="w-[18px] h-[18px] text-[#22C55E] flex-shrink-0" />
              ) : (
                <Circle className="w-[18px] h-[18px] text-[#D1D5DB] flex-shrink-0" />
              )}
              <span
                className={`text-[15px] font-['DM_Sans'] ${
                  set.completed
                    ? 'text-[#22C55E] font-semibold'
                    : 'text-[#1A1A1A] font-medium'
                }`}
              >
                {set.setNumber}
              </span>
            </div>
            <div className="flex items-center justify-center" onClick={(e) => e.stopPropagation()}>
              {editingReps === set.setNumber ? (
                <input
                  type="number"
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                  onBlur={() => {
                    const newReps = parseInt(editValue);
                    if (!isNaN(newReps) && newReps > 0) {
                      onUpdateReps(set.setNumber, newReps);
                    }
                    setEditingReps(null);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      const newReps = parseInt(editValue);
                      if (!isNaN(newReps) && newReps > 0) {
                        onUpdateReps(set.setNumber, newReps);
                      }
                      setEditingReps(null);
                    }
                    if (e.key === 'Escape') {
                      setEditingReps(null);
                    }
                  }}
                  autoFocus
                  className="w-12 text-center text-[15px] font-['DM_Sans'] font-medium bg-white border border-[#22C55E] rounded px-1 py-0.5 outline-none"
                />
              ) : (
                <span
                  onClick={() => {
                    setEditingReps(set.setNumber);
                    setEditValue(set.reps.toString());
                  }}
                  className={`text-[15px] font-['DM_Sans'] cursor-pointer hover:underline ${
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
              {editingKg === set.setNumber ? (
                <input
                  type="text"
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                  onBlur={() => {
                    onUpdateKg(set.setNumber, editValue || '-');
                    setEditingKg(null);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      onUpdateKg(set.setNumber, editValue || '-');
                      setEditingKg(null);
                    }
                    if (e.key === 'Escape') {
                      setEditingKg(null);
                    }
                  }}
                  autoFocus
                  className="w-12 text-center text-[15px] font-['DM_Sans'] font-medium bg-white border border-[#22C55E] rounded px-1 py-0.5 outline-none"
                />
              ) : (
                <span
                  onClick={() => {
                    setEditingKg(set.setNumber);
                    setEditValue(set.kg === '-' ? '' : set.kg);
                  }}
                  className={`text-[15px] font-['DM_Sans'] cursor-pointer hover:underline ${
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
    </div>
  );
};

// Main Content Wrapper Component
export const WorkoutContent: React.FC<WorkoutContentProps> = ({
  exercises = defaultExercises,
  onComplete,
}) => {
  const [exerciseList, setExerciseList] = useState<ExerciseData[]>(exercises);
  const exerciseRefs = useRef<{ [key: string]: HTMLDivElement | null }>({});

  const handleToggleInstructions = (exerciseId: string) => {
    setExerciseList((prev) =>
      prev.map((ex) =>
        ex.id === exerciseId
          ? { ...ex, instructionsExpanded: !ex.instructionsExpanded }
          : ex
      )
    );
  };

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
        // Collapse the completed exercise
        newList = newList.map((ex) =>
          ex.id === exerciseId
            ? { ...ex, instructionsExpanded: false }
            : ex
        );

        // Find next exercise and scroll to it
        const nextExercise = newList[currentIndex + 1];
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
              sets: ex.sets.map((set) =>
                set.setNumber === setNumber
                  ? { ...set, kg }
                  : set
              ),
            }
          : ex
      )
    );
  };

  return (
    <div className="flex flex-col gap-6 w-full">
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
            />
          </div>
        ))}
      </div>

      {/* Complete Button */}
      <button
        onClick={onComplete}
        className="flex items-center justify-center gap-2 w-full h-[52px] bg-[#22C55E] rounded-[26px] hover:bg-[#16A34A] transition-colors"
      >
        <CheckCircle className="w-[22px] h-[22px] text-white" />
        <span className="text-white text-base font-bold font-['DM_Sans']">
          Completar Rutina
        </span>
      </button>
    </div>
  );
};

export default WorkoutContent;
