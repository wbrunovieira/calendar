/**
 * Calendar Loading Component
 * Modern loading state with animations
 */

export default function CalendarLoading() {
  return (
    <div className="w-full max-w-[1800px] mx-auto p-2 md:p-4 h-full flex items-center justify-center min-h-screen">
      <div className="flex flex-col items-center gap-6">
        {/* Main Calendar Icon with Pulse Animation */}
        <div className="relative">
          {/* Outer Pulse Ring */}
          <div className="absolute inset-0 rounded-full bg-gradient-to-br from-[#792990] to-[#350545] opacity-20 animate-ping" />

          {/* Middle Pulse Ring */}
          <div
            className="absolute inset-0 rounded-full bg-gradient-to-br from-[#792990] to-[#350545] opacity-30 animate-pulse"
            style={{ animationDelay: '0.2s' }}
          />

          {/* Calendar Icon Container */}
          <div className="relative w-24 h-24 rounded-2xl bg-gradient-to-br from-[#792990] to-[#350545] shadow-2xl flex items-center justify-center">
            <svg
              className="w-14 h-14 text-white animate-bounce"
              style={{ animationDuration: '1.5s' }}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
              />
            </svg>
          </div>
        </div>

        {/* Loading Text with Fade Animation */}
        <div className="flex flex-col items-center gap-3">
          <h2 className="text-white text-2xl font-semibold animate-pulse">Carregando calendário</h2>

          {/* Loading Dots */}
          <div className="flex gap-2">
            <div
              className="w-2 h-2 rounded-full bg-[#792990] animate-bounce"
              style={{ animationDelay: '0s' }}
            />
            <div
              className="w-2 h-2 rounded-full bg-[#792990] animate-bounce"
              style={{ animationDelay: '0.2s' }}
            />
            <div
              className="w-2 h-2 rounded-full bg-[#792990] animate-bounce"
              style={{ animationDelay: '0.4s' }}
            />
          </div>
        </div>

        {/* Progress Bar */}
        <div className="w-64 h-1 bg-white/10 rounded-full overflow-hidden">
          <div className="h-full bg-gradient-to-r from-[#792990] to-[#350545] animate-[loading_1.5s_ease-in-out_infinite]" />
        </div>
      </div>

      <style jsx>{`
        @keyframes loading {
          0% {
            width: 0%;
            margin-left: 0%;
          }
          50% {
            width: 75%;
            margin-left: 12.5%;
          }
          100% {
            width: 0%;
            margin-left: 100%;
          }
        }
      `}</style>
    </div>
  );
}
