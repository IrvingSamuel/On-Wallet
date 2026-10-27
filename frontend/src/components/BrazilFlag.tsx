/** Bandeira do Brasil em SVG (proporção 10:7). Uso decorativo acadêmico — não é marca oficial. */
export default function BrazilFlag({ height = 24, className = "" }: { height?: number; className?: string }) {
  const width = Math.round((height * 10) / 7);
  return (
    <svg
      className={className}
      width={width}
      height={height}
      viewBox="0 0 20 14"
      role="img"
      aria-label="Bandeira do Brasil"
      xmlns="http://www.w3.org/2000/svg"
    >
      <rect width="20" height="14" fill="#46AD44" />
      <polygon points="10,1.4 18.2,7 10,12.6 1.8,7" fill="#FFCC29" />
      <circle cx="10" cy="7" r="3.2" fill="#2966B1" />
      <path
        d="M7.2 6.4c1.6-1.1 3.9-1.1 5.6 0"
        fill="none"
        stroke="#fff"
        strokeWidth="0.45"
        strokeLinecap="round"
      />
    </svg>
  );
}
