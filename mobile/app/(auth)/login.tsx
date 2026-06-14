import React, { useState } from 'react';
import { View, StyleSheet, KeyboardAvoidingView, Platform } from 'react-native';
import { Text, TextInput, Button, Surface, useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import FontAwesome from '@expo/vector-icons/FontAwesome';

import { useLogin } from '@/hooks/useAuth';
import { useAuthContext } from '@/contexts/AuthContext';
import { useToast } from '@/contexts/ToastContext';
import { useHaptic } from '@/hooks/useHaptic';
import { spacing, radius } from '@/constants/tokens';

export default function LoginScreen() {
  const router = useRouter();
  const { colors } = useTheme();
  const haptic = useHaptic();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const { setUser } = useAuthContext();
  const toast = useToast();

  const loginMutation = useLogin();

  const handleLogin = () => {
    if (!email || !password) {
      haptic.error();
      toast.show({ message: 'Lutfen tum alanlari doldurun', type: 'warning' });
      return;
    }

    haptic.light();
    loginMutation.mutate(
      { email, password },
      {
        onSuccess: (data) => {
          haptic.success();
          setUser(data.user);

          if (data.force_password_change) {
            toast.show({ message: 'Lutfen sifrenizi degistirin', type: 'warning' });
            router.replace('/screens/change-password');
            return;
          }

          router.replace('/(tabs)');
        },
        onError: (error: any) => {
          haptic.error();
          const errorData = error.response?.data;
          let message = 'Giris basarisiz';

          if (errorData?.error === 'ACCOUNT_LOCKED') {
            message = errorData.message || 'Hesabiniz gecici olarak kilitlendi';
          } else if (errorData?.error === 'ACCOUNT_DEACTIVATED') {
            message = 'Hesabiniz devre disi birakilmis';
          } else if (errorData?.error === 'INVALID_CREDENTIALS') {
            message = 'Gecersiz e-posta veya sifre';
          } else if (errorData?.message) {
            message = errorData.message;
          }

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
        <View style={styles.logoArea} accessibilityRole="header">
          <Surface style={[styles.logoCircle, { backgroundColor: colors.primaryContainer }]} elevation={0}>
            <FontAwesome name="graduation-cap" size={40} color={colors.primary} />
          </Surface>
          <Text variant="headlineLarge" style={[styles.title, { color: colors.primary }]}>
            MyDreamCampus
          </Text>
          <Text variant="bodyLarge" style={{ color: colors.onSurfaceVariant }}>
            Hosgeldiniz
          </Text>
        </View>

        <Surface style={[styles.formCard, { backgroundColor: colors.surface }]} elevation={1}>
          <TextInput
            label="E-posta"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            keyboardType="email-address"
            disabled={loginMutation.isPending}
            mode="outlined"
            left={<TextInput.Icon icon="email-outline" />}
            style={styles.input}
            accessibilityLabel="E-posta adresi"
            accessibilityHint="Universite e-posta adresinizi girin"
          />

          <TextInput
            label="Sifre"
            value={password}
            onChangeText={setPassword}
            secureTextEntry={!showPassword}
            disabled={loginMutation.isPending}
            mode="outlined"
            left={<TextInput.Icon icon="lock-outline" />}
            right={
              <TextInput.Icon
                icon={showPassword ? 'eye-off' : 'eye'}
                onPress={() => { haptic.selection(); setShowPassword(!showPassword); }}
                accessibilityLabel={showPassword ? 'Sifreyi gizle' : 'Sifreyi goster'}
              />
            }
            style={styles.input}
            accessibilityLabel="Sifre"
          />

          <Button
            mode="contained"
            onPress={handleLogin}
            loading={loginMutation.isPending}
            disabled={loginMutation.isPending}
            style={styles.loginButton}
            contentStyle={styles.buttonContent}
            labelStyle={styles.buttonLabel}
            accessibilityLabel="Giris yap"
          >
            Giris Yap
          </Button>
        </Surface>
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  content: {
    flex: 1,
    justifyContent: 'center',
    paddingHorizontal: spacing.lg,
  },
  logoArea: {
    alignItems: 'center',
    marginBottom: spacing.xl,
  },
  logoCircle: {
    width: 88,
    height: 88,
    borderRadius: 44,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: spacing.md,
  },
  title: {
    fontWeight: 'bold',
    marginBottom: spacing.xs,
  },
  formCard: {
    borderRadius: radius.xl,
    padding: spacing.lg,
  },
  input: {
    marginBottom: spacing.md,
  },
  loginButton: {
    marginTop: spacing.sm,
    borderRadius: radius.md,
  },
  buttonContent: {
    paddingVertical: 6,
  },
  buttonLabel: {
    fontSize: 16,
    fontWeight: '600',
  },
});
