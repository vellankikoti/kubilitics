import { motion } from 'framer-motion';

interface HealthRingProps {
    score: number;
    size?: number;
    strokeWidth?: number;
    showText?: boolean;
}

/** Maps score thresholds to design-system HSL token values */
function getScoreColor(s: number): string {
    if (s >= 80) return 'hsl(142, 71%, 45%)';   // --success
    if (s >= 50) return 'hsl(38, 92%, 50%)';    // --warning
    return 'hsl(0, 84%, 60%)';                   // --destructive
}

function getScoreGlow(s: number): string {
    if (s >= 80) return 'drop-shadow(0 0 8px hsl(142, 71%, 45%))';
    if (s >= 50) return 'drop-shadow(0 0 8px hsl(38, 92%, 50%))';
    return 'drop-shadow(0 0 8px hsl(0, 84%, 60%))';
}

export function HealthRing({
    score,
    size = 64,
    strokeWidth = 6,
    showText = true,
}: HealthRingProps) {
    const radius = (size - strokeWidth) / 2;
    const circumference = radius * 2 * Math.PI;
    const offset = circumference - (score / 100) * circumference;

    const color = getScoreColor(score);
    const glowFilter = getScoreGlow(score);

    return (
        <div className="relative flex items-center justify-center" style={{ width: size, height: size }}>
            <svg
                width={size}
                height={size}
                className="transform -rotate-90"
                style={{ filter: glowFilter }}
            >
                <circle
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    stroke="var(--chart-bg-track)"
                    strokeWidth={strokeWidth}
                    fill="none"
                />
                {/* Progress circle */}
                <motion.circle
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    stroke={color}
                    strokeWidth={strokeWidth}
                    strokeDasharray={circumference}
                    initial={{ strokeDashoffset: circumference }}
                    animate={{ strokeDashoffset: offset }}
                    transition={{ duration: 1.5, ease: "easeOut" }}
                    strokeLinecap="round"
                    fill="none"
                />
            </svg>
            {showText && (
                <motion.div
                    className="absolute inset-0 flex items-center justify-center flex-col leading-none"
                    initial={{ opacity: 0, scale: 0.8 }}
                    animate={{ opacity: 1, scale: 1 }}
                    transition={{ delay: 0.8, duration: 0.4, ease: 'easeOut' }}
                >
                    <span
                        className="font-bold tracking-tight tabular-nums text-foreground"
                        style={{ fontSize: `${Math.max(12, size * 0.28)}px` }}
                    >
                        {score}
                    </span>
                    <span
                        className="uppercase tracking-[0.1em] text-muted-foreground font-bold"
                        style={{ fontSize: `${Math.max(7, size * 0.08)}px`, marginTop: `${size * 0.05}px` }}
                    >
                        Score
                    </span>
                </motion.div>
            )}
        </div>
    );
}
