// toggle theme
document.querySelectorAll("[data-theme-toggle]").forEach((el) => {
  el.addEventListener("click", () => {
    const isDark = document.documentElement.classList.toggle("dark")
    if (isDark) localStorage.setItem("theme", "dark")
    else localStorage.removeItem("theme")
  });
});