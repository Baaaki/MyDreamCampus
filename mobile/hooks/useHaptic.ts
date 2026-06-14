/**
 * useHaptic — Haptic feedback hook.
 *
 * Buton tıklamaları, kart basımları ve pull-to-refresh gibi
 * etkileşimlerde dokunsal geri bildirim sağlar.
 */
import * as Haptics from 'expo-haptics';
import { Platform } from 'react-native';

export function useHaptic() {
  const light = () => {
    if (Platform.OS !== 'web') {
      Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    }
  };

  const medium = () => {
    if (Platform.OS !== 'web') {
      Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    }
  };

  const success = () => {
    if (Platform.OS !== 'web') {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
    }
  };

  const error = () => {
    if (Platform.OS !== 'web') {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    }
  };

  const selection = () => {
    if (Platform.OS !== 'web') {
      Haptics.selectionAsync();
    }
  };

  return { light, medium, success, error, selection };
}
