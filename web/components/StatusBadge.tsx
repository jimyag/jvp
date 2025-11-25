interface StatusBadgeProps {
  status: string;
  color?: string;
  text?: string;
}

const statusColors: Record<string, string> = {
  running: "bg-green-100 text-green-800 border-green-300",
  stopped: "bg-gray-100 text-gray-800 border-gray-300",
  pending: "bg-yellow-100 text-yellow-800 border-yellow-300",
  error: "bg-red-100 text-red-800 border-red-300",
  available: "bg-blue-100 text-blue-800 border-blue-300",
  in_use: "bg-purple-100 text-purple-800 border-purple-300",
  active: "bg-green-100 text-green-800 border-green-300",
  green: "bg-green-100 text-green-800 border-green-300",
  red: "bg-red-100 text-red-800 border-red-300",
  yellow: "bg-yellow-100 text-yellow-800 border-yellow-300",
  blue: "bg-blue-100 text-blue-800 border-blue-300",
  purple: "bg-purple-100 text-purple-800 border-purple-300",
  orange: "bg-orange-100 text-orange-800 border-orange-300",
  indigo: "bg-indigo-100 text-indigo-800 border-indigo-300",
  gray: "bg-gray-100 text-gray-800 border-gray-300",
};

export default function StatusBadge({ status, color, text }: StatusBadgeProps) {
  // 处理 undefined 或空字符串的情况
  if (!status) {
    return (
      <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium border bg-gray-100 text-gray-800 border-gray-300">
        UNKNOWN
      </span>
    );
  }

  // 如果提供了 color，使用指定的颜色，否则根据 status 查找
  const colorClass = color
    ? statusColors[color.toLowerCase()] || statusColors.gray
    : statusColors[status.toLowerCase()] || statusColors.pending;

  // 显示文本：优先使用 text，其次使用 status
  const displayText = text || status;

  return (
    <span
      className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium border ${colorClass}`}
    >
      {displayText.toUpperCase()}
    </span>
  );
}
