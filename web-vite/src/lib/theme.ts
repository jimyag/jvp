// 主题切换：system / light / dark，偏好保存在 localStorage

export type ThemePreference = "system" | "light" | "dark";

const STORAGE_KEY = "jvp.theme";
const media = typeof window !== "undefined" ? window.matchMedia("(prefers-color-scheme: dark)") : null;

export function getThemePreference(): ThemePreference {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    if (value === "light" || value === "dark" || value === "system") return value;
  } catch {
    // ignore
  }
  return "system";
}

function apply(pref: ThemePreference) {
  const dark = pref === "dark" || (pref === "system" && !!media?.matches);
  document.documentElement.classList.toggle("dark", dark);
}

export function setThemePreference(pref: ThemePreference) {
  try {
    localStorage.setItem(STORAGE_KEY, pref);
  } catch {
    // ignore
  }
  apply(pref);
}

media?.addEventListener("change", () => {
  if (getThemePreference() === "system") apply("system");
});
