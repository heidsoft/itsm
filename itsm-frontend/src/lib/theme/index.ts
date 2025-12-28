// Theme configuration and design tokens for the ITSM application
// Following design system best practices with semantic color naming

export const colors = {
  // Primary colors - Blue theme for professional ITSM appearance
  primary: {
    50: '#eff6ff',
    100: '#dbeafe',
    200: '#bfdbfe',
    300: '#93c5fd',
    400: '#60a5fa',
    500: '#3b82f6', // Primary brand color
    600: '#2563eb',
    700: '#1d4ed8',
    800: '#1e40af',
    900: '#1e3a8a',
    950: '#172554',
  },

  // Gray colors for neutral elements
  gray: {
    50: '#f9fafb',
    100: '#f3f4f6',
    200: '#e5e7eb',
    300: '#d1d5db',
    400: '#9ca3af',
    500: '#6b7280',
    600: '#4b5563',
    700: '#374151',
    800: '#1f2937',
    900: '#111827',
    950: '#030712',
  },

  // Semantic colors for status and feedback
  success: {
    50: '#f0fdf4',
    100: '#dcfce7',
    200: '#bbf7d0',
    300: '#86efac',
    400: '#4ade80',
    500: '#22c55e',
    600: '#16a34a',
    700: '#15803d',
    800: '#166534',
    900: '#14532d',
  },

  warning: {
    50: '#fffbeb',
    100: '#fef3c7',
    200: '#fde68a',
    300: '#fcd34d',
    400: '#fbbf24',
    500: '#f59e0b',
    600: '#d97706',
    700: '#b45309',
    800: '#92400e',
    900: '#78350f',
  },

  danger: {
    50: '#fef2f2',
    100: '#fee2e2',
    200: '#fecaca',
    300: '#fca5a5',
    400: '#f87171',
    500: '#ef4444',
    600: '#dc2626',
    700: '#b91c1c',
    800: '#991b1b',
    900: '#7f1d1d',
  },

  info: {
    50: '#f0f9ff',
    100: '#e0f2fe',
    200: '#bae6fd',
    300: '#7dd3fc',
    400: '#38bdf8',
    500: '#0ea5e9',
    600: '#0284c7',
    700: '#0369a1',
    800: '#075985',
    900: '#0c4a6e',
  },

  // Ticket status colors
  status: {
    new: '#3b82f6',      // Blue
    open: '#06b6d4',     // Cyan
    in_progress: '#f59e0b', // Amber
    pending: '#eab308',  // Yellow
    resolved: '#22c55e', // Green
    closed: '#6b7280',   // Gray
    cancelled: '#ef4444', // Red
  },

  // Priority colors
  priority: {
    low: '#22c55e',      // Green
    normal: '#3b82f6',   // Blue
    high: '#f59e0b',     // Amber
    urgent: '#ef4444',   // Red
    critical: '#a855f7', // Purple
  },
} as const;

export const typography = {
  fontFamily: {
    sans: ['Inter', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
    mono: ['JetBrains Mono', 'Monaco', 'Consolas', 'Liberation Mono', 'Courier New', 'monospace'],
  },
  
  fontSize: {
    xs: ['0.75rem', { lineHeight: '1rem' }],
    sm: ['0.875rem', { lineHeight: '1.25rem' }],
    base: ['1rem', { lineHeight: '1.5rem' }],
    lg: ['1.125rem', { lineHeight: '1.75rem' }],
    xl: ['1.25rem', { lineHeight: '1.75rem' }],
    '2xl': ['1.5rem', { lineHeight: '2rem' }],
    '3xl': ['1.875rem', { lineHeight: '2.25rem' }],
    '4xl': ['2.25rem', { lineHeight: '2.5rem' }],
    '5xl': ['3rem', { lineHeight: '1' }],
    '6xl': ['3.75rem', { lineHeight: '1' }],
  },

  fontWeight: {
    thin: '100',
    light: '300',
    normal: '400',
    medium: '500',
    semibold: '600',
    bold: '700',
    extrabold: '800',
  },
} as const;

export const spacing = {
  px: '1px',
  0: '0',
  0.5: '0.125rem',
  1: '0.25rem',
  1.5: '0.375rem',
  2: '0.5rem',
  2.5: '0.625rem',
  3: '0.75rem',
  3.5: '0.875rem',
  4: '1rem',
  5: '1.25rem',
  6: '1.5rem',
  7: '1.75rem',
  8: '2rem',
  9: '2.25rem',
  10: '2.5rem',
  11: '2.75rem',
  12: '3rem',
  14: '3.5rem',
  16: '4rem',
  20: '5rem',
  24: '6rem',
  28: '7rem',
  32: '8rem',
  36: '9rem',
  40: '10rem',
  44: '11rem',
  48: '12rem',
  52: '13rem',
  56: '14rem',
  60: '15rem',
  64: '16rem',
  72: '18rem',
  80: '20rem',
  96: '24rem',
} as const;

export const borderRadius = {
  none: '0',
  sm: '0.125rem',
  DEFAULT: '0.25rem',
  md: '0.375rem',
  lg: '0.5rem',
  xl: '0.75rem',
  '2xl': '1rem',
  '3xl': '1.5rem',
  full: '9999px',
} as const;

export const shadows = {
  sm: '0 1px 2px 0 rgb(0 0 0 / 0.05)',
  DEFAULT: '0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)',
  md: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)',
  lg: '0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)',
  xl: '0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1)',
  '2xl': '0 25px 50px -12px rgb(0 0 0 / 0.25)',
  inner: 'inset 0 2px 4px 0 rgb(0 0 0 / 0.05)',
  none: '0 0 #0000',
} as const;

