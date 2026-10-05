import { Search, X } from "lucide-react";

interface SearchFilterProps {
  value: string;
  onChange: (query: string) => void;
  placeholder?: string;
  className?: string;
}

export default function SearchFilter({ value, onChange, placeholder = "Search…", className = "" }: SearchFilterProps) {
  return (
    <div className={`relative ${className}`}>
      <Search size={15} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-fg-subtle" />
      <input
        type="search"
        className="input pl-9 pr-8 [&::-webkit-search-cancel-button]:hidden"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Escape") onChange("");
        }}
      />
      {value && (
        <button
          type="button"
          onClick={() => onChange("")}
          className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-fg-subtle hover:text-fg"
          aria-label="Clear search"
        >
          <X size={14} />
        </button>
      )}
    </div>
  );
}
