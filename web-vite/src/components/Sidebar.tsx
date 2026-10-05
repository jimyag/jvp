import { useState } from "react";
import { NavLink } from "react-router-dom";
import {
  Camera,
  Database,
  Github,
  KeyRound,
  Layers,
  LayoutDashboard,
  Menu,
  Monitor,
  Moon,
  Network,
  Server,
  ServerCog,
  Sun,
  X,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { getThemePreference, setThemePreference } from "@/lib/theme";
import type { ThemePreference } from "@/lib/theme";

interface NavItem {
  name: string;
  href: string;
  icon: LucideIcon;
  end?: boolean;
}

const navigation: { title?: string; items: NavItem[] }[] = [
  { items: [{ name: "Overview", href: "/", icon: LayoutDashboard, end: true }] },
  {
    title: "Compute",
    items: [
      { name: "Instances", href: "/instances", icon: Server },
      { name: "Snapshots", href: "/snapshots", icon: Camera },
      { name: "Templates", href: "/templates", icon: Layers },
      { name: "Key pairs", href: "/keypairs", icon: KeyRound },
    ],
  },
  {
    title: "Infrastructure",
    items: [
      { name: "Nodes", href: "/nodes", icon: ServerCog },
      { name: "Networks", href: "/networks", icon: Network },
      { name: "Storage", href: "/storage-pools", icon: Database },
    ],
  },
];

function Logo() {
  return (
    <div className="flex items-center gap-2.5">
      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent text-accent-fg shadow-card">
        <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
          <rect x="4" y="4" width="16" height="4.5" rx="1.2" />
          <rect x="4" y="10" width="16" height="4.5" rx="1.2" opacity="0.8" />
          <rect x="4" y="16" width="16" height="4" rx="1.2" opacity="0.6" />
        </svg>
      </div>
      <div className="leading-tight">
        <div className="text-[15px] font-semibold tracking-tight text-fg">JVP</div>
        <div className="text-[11px] text-fg-subtle">Virtualization Platform</div>
      </div>
    </div>
  );
}

const themeOptions: { value: ThemePreference; icon: LucideIcon; label: string }[] = [
  { value: "light", icon: Sun, label: "Light" },
  { value: "system", icon: Monitor, label: "System" },
  { value: "dark", icon: Moon, label: "Dark" },
];

function ThemeSwitcher() {
  const [theme, setTheme] = useState<ThemePreference>(getThemePreference);
  return (
    <div className="flex rounded-md bg-subtle p-0.5 ring-1 ring-inset ring-line" role="radiogroup" aria-label="Theme">
      {themeOptions.map(({ value, icon: Icon, label }) => (
        <button
          key={value}
          type="button"
          role="radio"
          aria-checked={theme === value}
          title={label}
          onClick={() => {
            setTheme(value);
            setThemePreference(value);
          }}
          className={`flex h-6 w-7 items-center justify-center rounded-[5px] transition-colors ${
            theme === value ? "bg-surface text-fg shadow-card ring-1 ring-line" : "text-fg-subtle hover:text-fg"
          }`}
        >
          <Icon size={13} />
        </button>
      ))}
    </div>
  );
}

function SidebarContent({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <div className="flex h-full flex-col">
      <div className="flex h-14 items-center px-4">
        <Logo />
      </div>

      <nav className="flex-1 space-y-5 overflow-y-auto px-3 py-3">
        {navigation.map((group, index) => (
          <div key={group.title || index}>
            {group.title && (
              <div className="mb-1.5 px-2 text-[11px] font-medium uppercase tracking-wider text-fg-subtle">{group.title}</div>
            )}
            <div className="space-y-0.5">
              {group.items.map((item) => {
                const Icon = item.icon;
                return (
                  <NavLink
                    key={item.href}
                    to={item.href}
                    end={item.end}
                    onClick={onNavigate}
                    className={({ isActive }) =>
                      `group flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm font-medium transition-colors ${
                        isActive ? "bg-accent-soft text-accent" : "text-fg-muted hover:bg-subtle hover:text-fg"
                      }`
                    }
                  >
                    <Icon size={16} className="flex-shrink-0" />
                    {item.name}
                  </NavLink>
                );
              })}
            </div>
          </div>
        ))}
      </nav>

      <div className="flex items-center justify-between border-t border-line px-4 py-3">
        <a
          href="https://github.com/jimyag/jvp"
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-1.5 text-xs text-fg-subtle transition-colors hover:text-fg"
        >
          <Github size={14} />
          GitHub
        </a>
        <ThemeSwitcher />
      </div>
    </div>
  );
}

export default function Sidebar() {
  const [open, setOpen] = useState(false);

  return (
    <>
      {/* Mobile top bar */}
      <div className="sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-line bg-surface/90 px-4 backdrop-blur lg:hidden">
        <button type="button" className="btn-icon -ml-1" onClick={() => setOpen(true)} aria-label="Open navigation">
          <Menu size={20} />
        </button>
        <Logo />
      </div>

      {/* Desktop sidebar */}
      <aside className="fixed inset-y-0 left-0 z-30 hidden w-60 border-r border-line bg-surface lg:block">
        <SidebarContent />
      </aside>

      {/* Mobile drawer */}
      {open && (
        <div className="fixed inset-0 z-50 lg:hidden">
          <div className="absolute inset-0 animate-fade-in bg-black/40" onClick={() => setOpen(false)} />
          <aside className="absolute inset-y-0 left-0 w-64 border-r border-line bg-surface shadow-pop animate-slide-up">
            <button
              type="button"
              className="btn-icon absolute right-2 top-3"
              onClick={() => setOpen(false)}
              aria-label="Close navigation"
            >
              <X size={18} />
            </button>
            <SidebarContent onNavigate={() => setOpen(false)} />
          </aside>
        </div>
      )}
    </>
  );
}
