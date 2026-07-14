import { ViewStyle } from 'react-native';

// Frosted-glass card style used across all dark-background screens.
export const GLASS: ViewStyle = {
  backgroundColor: 'rgba(255,255,255,0.06)',
  borderRadius: 16,
  borderWidth: 1,
  borderColor: 'rgba(255,255,255,0.1)',
};

export const COLORS = {
  teal: '#2dd4a7',
  bg: '#0f172a',
  navy: '#1e3a5f',
  red: '#fca5a5',
  green: '#4ade80',
  muted: 'rgba(255,255,255,0.4)',
  mutedLight: 'rgba(255,255,255,0.1)',
} as const;
