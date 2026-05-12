import { useState, useEffect, useCallback, useRef } from 'react';
import {
  Check,
  ChevronDown,
  ChevronUp,
  Play,
  ExternalLink,
  Repeat2,
  ChevronLeft,
  ChevronRight,
  Undo2,
  AlertCircle,
  CheckCircle2,
} from 'lucide-react';
import {
  getDraft,
  swapDraftExercise,
  approveDraft,
  type DraftData,
  type DraftDay,
  type DraftExercise,
  type DraftAlternative,
  type DraftResponse,
  type ApproveDraftResponse,
} from '../services/api';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatRest(seconds: number): string {
  if (seconds >= 60) {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${String(secs).padStart(2, '0')}`;
  }
  return `${seconds}s`;
}

// ---------------------------------------------------------------------------
// Types for local UI state
// ---------------------------------------------------------------------------

/** Per-exercise UI state: which alt is showing, whether instructions are open */
interface ExerciseUIState {
  showingAlternatives: boolean;
  alternativeIndex: number;
  instructionsExpanded: boolean;
}

type UIStateMap = Record<string, ExerciseUIState>;

function uiKey(dayNumber: number, exerciseOrder: number): string {
  return `${dayNumber}_${exerciseOrder}`;
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function VideoBadge({ videoLink, bgColor = '#374151' }: { videoLink?: string; bgColor?: string }) {
  if (videoLink) {
    return (
      <a
        href={videoLink}
        target="_blank"
        rel="noopener noreferrer"
        onClick={(e) => e.stopPropagation()}
        className="relative w-11 h-11 min-w-[44px] min-h-[44px] rounded-xl flex items-center justify-center shrink-0 cursor-pointer group transition-all duration-200 hover:scale-105 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6366F1] focus-visible:ring-offset-2 no-underline"
        style={{ backgroundColor: bgColor }}
      >
        <div
          className="absolute inset-0 rounded-xl opacity-0 group-hover:opacity-100 transition-opacity duration-200"
          style={{ boxShadow: `0 0 0 3px ${bgColor}40, 0 0 12px ${bgColor}60` }}
        />
        <Play className="w-4 h-4 text-white fill-white flex-shrink-0" />
        <span className="absolute -top-1 -right-1 w-4 h-4 bg-white rounded-full flex items-center justify-center shadow-sm">
          <ExternalLink className="w-2.5 h-2.5 text-gray-600" />
        </span>
      </a>
    );
  }

  return (
    <div
      className="relative w-11 h-11 min-w-[44px] min-h-[44px] rounded-xl flex items-center justify-center shrink-0"
      style={{ backgroundColor: bgColor }}
    >
      <Play className="w-4 h-4 text-white fill-white flex-shrink-0" />
    </div>
  );
}

function SpecRow({
  sets,
  reps,
  rir,
  restSeconds,
}: {
  sets: number;
  reps: string;
  rir: string;
  restSeconds: number;
}) {
  const isShortRest = restSeconds <= 60;
  return (
    <div
      className="bg-[#F9FAFB]/70 rounded-lg text-xs text-[#6B7280] flex flex-wrap"
      style={{ fontVariantNumeric: 'tabular-nums', padding: '10px', columnGap: '12px', rowGap: '4px' }}
    >
      <span>
        {sets} &times; {reps}
      </span>
      <span className="text-[#D1D5DB]">&middot;</span>
      <span>RIR {rir}</span>
      <span className="text-[#D1D5DB]">&middot;</span>
      <span className={isShortRest ? 'text-[#16A34A] font-medium' : ''}>
        Descanso {restSeconds}s
      </span>
    </div>
  );
}

function InstructionBanners({
  sets,
  reps,
  rir,
  restSeconds,
}: {
  sets: number;
  reps: string;
  rir: string;
  restSeconds: number;
}) {
  return (
    <div className="flex flex-col" style={{ gap: '8px' }}>
      <div className="flex items-center bg-[#F3F4F6] rounded-lg" style={{ gap: '8px', padding: '8px 12px' }}>
        <span className="text-[#374151] text-sm font-semibold font-['DM_Sans']">Volumen:</span>
        <span className="text-[#374151] text-sm font-normal font-['DM_Sans']">{sets} series &times; {reps} reps</span>
      </div>
      <div className="flex items-center bg-[#FEF3C7] rounded-lg" style={{ gap: '8px', padding: '8px 12px' }}>
        <span className="text-[#92400E] text-sm font-semibold font-['DM_Sans']">Esfuerzo:</span>
        <span className="text-[#92400E] text-sm font-normal font-['DM_Sans']">RIR {rir} (Deja {rir} reps en reserva)</span>
      </div>
      <div className="flex items-center bg-[#DBEAFE] rounded-lg" style={{ gap: '8px', padding: '8px 12px' }}>
        <span className="text-[#1E40AF] text-sm font-semibold font-['DM_Sans']">Descanso:</span>
        <span className="text-[#1E40AF] text-sm font-normal font-['DM_Sans']">{formatRest(restSeconds)} entre series</span>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Exercise card (primary view)
// ---------------------------------------------------------------------------

function PrimaryExerciseCard({
  exercise,
  uiState,
  onToggleInstructions,
  onShowAlternatives,
}: {
  exercise: DraftExercise;
  uiState: ExerciseUIState;
  onToggleInstructions: () => void;
  onShowAlternatives: () => void;
}) {
  const hasAlts = exercise.alternatives.length > 0;

  return (
    <div className="rounded-[20px] bg-[#EBEDF0] flex flex-col" style={{ padding: '24px', gap: '16px' }}>
      {/* Header row */}
      <div className="flex items-center justify-between">
        <div className="flex items-center flex-1 min-w-0" style={{ gap: '12px' }}>
          <VideoBadge videoLink={exercise.videoLink} />
          <div className="flex flex-col gap-0.5 min-w-0">
            <span className="text-xs text-[#9CA3AF] font-medium">{exercise.muscle}</span>
            <span className="font-['Bricolage_Grotesque'] text-lg font-bold text-[#1A1A1A] leading-tight">
              {exercise.name}
            </span>
            {hasAlts && (
              <button
                className="inline-flex items-center min-h-[44px] rounded-lg text-xs text-[#6B7280] font-medium bg-transparent border-none cursor-pointer font-['DM_Sans'] transition-all duration-150 hover:bg-black/5 hover:text-[#374151] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6366F1] focus-visible:ring-offset-2"
                style={{ gap: '6px', marginTop: '4px', padding: '10px 8px', marginLeft: '-8px' }}
                onClick={(e) => {
                  e.stopPropagation();
                  onShowAlternatives();
                }}
              >
                <Repeat2 className="w-3.5 h-3.5" />
                Ver alternativa
              </button>
            )}
          </div>
        </div>
        <button
          className="cursor-pointer bg-transparent border-none rounded-lg shrink-0 hover:bg-black/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6366F1] focus-visible:ring-offset-2"
          style={{ padding: '12px' }}
          onClick={onToggleInstructions}
          aria-label="Toggle instructions"
        >
          {uiState.instructionsExpanded ? (
            <ChevronUp className="w-5 h-5 text-[#6B7280]" />
          ) : (
            <ChevronDown className="w-5 h-5 text-[#6B7280]" />
          )}
        </button>
      </div>

      {/* Expanded view (banners) XOR collapsed view (chip) — single source of truth */}
      {uiState.instructionsExpanded ? (
        <InstructionBanners
          sets={exercise.sets}
          reps={exercise.reps}
          rir={exercise.rir}
          restSeconds={exercise.restSeconds}
        />
      ) : (
        <SpecRow
          sets={exercise.sets}
          reps={exercise.reps}
          rir={exercise.rir}
          restSeconds={exercise.restSeconds}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Alternative exercise card
// ---------------------------------------------------------------------------

function AlternativeExerciseCard({
  alternative,
  exercise,
  altIndex,
  totalAlts,
  onPrev,
  onNext,
  onPick,
  onBack,
}: {
  alternative: DraftAlternative;
  /** The primary exercise — we inherit sets/reps/rir/rest from it */
  exercise: DraftExercise;
  altIndex: number;
  totalAlts: number;
  onPrev: () => void;
  onNext: () => void;
  onPick: () => void;
  onBack: () => void;
}) {
  return (
    <div className="rounded-[20px] bg-[#E8EDF3] border-l-4 border-[#6366F1] flex flex-col" style={{ padding: '24px', gap: '16px' }}>
      {/* Alt label + context: which exercise is being replaced */}
      <div className="flex flex-col" style={{ gap: '2px' }}>
        <div className="text-xs text-[#6366F1] font-medium font-['DM_Sans']">
          Alternativa {altIndex + 1} de {totalAlts}
        </div>
        <div className="text-xs text-[#6B7280] font-['DM_Sans']">
          Reemplazaría a: <span className="text-[#374151] font-semibold">{exercise.name}</span>
        </div>
      </div>

      {/* Header row */}
      <div className="flex items-center justify-between">
        <div className="flex items-center flex-1 min-w-0" style={{ gap: '12px' }}>
          <VideoBadge videoLink={alternative.videoLink} bgColor="#6366F1" />
          <div className="flex flex-col gap-0.5 min-w-0">
            <span className="text-xs text-[#9CA3AF] font-medium">{alternative.muscle}</span>
            <span className="font-['Bricolage_Grotesque'] text-lg font-bold text-[#1A1A1A] leading-tight">
              {alternative.name}
            </span>
          </div>
        </div>
      </div>

      {/* Spec row — inherits from primary exercise */}
      <SpecRow
        sets={exercise.sets}
        reps={exercise.reps}
        rir={exercise.rir}
        restSeconds={exercise.restSeconds}
      />

      {/* Navigation */}
      <div className="flex items-center justify-between border-t border-[#6366F1]/20" style={{ gap: '8px', marginTop: '4px', paddingTop: '12px' }}>
        <button
          className="flex items-center rounded-lg text-xs font-medium cursor-pointer border-none font-['DM_Sans'] transition-all duration-150 bg-[#6366F1]/10 text-[#6366F1] hover:bg-[#6366F1]/20 disabled:opacity-30 disabled:cursor-default"
          style={{ gap: '4px', padding: '8px 12px' }}
          onClick={onPrev}
          disabled={altIndex === 0}
        >
          <ChevronLeft className="w-3 h-3" />
          Anterior
        </button>
        <span className="text-xs text-[#6B7280] font-medium">
          {altIndex + 1} de {totalAlts}
        </span>
        <button
          className="flex items-center rounded-lg text-xs font-medium cursor-pointer border-none font-['DM_Sans'] transition-all duration-150 bg-[#6366F1]/10 text-[#6366F1] hover:bg-[#6366F1]/20 disabled:opacity-30 disabled:cursor-default"
          style={{ gap: '4px', padding: '8px 12px' }}
          onClick={onNext}
          disabled={altIndex >= totalAlts - 1}
        >
          Siguiente
          <ChevronRight className="w-3 h-3" />
        </button>
      </div>

      {/* Pick / Back actions */}
      <div className="flex" style={{ gap: '8px' }}>
        <button
          className="flex items-center rounded-lg text-[13px] font-semibold cursor-pointer border-none font-['DM_Sans'] bg-[#22C55E] text-white transition-all duration-150 hover:bg-[#16A34A]"
          style={{ gap: '4px', padding: '10px 16px' }}
          onClick={onPick}
        >
          <Check className="w-3.5 h-3.5" />
          Elegir esta
        </button>
        <button
          className="flex items-center rounded-lg text-xs font-medium cursor-pointer border border-[#E5E7EB] font-['DM_Sans'] bg-white text-[#6B7280] transition-all duration-150 hover:bg-[#F9FAFB]"
          style={{ gap: '4px', padding: '8px 12px' }}
          onClick={onBack}
        >
          <Undo2 className="w-3 h-3" />
          Volver
        </button>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Success bottom sheet
// ---------------------------------------------------------------------------

export function SuccessOverlay({
  magicLinkUrl,
  onDismiss,
}: {
  magicLinkUrl?: string;
  onDismiss: () => void;
}) {
  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-[60] flex items-end justify-center">
      <div className="max-w-[400px] w-full bg-white rounded-t-3xl" style={{ paddingTop: '24px', paddingLeft: '24px', paddingRight: '24px', paddingBottom: 'max(env(safe-area-inset-bottom, 24px), 24px)' }}>
        {/* Drag handle */}
        <div className="w-10 h-1 bg-[#E5E7EB] rounded-full mx-auto mb-5" />

        {/* Icon */}
        <div className="w-20 h-20 rounded-full bg-[#DCFCE7] mx-auto mb-4 flex items-center justify-center">
          <CheckCircle2 className="w-10 h-10 text-[#22C55E]" />
        </div>

        {/* Title */}
        <h2 className="font-['Bricolage_Grotesque'] text-2xl font-bold text-[#1A1A1A] text-center">
          ¡Rutina aprobada!
        </h2>
        <p className="text-sm text-[#6B7280] text-center mt-2">
          Tu entrenador ya tiene tu rutina lista. ¡A entrenar!
        </p>

        {/* Next steps */}
        <div className="bg-[#F9FAFB] rounded-xl p-4 mt-5">
          <div className="text-[11px] font-semibold text-[#6B7280] uppercase tracking-wider mb-3">
            Que sigue
          </div>
          <div className="flex gap-2 mb-2 text-sm text-[#1A1A1A]">
            <span className="text-[#22C55E]">&rarr;</span>
            <span>Recibiras tu rutina diaria por WhatsApp</span>
          </div>
          <div className="flex gap-2 text-sm text-[#1A1A1A]">
            <span className="text-[#22C55E]">&rarr;</span>
            <span>Puedes pedir cambios en cualquier momento</span>
          </div>
        </div>

        {/* CTAs — magic link (when available) + dismiss; otherwise single dismiss */}
        {magicLinkUrl ? (
          <>
            <a
              href={magicLinkUrl}
              className="block w-full bg-[#22C55E] text-white text-center no-underline rounded-xl text-base font-semibold font-['DM_Sans'] transition-all duration-150 hover:bg-[#16A34A] active:scale-[0.97]"
              style={{ padding: '14px', marginTop: '20px' }}
            >
              Ver mi rutina de hoy
            </a>
            <button
              className="w-full bg-transparent text-[#6B7280] border-none text-sm font-medium font-['DM_Sans'] cursor-pointer transition-all duration-150 hover:text-[#374151]"
              style={{ padding: '10px', marginTop: '8px' }}
              onClick={onDismiss}
            >
              Entendido
            </button>
          </>
        ) : (
          <button
            className="w-full bg-[#22C55E] text-white border-none rounded-xl text-base font-semibold font-['DM_Sans'] cursor-pointer transition-all duration-150 hover:bg-[#16A34A] active:scale-[0.97]"
            style={{ padding: '14px', marginTop: '20px' }}
            onClick={onDismiss}
          >
            Entendido
          </button>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Demo mock data (for ?demo mode)
// ---------------------------------------------------------------------------

const DEMO_DRAFT: DraftData = {
  code: 'demo',
  status: 'pending',
  goal: 'Ganar masa muscular',
  level: 'Intermedio',
  daysPerWeek: 3,
  scheduleLabel: 'Cuerpo Completo (Full Body)',
  expiresAt: new Date(Date.now() + 86400000).toISOString(),
  days: [
    {
      dayNumber: 1, title: 'Full Body A',
      exercises: [
        { exerciseId: 'ex_hack_squat', exerciseOrder: 1, name: 'Sentadilla Hack', muscle: 'Quads', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 90, videoLink: 'https://youtube.com',
          alternatives: [
            { exerciseId: 'ex_leg_press', name: 'Prensa de piernas', muscle: 'Quads', videoLink: 'https://youtube.com' },
            { exerciseId: 'ex_goblet', name: 'Goblet Squat', muscle: 'Quads', videoLink: 'https://youtube.com' },
            { exerciseId: 'ex_bulgarian', name: 'Sentadilla Bulgara', muscle: 'Quads', videoLink: '' },
            { exerciseId: 'ex_smith_sq', name: 'Sentadilla en Smith', muscle: 'Quads', videoLink: 'https://youtube.com' },
          ]},
        { exerciseId: 'ex_incline_db', exerciseOrder: 2, name: 'Press inclinado con mancuerna', muscle: 'Pecho', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 90, videoLink: 'https://youtube.com',
          alternatives: [
            { exerciseId: 'ex_chest_m', name: 'Maquina de pecho', muscle: 'Pecho', videoLink: 'https://youtube.com' },
            { exerciseId: 'ex_fly', name: 'Aperturas con mancuerna', muscle: 'Pecho', videoLink: 'https://youtube.com' },
            { exerciseId: 'ex_bench', name: 'Press plano con barra', muscle: 'Pecho', videoLink: 'https://youtube.com' },
          ]},
        { exerciseId: 'ex_t_row', exerciseOrder: 3, name: 'Remo en T', muscle: 'Espalda', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 90, videoLink: 'https://youtube.com',
          alternatives: [
            { exerciseId: 'ex_db_row', name: 'Remo con mancuerna', muscle: 'Espalda', videoLink: 'https://youtube.com' },
            { exerciseId: 'ex_machine_row', name: 'Remo en maquina', muscle: 'Espalda', videoLink: 'https://youtube.com' },
          ]},
        { exerciseId: 'ex_core4', exerciseOrder: 4, name: 'Estabilidad del Nucleo 4', muscle: 'Core', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 60, videoLink: '',
          alternatives: [
            { exerciseId: 'ex_plank', name: 'Plancha', muscle: 'Core', videoLink: '' },
            { exerciseId: 'ex_deadbug', name: 'Dead Bug', muscle: 'Core', videoLink: '' },
          ]},
      ],
    },
    {
      dayNumber: 2, title: 'Full Body B',
      exercises: [
        { exerciseId: 'ex_rdl', exerciseOrder: 1, name: 'Peso muerto / RDL', muscle: 'Isquiotibiales', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 90, videoLink: 'https://youtube.com',
          alternatives: [{ exerciseId: 'ex_curl_fem', name: 'Curl femoral acostado', muscle: 'Isquiotibiales', videoLink: 'https://youtube.com' }] },
        { exerciseId: 'ex_lat_open', exerciseOrder: 2, name: 'Jalon abierto', muscle: 'Espalda', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 90, videoLink: 'https://youtube.com',
          alternatives: [{ exerciseId: 'ex_lat_close', name: 'Jalon al pecho', muscle: 'Espalda', videoLink: 'https://youtube.com' }] },
        { exerciseId: 'ex_mil_press', exerciseOrder: 3, name: 'Press militar con mancuerna', muscle: 'Hombros', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 90, videoLink: 'https://youtube.com',
          alternatives: [{ exerciseId: 'ex_lat_raise', name: 'Elevaciones laterales', muscle: 'Hombros', videoLink: 'https://youtube.com' }] },
        { exerciseId: 'ex_bicep_m', exerciseOrder: 4, name: 'Maquina de biceps', muscle: 'Biceps', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 90, videoLink: 'https://youtube.com',
          alternatives: [{ exerciseId: 'ex_curl_db', name: 'Curl con mancuerna', muscle: 'Biceps', videoLink: 'https://youtube.com' }] },
        { exerciseId: 'ex_bench_act', exerciseOrder: 5, name: 'Levantamiento Activo de Banca', muscle: 'Core', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 60, videoLink: '',
          alternatives: [{ exerciseId: 'ex_pallof', name: 'Pallof Press', muscle: 'Core', videoLink: '' }] },
      ],
    },
    {
      dayNumber: 3, title: 'Full Body C',
      exercises: [
        { exerciseId: 'ex_lunge', exerciseOrder: 1, name: 'Zancadas con barra', muscle: 'Quads / Gluteo', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 90, videoLink: 'https://youtube.com',
          alternatives: [{ exerciseId: 'ex_stepup', name: 'Step Up', muscle: 'Quads / Gluteo', videoLink: 'https://youtube.com' }] },
        { exerciseId: 'ex_seated_curl', exerciseOrder: 2, name: 'Femoral sentado', muscle: 'Isquiotibiales', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 90, videoLink: 'https://youtube.com',
          alternatives: [] },
        { exerciseId: 'ex_chest_p', exerciseOrder: 3, name: 'Maquina de pecho / Chest press', muscle: 'Pecho', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 90, videoLink: 'https://youtube.com',
          alternatives: [{ exerciseId: 'ex_dip_m', name: 'Fondos en maquina', muscle: 'Pecho', videoLink: 'https://youtube.com' }] },
        { exerciseId: 'ex_tri_lat', exerciseOrder: 4, name: 'Jalon con el triangulo', muscle: 'Espalda', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 90, videoLink: 'https://youtube.com',
          alternatives: [] },
        { exerciseId: 'ex_tri_pushdown', exerciseOrder: 5, name: 'Polea de triceps', muscle: 'Triceps', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 90, videoLink: 'https://youtube.com',
          alternatives: [{ exerciseId: 'ex_bench_dip', name: 'Fondos en banco', muscle: 'Triceps', videoLink: '' }] },
        { exerciseId: 'ex_static_bench', exerciseOrder: 6, name: 'Levantamiento Estatico en Banco', muscle: 'Core', sets: 3, reps: '12-15', rir: '1-2', restSeconds: 60, videoLink: '',
          alternatives: [{ exerciseId: 'ex_side_plank', name: 'Plancha lateral', muscle: 'Core', videoLink: '' }] },
      ],
    },
  ],
};

// ---------------------------------------------------------------------------
// Main page component
// ---------------------------------------------------------------------------

export function DraftPreviewPage() {
  // --- Data state ---
  const [draft, setDraft] = useState<DraftData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isExpired, setIsExpired] = useState(false);

  // --- UI state ---
  const [activeDay, setActiveDay] = useState(0);
  const [uiStates, setUIStates] = useState<UIStateMap>({});
  const [isApproving, setIsApproving] = useState(false);
  const [approved, setApproved] = useState(false);
  const [showSuccess, setShowSuccess] = useState(false);
  const [approveResponse, setApproveResponse] = useState<ApproveDraftResponse | null>(null);

  const tabsRef = useRef<HTMLDivElement>(null);

  // --- Read code from URL ---
  const params = new URLSearchParams(window.location.search);
  const code = params.get('c') ?? '';
  const isDemo = params.has('demo');

  // --- Fetch draft ---
  const fetchDraft = useCallback(async () => {
    if (!code && !isDemo) {
      setError('No se encontro codigo de acceso');
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const response: DraftResponse = isDemo
        ? { success: true, data: DEMO_DRAFT }
        : await getDraft(code);

      if (!response.success || !response.data) {
        const errCode = response.error?.code;
        if (errCode === 410) {
          setIsExpired(true);
        }
        setError(response.error?.message ?? 'Error al cargar el borrador');
        return;
      }

      const data = response.data;
      setDraft(data);

      if (data.status === 'approved') {
        setApproved(true);
      }

      // Initialize UI states: first exercise of each day expanded
      const initialUI: UIStateMap = {};
      for (const day of data.days) {
        for (let i = 0; i < day.exercises.length; i++) {
          const key = uiKey(day.dayNumber, day.exercises[i].exerciseOrder);
          initialUI[key] = {
            showingAlternatives: false,
            alternativeIndex: 0,
            instructionsExpanded: i === 0,
          };
        }
      }
      setUIStates(initialUI);
    } catch (err) {
      setError('Error de conexion. Intenta de nuevo.');
      console.error('Draft fetch error:', err);
    } finally {
      setLoading(false);
    }
  }, [code, isDemo]);

  useEffect(() => {
    fetchDraft();
  }, [fetchDraft]);

  // --- UI state helpers ---
  const getUIState = (dayNum: number, order: number): ExerciseUIState => {
    return (
      uiStates[uiKey(dayNum, order)] ?? {
        showingAlternatives: false,
        alternativeIndex: 0,
        instructionsExpanded: false,
      }
    );
  };

  const updateUIState = (dayNum: number, order: number, patch: Partial<ExerciseUIState>) => {
    setUIStates((prev) => ({
      ...prev,
      [uiKey(dayNum, order)]: { ...getUIState(dayNum, order), ...patch },
    }));
  };

  // --- Swap handler (optimistic) ---
  const handleSwap = (dayNumber: number, exerciseOrder: number, altIndex: number) => {
    if (!draft) return;

    // Find the day and exercise
    const dayIdx = draft.days.findIndex((d) => d.dayNumber === dayNumber);
    if (dayIdx === -1) return;
    const exIdx = draft.days[dayIdx].exercises.findIndex(
      (e) => e.exerciseOrder === exerciseOrder,
    );
    if (exIdx === -1) return;

    const exercise = draft.days[dayIdx].exercises[exIdx];
    const alt = exercise.alternatives[altIndex];
    if (!alt) return;

    // Build the old exercise as a new alternative entry
    const oldAsAlt: DraftAlternative = {
      exerciseId: exercise.exerciseId,
      name: exercise.name,
      muscle: exercise.muscle,
      videoLink: exercise.videoLink,
    };

    // Build new alternatives list: remove picked, add old
    const newAlts = [
      ...exercise.alternatives.slice(0, altIndex),
      ...exercise.alternatives.slice(altIndex + 1),
      oldAsAlt,
    ];

    // Build updated exercise
    const updatedExercise: DraftExercise = {
      ...exercise,
      exerciseId: alt.exerciseId,
      name: alt.name,
      muscle: alt.muscle,
      videoLink: alt.videoLink,
      alternatives: newAlts,
    };

    // Optimistic update
    const newDays = [...draft.days];
    const newExercises = [...newDays[dayIdx].exercises];
    newExercises[exIdx] = updatedExercise;
    newDays[dayIdx] = { ...newDays[dayIdx], exercises: newExercises };
    setDraft({ ...draft, days: newDays });

    // Close alternatives view
    updateUIState(dayNumber, exerciseOrder, {
      showingAlternatives: false,
      alternativeIndex: 0,
    });

    // Fire-and-forget PATCH
    swapDraftExercise(code, dayNumber, exerciseOrder, alt.exerciseId).catch((err) => {
      console.error('Swap failed (UI already updated):', err);
    });
  };

  // --- Approve handler ---
  const handleApprove = async () => {
    if (isApproving || approved || !draft) return;

    setIsApproving(true);
    try {
      if (!isDemo) {
        const resp = await approveDraft(code);
        setApproveResponse(resp);
      } else {
        // Simulate API delay in demo mode
        await new Promise(r => setTimeout(r, 800));
      }
      setApproved(true);
      setShowSuccess(true);
      document.body.style.overflow = 'hidden';
      // Haptic feedback on mobile
      if (navigator.vibrate) navigator.vibrate(200);
    } catch (err) {
      console.error('Approve failed:', err);
      alert('Error al aprobar la rutina. Intenta de nuevo.');
    } finally {
      setIsApproving(false);
    }
  };

  const handleDismissSuccess = () => {
    setShowSuccess(false);
    document.body.style.overflow = '';
  };

  const handleRequestChanges = () => {
    // Open WhatsApp with a pre-filled message
    const waUrl = 'https://wa.me/573506267523?text=Quiero%20hacer%20cambios%20en%20mi%20rutina';
    window.open(waUrl, '_blank');
  };

  // --- Loading ---
  if (loading) {
    return (
      <div className="min-h-screen bg-white flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-[#22C55E] mx-auto mb-4" />
          <p className="text-[#6B7280] font-['DM_Sans']">Cargando borrador...</p>
        </div>
      </div>
    );
  }

  // --- No code ---
  if (!code && !isDemo) {
    return (
      <div className="min-h-screen bg-white flex items-center justify-center">
        <div className="text-center px-6">
          <div className="w-16 h-16 bg-[#FEE2E2] rounded-full flex items-center justify-center mx-auto mb-4">
            <AlertCircle className="w-8 h-8 text-[#DC2626]" />
          </div>
          <h2 className="text-xl font-bold font-['Bricolage_Grotesque'] text-[#1A1A1A] mb-2">
            Sin codigo de acceso
          </h2>
          <p className="text-[#6B7280] font-['DM_Sans']">
            No se encontro codigo de acceso. Usa el link que recibiste por WhatsApp.
          </p>
        </div>
      </div>
    );
  }

  // --- Expired ---
  if (isExpired) {
    return (
      <div className="min-h-screen bg-white flex items-center justify-center">
        <div className="text-center px-6">
          <div className="w-16 h-16 bg-[#FEF3C7] rounded-full flex items-center justify-center mx-auto mb-4">
            <AlertCircle className="w-8 h-8 text-[#D97706]" />
          </div>
          <h2 className="text-xl font-bold font-['Bricolage_Grotesque'] text-[#1A1A1A] mb-2">
            Borrador expirado
          </h2>
          <p className="text-[#6B7280] font-['DM_Sans']">
            Este borrador ha expirado. Solicita uno nuevo por WhatsApp.
          </p>
        </div>
      </div>
    );
  }

  // --- Error ---
  if (error || !draft) {
    return (
      <div className="min-h-screen bg-white flex items-center justify-center">
        <div className="text-center px-6">
          <div className="w-16 h-16 bg-[#FEE2E2] rounded-full flex items-center justify-center mx-auto mb-4">
            <AlertCircle className="w-8 h-8 text-[#DC2626]" />
          </div>
          <h2 className="text-xl font-bold font-['Bricolage_Grotesque'] text-[#1A1A1A] mb-2">
            Error
          </h2>
          <p className="text-[#6B7280] font-['DM_Sans'] mb-4">
            {error ?? 'Error desconocido'}
          </p>
          <button
            className="bg-[#22C55E] text-white border-none rounded-xl py-3 px-6 text-sm font-semibold font-['DM_Sans'] cursor-pointer"
            onClick={fetchDraft}
          >
            Reintentar
          </button>
        </div>
      </div>
    );
  }

  // --- Main render ---
  const currentDay: DraftDay | undefined = draft.days[activeDay];

  return (
    <div className="min-h-screen bg-[#F3F4F6] flex justify-center">
      <div className="max-w-[400px] w-full bg-white min-h-screen relative">
        {/* ---------- Header ---------- */}
        <div className="border-b border-[#F3F4F6]" style={{ paddingTop: 24, paddingLeft: 20, paddingRight: 20, paddingBottom: 16 }}>
          {/* Badge */}
          <div className="inline-flex items-center gap-1 bg-[#FEF9C3] text-[#854D0E] text-[11px] font-semibold py-0.5 px-2.5 rounded-full mb-2">
            {approved ? (
              <>
                <Check className="w-3 h-3" /> APROBADA
              </>
            ) : (
              <>&#9670; BORRADOR</>
            )}
          </div>

          {/* Title — status already communicated by the pill above */}
          <h1 className="font-['Bricolage_Grotesque'] text-2xl font-bold text-[#1A1A1A]">
            Tu Rutina
          </h1>

          {/* Pills */}
          <div className="flex gap-2 mt-3">
            <span className="bg-[#DCFCE7] text-[#166534] text-xs font-medium py-1 px-3 rounded-full">
              {draft.goal}
            </span>
            <span className="bg-[#DBEAFE] text-[#1E40AF] text-xs font-medium py-1 px-3 rounded-full">
              {draft.level}
            </span>
          </div>

          {/* Summary */}
          <p className="text-xs text-[#6B7280] mt-2">
            {draft.daysPerWeek} días &middot; {draft.scheduleLabel}
          </p>
        </div>

        {/* ---------- Day Tabs ---------- */}
        <div
          ref={tabsRef}
          className="flex gap-2 border-b border-[#F3F4F6] sticky top-0 bg-white z-10 overflow-x-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]"
          style={{ padding: '12px 20px' }}
        >
          {draft.days.map((day, idx) => (
            <button
              key={day.dayNumber}
              style={{ padding: '6px 16px' }}
              className={`rounded-full text-sm font-semibold cursor-pointer border-none font-['DM_Sans'] shrink-0 transition-all duration-150 ${
                idx === activeDay
                  ? 'bg-[#22C55E] text-white shadow-sm shadow-[#22C55E]/30'
                  : 'bg-[#F9FAFB] text-[#6B7280] border border-[#E5E7EB] hover:bg-[#F3F4F6] hover:text-[#374151]'
              }`}
              onClick={() => setActiveDay(idx)}
            >
              Día {day.dayNumber}
            </button>
          ))}
        </div>

        {/* ---------- Content ---------- */}
        <div className="relative" style={{ paddingTop: 16, paddingLeft: 20, paddingRight: 20, paddingBottom: 180 }}>
          {currentDay && (
            <>
              {/* Day title */}
              <h2 className="font-['Bricolage_Grotesque'] text-lg font-bold text-[#1A1A1A] mb-4">
                {currentDay.title}
              </h2>

              {/* Exercise cards */}
              <div className="flex flex-col gap-4">
                {currentDay.exercises.map((exercise) => {
                  const ui = getUIState(currentDay.dayNumber, exercise.exerciseOrder);

                  if (ui.showingAlternatives && exercise.alternatives.length > 0) {
                    const alt = exercise.alternatives[ui.alternativeIndex];
                    if (!alt) return null;

                    return (
                      <AlternativeExerciseCard
                        key={`alt-${exercise.exerciseOrder}`}
                        alternative={alt}
                        exercise={exercise}
                        altIndex={ui.alternativeIndex}
                        totalAlts={exercise.alternatives.length}
                        onPrev={() =>
                          updateUIState(currentDay.dayNumber, exercise.exerciseOrder, {
                            alternativeIndex: Math.max(0, ui.alternativeIndex - 1),
                          })
                        }
                        onNext={() =>
                          updateUIState(currentDay.dayNumber, exercise.exerciseOrder, {
                            alternativeIndex: Math.min(
                              exercise.alternatives.length - 1,
                              ui.alternativeIndex + 1,
                            ),
                          })
                        }
                        onPick={() =>
                          handleSwap(
                            currentDay.dayNumber,
                            exercise.exerciseOrder,
                            ui.alternativeIndex,
                          )
                        }
                        onBack={() =>
                          updateUIState(currentDay.dayNumber, exercise.exerciseOrder, {
                            showingAlternatives: false,
                            alternativeIndex: 0,
                          })
                        }
                      />
                    );
                  }

                  return (
                    <PrimaryExerciseCard
                      key={`pri-${exercise.exerciseOrder}`}
                      exercise={exercise}
                      uiState={ui}
                      onToggleInstructions={() =>
                        updateUIState(currentDay.dayNumber, exercise.exerciseOrder, {
                          instructionsExpanded: !ui.instructionsExpanded,
                        })
                      }
                      onShowAlternatives={() =>
                        updateUIState(currentDay.dayNumber, exercise.exerciseOrder, {
                          showingAlternatives: true,
                          alternativeIndex: 0,
                        })
                      }
                    />
                  );
                })}
              </div>
            </>
          )}
        </div>

        {/* Scroll fade hint */}
        <div className="fixed bottom-[140px] left-1/2 -translate-x-1/2 max-w-[400px] w-full h-8 bg-gradient-to-b from-transparent to-white z-40 pointer-events-none" />

        {/* ---------- Sticky Bottom Bar ---------- */}
        <div className="fixed bottom-0 left-1/2 -translate-x-1/2 max-w-[400px] w-full bg-white border-t border-[#F3F4F6] shadow-[0_-4px_16px_rgba(0,0,0,0.06)] z-50" style={{ paddingTop: '16px', paddingLeft: '20px', paddingRight: '20px', paddingBottom: 'max(env(safe-area-inset-bottom, 16px), 16px)' }}>
          {approved ? (
            <div className="text-center text-sm text-[#6B7280] font-['DM_Sans'] py-2">
              Rutina aprobada — revisa tu WhatsApp
            </div>
          ) : (
            <>
              <button
                className="w-full bg-[#22C55E] text-white border-none rounded-xl text-base font-semibold font-['DM_Sans'] cursor-pointer flex items-center justify-center active:scale-[0.97] transition-all duration-150 hover:bg-[#16A34A] disabled:opacity-60 disabled:cursor-default"
                style={{ padding: '14px', gap: '8px' }}
                onClick={handleApprove}
                disabled={isApproving}
              >
                {isApproving ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white" />
                    Aprobando...
                  </>
                ) : (
                  <>
                    <Check className="w-[18px] h-[18px]" />
                    Aprobar rutina
                  </>
                )}
              </button>
              <button
                className="w-full bg-transparent text-[#1A1A1A] border-[1.5px] border-[#E5E7EB] rounded-xl text-sm font-medium font-['DM_Sans'] cursor-pointer transition-all duration-150 hover:bg-[#F9FAFB] hover:border-[#D1D5DB]"
                style={{ padding: '12px', marginTop: '8px' }}
                onClick={handleRequestChanges}
              >
                Quiero hacer cambios
              </button>
              <p className="text-[11px] text-[#9CA3AF] text-center mt-2">
                Los cambios los puedes pedir por WhatsApp
              </p>
            </>
          )}
        </div>

        {/* ---------- Success Overlay ---------- */}
        {showSuccess && (
          <SuccessOverlay
            magicLinkUrl={approveResponse?.magicLinkUrl}
            onDismiss={handleDismissSuccess}
          />
        )}
      </div>
    </div>
  );
}
