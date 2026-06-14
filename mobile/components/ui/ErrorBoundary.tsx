/**
 * ErrorBoundary — Uygulama crash fallback bileşeni.
 *
 * Beklenmeyen hatalar oluştuğunda kullanıcıya bilgilendirme gösterir
 * ve uygulamayı yeniden başlatma seçeneği sunar.
 * withTheme HOC ile tema renklerine erişir.
 */
import React, { Component, type ReactNode } from 'react';
import { StyleSheet, View } from 'react-native';
import { Text, Button, Surface, withTheme, type MD3Theme } from 'react-native-paper';
import FontAwesome from '@expo/vector-icons/FontAwesome';
import { spacing, radius, semanticColors } from '../../constants/tokens';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  theme: MD3Theme;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

class ErrorBoundaryInner extends Component<Props, State> {
  state: State = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;

      const { colors } = this.props.theme;

      return (
        <View style={[styles.container, { backgroundColor: colors.background }]} accessibilityRole="alert">
          <Surface style={styles.card} elevation={2}>
            <View style={styles.iconContainer}>
              <FontAwesome name="exclamation-triangle" size={48} color={semanticColors.warning} />
            </View>
            <Text variant="headlineSmall" style={[styles.title, { color: colors.onSurface }]}>
              Bir hata olustu
            </Text>
            <Text variant="bodyMedium" style={[styles.message, { color: colors.onSurfaceVariant }]}>
              Beklenmeyen bir sorun meydana geldi. Lutfen tekrar deneyin.
            </Text>
            {__DEV__ && this.state.error && (
              <Surface style={[styles.errorDetail, { backgroundColor: colors.errorContainer }]} elevation={0}>
                <Text variant="bodySmall" style={[styles.errorText, { color: colors.onErrorContainer }]} numberOfLines={4}>
                  {this.state.error.message}
                </Text>
              </Surface>
            )}
            <Button
              mode="contained"
              onPress={this.handleReset}
              style={styles.button}
              accessibilityLabel="Tekrar dene"
            >
              Tekrar Dene
            </Button>
          </Surface>
        </View>
      );
    }

    return this.props.children;
  }
}

export const ErrorBoundary = withTheme(ErrorBoundaryInner);

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: spacing.lg,
  },
  card: {
    borderRadius: radius.xl,
    padding: spacing.xl,
    alignItems: 'center',
    maxWidth: 360,
    width: '100%',
  },
  iconContainer: {
    marginBottom: spacing.lg,
  },
  title: {
    fontWeight: '700',
    textAlign: 'center',
    marginBottom: spacing.sm,
  },
  message: {
    textAlign: 'center',
    marginBottom: spacing.lg,
  },
  errorDetail: {
    borderRadius: radius.sm,
    padding: spacing.md,
    width: '100%',
    marginBottom: spacing.md,
  },
  errorText: {
    fontFamily: 'SpaceMono',
    fontSize: 11,
  },
  button: {
    borderRadius: radius.sm,
    minWidth: 160,
  },
});
