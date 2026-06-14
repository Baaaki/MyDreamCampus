/**
 * ScreenWrapper — Pull-to-refresh destekli ekran container'ı.
 *
 * Tüm data-driven ekranlar için ortak ScrollView wrapper'ı.
 * RefreshControl ile pull-to-refresh, standart padding ve background sağlar.
 */
import { type ReactNode, useCallback, useState } from 'react';
import { StyleSheet, ScrollView, RefreshControl, type ScrollViewProps } from 'react-native';
import { useTheme } from 'react-native-paper';
import { layout } from '../../constants/tokens';
import { useHaptic } from '../../hooks/useHaptic';

interface ScreenWrapperProps extends Omit<ScrollViewProps, 'refreshControl'> {
  children: ReactNode;
  onRefresh?: () => Promise<void>;
  noPadding?: boolean;
}

export function ScreenWrapper({ children, onRefresh, noPadding, style, ...rest }: ScreenWrapperProps) {
  const { colors } = useTheme();
  const haptic = useHaptic();
  const [refreshing, setRefreshing] = useState(false);

  const handleRefresh = useCallback(async () => {
    if (!onRefresh) return;
    haptic.light();
    setRefreshing(true);
    try {
      await onRefresh();
    } finally {
      setRefreshing(false);
    }
  }, [onRefresh, haptic]);

  return (
    <ScrollView
      style={[styles.scrollView, { backgroundColor: colors.background }, style]}
      contentContainerStyle={noPadding ? undefined : styles.content}
      refreshControl={
        onRefresh ? (
          <RefreshControl
            refreshing={refreshing}
            onRefresh={handleRefresh}
            colors={[colors.primary]}
            tintColor={colors.primary}
            progressBackgroundColor={colors.surface}
          />
        ) : undefined
      }
      {...rest}
    >
      {children}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  scrollView: { flex: 1 },
  content: { padding: layout.screenPadding },
});
