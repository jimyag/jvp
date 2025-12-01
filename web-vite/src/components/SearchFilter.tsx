import { Search, X } from "lucide-react";
import { useState } from "react";

interface SearchFilterProps {
  onSearch: (query: string) => void;
  placeholder?: string;
}

export default function SearchFilter({
  onSearch,
  placeholder = "Search...",
}: SearchFilterProps) {
  const [query, setQuery] = useState("");

  const handleChange = (value: string) => {
    setQuery(value);
    onSearch(value);
  };

  const handleClear = () => {
    setQuery("");
    onSearch("");
  };

  return (
    <div className="relative">
      <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
        <Search className="h-5 w-5 text-gray-400" />
      </div>
      <input
        type="text"
        className="block w-full pl-10 pr-10 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-accent focus:border-accent"
        placeholder={placeholder}
        value={query}
        onChange={(e) => handleChange(e.target.value)}
      />
      {query && (
        <button
          onClick={handleClear}
          className="absolute inset-y-0 right-0 pr-3 flex items-center text-gray-400 hover:text-gray-600"
        >
          <X className="h-5 w-5" />
        </button>
      )}
    </div>
  );
}
