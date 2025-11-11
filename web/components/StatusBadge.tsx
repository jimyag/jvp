interface StatusBadgeProps {
  status: string;
}

const statusColors: Record<string, string> = {
  running: "bg-green-100 text-green-800 border-green-300",
  stopped: "bg-gray-100 text-gray-800 border-gray-300",
  pending: "bg-yellow-100 text-yellow-800 border-yellow-300",
  error: "bg-red-100 text-red-800 border-red-300",
  available: "bg-blue-100 text-blue-800 border-blue-300",
  in_use: "bg-purple-100 text-purple-800 border-purple-300",
  active: "bg-green-100 text-green-800 border-green-300",
};

export default function StatusBadge({ status }: StatusBadgeProps) {
  const colorClass = statusColors[status.toLowerCase()] || statusColors.pending;

  return (
    <span
      className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium border ${colorClass}`}
    >
      {status.toUpperCase()}
    </span>
  );
}
