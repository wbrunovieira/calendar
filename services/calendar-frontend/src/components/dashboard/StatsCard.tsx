/**
 * Stats Card Component
 * Displays a single metric in a card format
 */

interface StatsCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon?: string;
  color?: 'blue' | 'green' | 'yellow' | 'orange' | 'purple' | 'red' | 'cyan';
}

const colorClasses = {
  blue: 'from-blue-500/20 to-blue-600/10 border-blue-500/30 shadow-blue-500/20',
  green: 'from-green-500/20 to-green-600/10 border-green-500/30 shadow-green-500/20',
  yellow: 'from-yellow-500/20 to-yellow-600/10 border-yellow-500/30 shadow-yellow-500/20',
  orange: 'from-orange-500/20 to-orange-600/10 border-orange-500/30 shadow-orange-500/20',
  purple: 'from-purple-500/20 to-purple-600/10 border-purple-500/30 shadow-purple-500/20',
  red: 'from-red-500/20 to-red-600/10 border-red-500/30 shadow-red-500/20',
  cyan: 'from-cyan-500/20 to-cyan-600/10 border-cyan-500/30 shadow-cyan-500/20',
};

export default function StatsCard({ title, value, subtitle, icon, color = 'blue' }: StatsCardProps) {
  return (
    <div className={`bg-gradient-to-br ${colorClasses[color]} backdrop-blur-sm rounded-xl p-6 border shadow-lg transition-all duration-300 hover:scale-105`}>
      <div className="flex items-start justify-between mb-2">
        <h3 className="text-sm font-medium text-white/70 uppercase tracking-wide">
          {title}
        </h3>
        {icon && <span className="text-2xl">{icon}</span>}
      </div>
      <div className="mt-2">
        <p className="text-3xl font-bold text-white">
          {value}
        </p>
        {subtitle && (
          <p className="text-sm text-white/60 mt-1">
            {subtitle}
          </p>
        )}
      </div>
    </div>
  );
}