export const zIndex = {
  auto: 'auto',
  0: '0',
  10: '10',
  20: '20',
  30: '30',
  40: '40',
  50: '50',
  dropdown: '1000',
  sticky: '1020',
  fixed: '1030',
  modal: '1040',
  popover: '1050',
  tooltip: '1060',
  notification: '1070',
} as const;

// Component-specific design tokens
export const components = {
  button: {
    // Button sizes
    size: {
      xs: {
        padding: '0.25rem 0.5rem',
        fontSize: '0.75rem',
        borderRadius: '0.25rem',
      },
      sm: {
        padding: '0.375rem 0.75rem',
        fontSize: '0.875rem',
        borderRadius: '0.25rem',
      },
      md: {
        padding: '0.5rem 1rem',
        fontSize: '0.875rem',
        borderRadius: '0.375rem',
      },
      lg: {
        padding: '0.625rem 1.25rem',
        fontSize: '1rem',
        borderRadius: '0.375rem',
      },
      xl: {
        padding: '0.75rem 1.5rem',
        fontSize: '1.125rem',
        borderRadius: '0.5rem',
      },
    },
    
    // Button variants
    variant: {
      primary: {
        backgroundColor: colors.primary[600],
        color: '#ffffff',
        border: `1px solid ${colors.primary[600]}`,
        hover: {
          backgroundColor: colors.primary[700],
        },
        focus: {
          ring: `2px solid ${colors.primary[500]}`,
          ringOffset: '2px',
        },
      },
      secondary: {
        backgroundColor: colors.gray[100],
        color: colors.gray[900],
        border: `1px solid ${colors.gray[300]}`,
        hover: {
          backgroundColor: colors.gray[200],
        },
        focus: {
          ring: `2px solid ${colors.gray[500]}`,
          ringOffset: '2px',
        },
      },
      outline: {
        backgroundColor: 'transparent',
        color: colors.primary[600],
        border: `1px solid ${colors.primary[600]}`,
        hover: {
          backgroundColor: colors.primary[50],
        },
        focus: {
          ring: `2px solid ${colors.primary[500]}`,
          ringOffset: '2px',
        },
      },
      ghost: {
        backgroundColor: 'transparent',
        color: colors.gray[700],
        border: '1px solid transparent',
        hover: {
          backgroundColor: colors.gray[100],
        },
        focus: {
          ring: `2px solid ${colors.gray[500]}`,
          ringOffset: '2px',
        },
      },
      danger: {
        backgroundColor: colors.danger[600],
        color: '#ffffff',
        border: `1px solid ${colors.danger[600]}`,
        hover: {
          backgroundColor: colors.danger[700],
        },
        focus: {
          ring: `2px solid ${colors.danger[500]}`,
          ringOffset: '2px',
        },
      },
    },
  },

  input: {
    size: {
      sm: {
        padding: '0.375rem 0.75rem',
        fontSize: '0.875rem',
        borderRadius: '0.25rem',
      },
      md: {
        padding: '0.5rem 0.75rem',
        fontSize: '0.875rem',
        borderRadius: '0.375rem',
      },
      lg: {
        padding: '0.625rem 1rem',
        fontSize: '1rem',
        borderRadius: '0.375rem',
      },
    },
    
    variant: {
      default: {
        backgroundColor: '#ffffff',
        border: `1px solid ${colors.gray[300]}`,
        color: colors.gray[900],
        placeholder: colors.gray[500],
        focus: {
          borderColor: colors.primary[500],
          ring: `2px solid ${colors.primary[500]}`,
        },
      },
      error: {
        backgroundColor: '#ffffff',
        border: `1px solid ${colors.danger[300]}`,
        color: colors.gray[900],
        placeholder: colors.gray[500],
        focus: {
          borderColor: colors.danger[500],
          ring: `2px solid ${colors.danger[500]}`,
        },
      },
    },
  },

  card: {
    variant: {
      default: {
        backgroundColor: '#ffffff',
        border: `1px solid ${colors.gray[200]}`,
        borderRadius: '0.5rem',
        boxShadow: shadows.sm,
      },
      elevated: {
        backgroundColor: '#ffffff',
        border: `1px solid ${colors.gray[100]}`,
        borderRadius: '0.5rem',
        boxShadow: shadows.lg,
      },
      outlined: {
        backgroundColor: '#ffffff',
        border: `2px solid ${colors.gray[300]}`,
        borderRadius: '0.5rem',
        boxShadow: shadows.none,
      },
      filled: {
        backgroundColor: colors.gray[50],
        border: `1px solid ${colors.gray[200]}`,
        borderRadius: '0.5rem',
        boxShadow: shadows.none,
      },
    },
  },

  badge: {
    size: {
      xs: {
        padding: '0.125rem 0.375rem',
        fontSize: '0.75rem',
        borderRadius: '9999px',
      },
      sm: {
        padding: '0.25rem 0.5rem',
        fontSize: '0.75rem',
        borderRadius: '9999px',
      },
      md: {
        padding: '0.25rem 0.75rem',
        fontSize: '0.875rem',
        borderRadius: '9999px',
      },
    },

    variant: {
      default: {
        backgroundColor: colors.gray[100],
        color: colors.gray[800],
      },
      primary: {
        backgroundColor: colors.primary[100],
        color: colors.primary[800],
      },
      success: {
        backgroundColor: colors.success[100],
        color: colors.success[800],
      },
      warning: {
        backgroundColor: colors.warning[100],
        color: colors.warning[800],
      },
      danger: {
        backgroundColor: colors.danger[100],
        color: colors.danger[800],
      },
      info: {
        backgroundColor: colors.info[100],
        color: colors.info[800],
      },
    },
  },
} as const;

