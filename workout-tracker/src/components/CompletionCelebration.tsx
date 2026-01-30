import { useEffect, useState } from 'react';

interface CompletionCelebrationProps {
  onDismiss: () => void;
}

const confettiEmojis = ['🎉', '💪', '🏋️', '⭐', '🔥', '✨', '🎊', '👏'];

export function CompletionCelebration({ onDismiss }: CompletionCelebrationProps) {
  const [confetti, setConfetti] = useState<Array<{ id: number; emoji: string; left: number; delay: number }>>([]);

  useEffect(() => {
    // Generate confetti pieces
    const pieces = Array.from({ length: 20 }, (_, i) => ({
      id: i,
      emoji: confettiEmojis[Math.floor(Math.random() * confettiEmojis.length)],
      left: Math.random() * 100,
      delay: Math.random() * 0.5,
    }));
    setConfetti(pieces);

    // Auto-dismiss after 5 seconds
    const timer = setTimeout(() => {
      onDismiss();
    }, 5000);

    return () => clearTimeout(timer);
  }, [onDismiss]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={onDismiss}
    >
      {/* Confetti */}
      {confetti.map((piece) => (
        <span
          key={piece.id}
          className="absolute text-3xl animate-fall pointer-events-none"
          style={{
            left: `${piece.left}%`,
            top: '-50px',
            animationDelay: `${piece.delay}s`,
          }}
        >
          {piece.emoji}
        </span>
      ))}

      {/* Message */}
      <div
        className="bg-white rounded-3xl text-center shadow-2xl animate-bounce-in"
        style={{ padding: '48px 64px', margin: '0 32px' }}
      >
        <div className="text-6xl mb-6">🎉</div>
        <h2 className="text-2xl font-bold font-['Bricolage_Grotesque'] text-[#1A1A1A] mb-3">
          ¡Felicidades!
        </h2>
        <p className="text-[#6B7280] font-['DM_Sans'] text-lg">
          Rutina completada con exito
        </p>
        <p className="text-sm text-[#9CA3AF] mt-6 font-['DM_Sans']">
          Toca para cerrar
        </p>
      </div>

      <style>{`
        @keyframes fall {
          0% {
            transform: translateY(0) rotate(0deg);
            opacity: 1;
          }
          100% {
            transform: translateY(100vh) rotate(720deg);
            opacity: 0;
          }
        }

        @keyframes bounce-in {
          0% {
            transform: scale(0.5);
            opacity: 0;
          }
          50% {
            transform: scale(1.1);
          }
          100% {
            transform: scale(1);
            opacity: 1;
          }
        }

        .animate-fall {
          animation: fall 5s ease-out forwards;
        }

        .animate-bounce-in {
          animation: bounce-in 0.5s ease-out forwards;
        }
      `}</style>
    </div>
  );
}
