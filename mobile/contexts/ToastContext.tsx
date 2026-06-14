/**
 * ToastContext — Global snackbar/toast bildirimleri.
 *
 * Herhangi bir ekrandan `useToast().show(...)` ile bildirim gösterir.
 * react-native-paper Snackbar bileşenini kullanır.
 *
 * Kullanım:
 *   const toast = useToast();
 *   toast.show({ message: 'Basarili!', type: 'success' });
 *   toast.show({ message: 'Hata olustu', type: 'error' });
 */
import React, { createContext, useContext, useState, useCallback, useEffect, useRef, type ReactNode } from 'react';
import { StyleSheet, View } from 'react-native';
import { Snackbar, Text, useTheme } from 'react-native-paper';
import FontAwesome from '@expo/vector-icons/FontAwesome';
import { semanticColors, spacing, radius } from '@/constants/tokens';

type ToastType = 'success' | 'error' | 'warning' | 'info';

interface ToastOptions {
  message: string;
  type?: ToastType;
  duration?: number;
  action?: { label: string; onPress: () => void };
}

interface ToastContextValue {
  show: (options: ToastOptions) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

const toastConfig: Record<ToastType, { icon: keyof typeof FontAwesome.glyphMap; color: string; bg: string }> = {
  success: { icon: 'check-circle', color: semanticColors.successDark, bg: semanticColors.successLight },
  error: { icon: 'times-circle', color: semanticColors.dangerDark, bg: semanticColors.dangerLight },
  warning: { icon: 'exclamation-circle', color: semanticColors.warningDark, bg: semanticColors.warningLight },
  info: { icon: 'info-circle', color: semanticColors.infoDark, bg: semanticColors.infoLight },
};

const DEFAULT_DURATION = 3000;

export function ToastProvider({ children }: { children: ReactNode }) {
  const { colors } = useTheme();
  const [visible, setVisible] = useState(false);
  const [current, setCurrent] = useState<ToastOptions>({ message: '' });
  const queueRef = useRef<ToastOptions[]>([]);
  // Tracked synchronously so two show() calls in the same tick don't both
  // see visible=false and race to display.
  const showingRef = useRef(false);

  const driveNext = useCallback(() => {
    if (showingRef.current) return;
    const next = queueRef.current.shift();
    if (!next) return;
    showingRef.current = true;
    setCurrent(next);
    setVisible(true);
  }, []);

  const show = useCallback((options: ToastOptions) => {
    queueRef.current.push(options);
    driveNext();
  }, [driveNext]);

  const handleDismiss = useCallback(() => {
    setVisible(false);
  }, []);

  // After the snackbar finishes hiding, advance the queue.
  useEffect(() => {
    if (!visible && showingRef.current) {
      showingRef.current = false;
      driveNext();
    }
  }, [visible, driveNext]);

  const type = current.type ?? 'info';
  const config = toastConfig[type];

  return (
    <ToastContext.Provider value={{ show }}>
      {children}
      <Snackbar
        visible={visible}
        onDismiss={handleDismiss}
        duration={current.duration ?? DEFAULT_DURATION}
        action={current.action}
        style={[styles.snackbar, { backgroundColor: config.bg, borderLeftColor: config.color }]}
        wrapperStyle={styles.wrapper}
        accessibilityLiveRegion="polite"
      >
        <View style={styles.content}>
          <FontAwesome name={config.icon} size={16} color={config.color} />
          <Text variant="bodyMedium" style={{ color: config.color, flex: 1 }}>
            {current.message}
          </Text>
        </View>
      </Snackbar>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast must be used within ToastProvider');
  return ctx;
}

const styles = StyleSheet.create({
  wrapper: {
    position: 'absolute',
    bottom: 0,
  },
  snackbar: {
    borderLeftWidth: 4,
    borderRadius: radius.md,
    marginHorizontal: spacing.md,
  },
  content: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
  },
});
