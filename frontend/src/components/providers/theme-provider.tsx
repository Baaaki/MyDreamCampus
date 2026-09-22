import { useEffect, useState } from "react"
import { ThemeContext, type Theme } from "./theme-context"

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => {
    const savedTheme = localStorage.getItem("theme") as Theme | null
    if (savedTheme === "light" || savedTheme === "dark") {
      return savedTheme
    }
    if (
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches
    ) {
      return "dark"
    }
    return "light"
  })

  useEffect(() => {
    // HTML element'e dark class ekle/kaldır
    const root = document.documentElement
    if (theme === "dark") {
      root.classList.add("dark")
    } else {
      root.classList.remove("dark")
    }
    // localStorage'a kaydet
    localStorage.setItem("theme", theme)
  }, [theme])

  const toggleTheme = () => {
    setThemeState((prev) => (prev === "light" ? "dark" : "light"))
  }

  const setTheme = (newTheme: Theme) => {
    setThemeState(newTheme)
  }

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}