// Layout constants
export const layout = {
  // Container max widths
  container: {
    sm: '640px',
    md: '768px',
    lg: '1024px',
    xl: '1280px',
    '2xl': '1536px',
  },

  // Sidebar widths
  sidebar: {
    narrow: '256px',
    normal: '288px',
    wide: '320px',
  },

  // Header heights
  header: {
    sm: '3rem',
    md: '4rem',
    lg: '5rem',
  },

  // Content spacing
  section: {
    padding: '2rem',
    gap: '1.5rem',
  },
} as const;

// Animation and transition configurations
export const animation = {
  duration: {
    fast: '150ms',
    normal: '200ms',
    slow: '300ms',
  },

  easing: {
    ease: 'ease',
    'ease-in': 'ease-in',
    'ease-out': 'ease-out',
    'ease-in-out': 'ease-in-out',
    linear: 'linear',
  },

  // Common transitions
  transition: {
    all: 'all 200ms ease-in-out',
    colors: 'color 200ms ease-in-out, background-color 200ms ease-in-out, border-color 200ms ease-in-out',
    opacity: 'opacity 200ms ease-in-out',
    transform: 'transform 200ms ease-in-out',
  },
} as const;

// Responsive breakpoints
export const breakpoints = {
  sm: '640px',
  md: '768px',
  lg: '1024px',
  xl: '1280px',
  '2xl': '1536px',
} as const;

// Export the complete theme
export const theme = {
  colors,
  typography,
  spacing,
  borderRadius,
  shadows,
  zIndex,
  components,
  layout,
  animation,
  breakpoints,
} as const;

export type Theme = typeof theme;
export type Colors = typeof colors;
export type Typography = typeof typography;
export type Components = typeof components;

export default theme;