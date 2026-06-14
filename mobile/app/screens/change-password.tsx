import React, { useState } from 'react';
import { View, StyleSheet, KeyboardAvoidingView, Platform } from 'react-native';
import { Text, TextInput, Button, Surface, useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';

import { useChangePassword } from '@/hooks/useAuth';
import { useToast } from '@/contexts/ToastContext';
import { useHaptic } from '@/hooks/useHaptic';
import { spacing, radius } from '@/constants/tokens';
import { validatePasswordPolicy } from '@/lib/password-policy';

export default function ChangePasswordScreen() {
  const router = useRouter();
  const { colors } = useTheme();
  const haptic = useHaptic();
  const toast = useToast();
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);

  const changePasswordMutation = useChangePassword();

  const handleSubmit = () => {
    if (!oldPassword || !newPassword || !confirmPassword) {
      haptic.error();
      toast.show({ message: 'Lutfen tum alanlari doldurun', type: 'warning' });
      return;
    }

    if (newPassword !== confirmPassword) {
      haptic.error();
      toast.show({ message: 'Yeni sifreler eslesmedi', type: 'error' });
      return;
    }

    const policyError = validatePasswordPolicy(newPassword);
    if (policyError) {
      haptic.error();
      toast.show({ message: policyError, type: 'warning' });
      return;
    }

    haptic.light();
    changePasswordMutation.mutate(
      { old_password: oldPassword, new_password: newPassword },
      {
        onSuccess: () => {
          haptic.success();
          toast.show({ message: 'Sifreniz basariyla degistirildi', type: 'success' });
          router.replace('/(tabs)');
        },
        onError: (error: any) => {
          haptic.error();
          const errorData = error.response?.data;
          const message = errorData?.message || 'Sifre degistirme basarisiz';
          toast.show({ message, type: 'error' });
        },
      }
    );
  };

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      style={[styles.container, { backgroundColor: colors.background }]}
    >
      <View style={styles.content}>
        <Surface style={[styles.formCard, { backgroundColor: colors.surface }]} elevation={1}>
          <Text variant="titleMedium" style={{ color: colors.onSurface, marginBottom: spacing.md }}>
            Sifrenizi degistirin
          </Text>

          <TextInput
            label="Mevcut Sifre"
            value={oldPassword}
            onChangeText={setOldPassword}
            secureTextEntry={!showPassword}
            disabled={changePasswordMutation.isPending}
            mode="outlined"
            left={<TextInput.Icon icon="lock-outline" />}
            style={styles.input}
          />

          <TextInput
            label="Yeni Sifre"
            value={newPassword}
            onChangeText={setNewPassword}
            secureTextEntry={!showPassword}
            disabled={changePasswordMutation.isPending}
            mode="outlined"
            left={<TextInput.Icon icon="lock-plus-outline" />}
            style={styles.input}
          />

          <TextInput
            label="Yeni Sifre Tekrar"
            value={confirmPassword}
            onChangeText={setConfirmPassword}
            secureTextEntry={!showPassword}
            disabled={changePasswordMutation.isPending}
            mode="outlined"
            left={<TextInput.Icon icon="lock-check-outline" />}
            right={
              <TextInput.Icon
                icon={showPassword ? 'eye-off' : 'eye'}
                onPress={() => setShowPassword(!showPassword)}
              />
            }
            style={styles.input}
          />

          <Button
            mode="contained"
            onPress={handleSubmit}
            loading={changePasswordMutation.isPending}
            disabled={changePasswordMutation.isPending}
            style={styles.submitButton}
            contentStyle={styles.buttonContent}
          >
            Sifreyi Degistir
          </Button>
        </Surface>
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  content: {
    flex: 1,
    justifyContent: 'center',
    paddingHorizontal: spacing.lg,
  },
  formCard: {
    borderRadius: radius.xl,
    padding: spacing.lg,
  },
  input: {
    marginBottom: spacing.md,
  },
  submitButton: {
    marginTop: spacing.sm,
    borderRadius: radius.md,
  },
  buttonContent: {
    paddingVertical: 6,
  },
});
