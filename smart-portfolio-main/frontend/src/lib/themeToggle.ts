/**
 * Theme toggle utility with View Transition API support
 */

const THEME_KEY = 'theme-preference';
const DARK_CLASS = 'dark';

/**
 * Get the current theme preference from localStorage or system preference
 */
export function getThemePreference(): 'light' | 'dark' {
  const stored = localStorage.getItem(THEME_KEY);
  if (stored === 'light' || stored === 'dark') {
    return stored;
  }

  // Check system preference
  if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'dark';
  }

  return 'light';
}

/**
 * Apply theme to the document
 */
export function applyTheme(theme: 'light' | 'dark') {
  const html = document.documentElement;

  if (theme === 'dark') {
    html.classList.add(DARK_CLASS);
  } else {
    html.classList.remove(DARK_CLASS);
  }

  localStorage.setItem(THEME_KEY, theme);
}

/**
 * Toggle between light and dark themes with View Transition API
 * @param originElement - Element to originate circular transition from
 */
export async function toggleTheme(originElement?: Element) {
  const currentTheme = getThemePreference();
  const newTheme = currentTheme === 'dark' ? 'light' : 'dark';

  // Get the element's position for circular transition SYNCHRONOUSLY
  let x = window.innerWidth / 2;
  let y = window.innerHeight / 2;

  if (originElement) {
    const rect = originElement.getBoundingClientRect();
    x = rect.left + rect.width / 2;
    y = rect.top + rect.height / 2;
  }

  // Store the transition origin IMMEDIATELY before any async operations
  const html = document.documentElement as any;
  html.style.setProperty('--transition-x', `${x}px`);
  html.style.setProperty('--transition-y', `${y}px`);

  // Use View Transition API if available
  if (document.startViewTransition) {
    // Create transition and apply theme synchronously inside callback
    document.startViewTransition(() => {
      applyTheme(newTheme);
    });
  } else {
    applyTheme(newTheme);
  }
}

/**
 * Initialize theme on page load
 */
export function initializeTheme() {
  const theme = getThemePreference();
  applyTheme(theme);
}
