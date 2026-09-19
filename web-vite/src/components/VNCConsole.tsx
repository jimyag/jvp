import { ExternalLink } from "lucide-react";

interface VNCConsoleProps {
  wsUrl: string;
}

export default function VNCConsole({ wsUrl }: VNCConsoleProps) {
  if (!wsUrl) {
    return (
      <div className="flex items-center justify-center h-full bg-black">
        <p className="text-white">Initializing VNC console...</p>
      </div>
    );
  }

  const consoleUrl = `/vnc.html?ws=${encodeURIComponent(wsUrl)}`;

  return (
    <div className="flex min-h-44 items-center justify-center bg-gray-50">
      <a
        href={consoleUrl}
        target="_blank"
        rel="noopener noreferrer"
        className="btn-primary inline-flex items-center gap-2"
      >
        <ExternalLink size={16} />
        Open VNC Console
      </a>
    </div>
  );
}
