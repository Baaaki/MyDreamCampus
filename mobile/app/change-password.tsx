import Ionicons from '@expo/vector-icons/Ionicons';
import { useRouter } from 'expo-router';
import React, { useState } from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, View } from 'react-native';
import Animated, { FadeInDown } from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';

import { Button, Input, Text } from '@/components/ui';
import { useTheme } from '@/contexts/ThemeContext';
import { useChangePassword } from '@/hooks/useAuth';
import { useHaptic } from '@/hooks/useHaptic';
import { validatePasswordPolicy } from '@/lib/password-policy';
import { COLORS } from '@/lib/theme';

export default function ChangePasswordScreen() {
  const router = useRouter();
  const haptic = useHaptic();
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];

  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [formError, setFormError] = useState<string | null>(null);

  const changePasswordMutation = useChangePassword();

  const handleSubmit = () => {
    if (!oldPassword || !newPassword || !confirmPassword) {
      haptic.error();
      setFormError('Lutfen tum alanlari doldur');
      return;
    }
    const policyError = validatePasswordPolicy(newPassword);
    if (policyError) {
      haptic.error();
      setFormError(policyError);
      return;
    }
    if (newPassword !== confirmPassword) {
      haptic.error();
      setFormError('Yeni sifreler birbiriyle eslesmiyor');
      return;
    }

    setFormError(null);
    haptic.light();
    changePasswordMutation.mutate(
      { old_password: oldPassword, new_password: newPassword },
      {
        onSuccess: () => {
          haptic.success();
          router.replace('/(tabs)');
        },
        onError: (error: any) => {
          haptic.error();
          const errorData = error.response?.data;
          setFormError(errorData?.message ?? 'Sifre degistirilemedi. Tekrar dene.');
        },
      }
    );
  };

  return (
    <SafeAreaView className="flex-1 bg-background" edges={['top', 'bottom']}>
      <KeyboardAvoidingView
        className="flex-1"
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      >
        <ScrollView
          contentContainerClassName="flex-grow justify-center px-6 py-10"
          keyboardShouldPersistTaps="handled"
        >
          <Animated.View entering={FadeInDown.duration(400)} className="mb-10">
            <View className="mb-6 h-14 w-14 items-center justify-center rounded-2xl bg-warning">
              <Ionicons name="key" size={26} color="#fff" />
            </View>
            <Text className="text-3xl font-extrabold text-foreground">Sifreni yenile</Text>
            <Text className="mt-2 text-base text-muted-foreground">
              Guvenligin icin ilk giriste sifreni degistirmen gerekiyor.
            </Text>
          </Animated.View>

          <Animated.View entering={FadeInDown.delay(120).duration(400)} className="gap-4">
            {formError && (
              <View className="flex-row items-center gap-2 rounded-2xl border border-destructive/30 bg-destructive/10 p-3">
                <Ionicons name="alert-circle" size={18} color={colors.destructive} />
                <Text className="flex-1 text-sm font-medium text-destructive">{formError}</Text>
              </View>
            )}

            <View>
              <Text className="mb-2 ml-1 text-sm font-semibold text-foreground">Mevcut sifre</Text>
              <Input
                placeholder="••••••••"
                value={oldPassword}
                onChangeText={setOldPassword}
                secureTextEntry
                autoComplete="password"
                returnKeyType="next"
                editable={!changePasswordMutation.isPending}
                accessibilityLabel="Mevcut sifre"
              />
            </View>

            <View>
              <Text className="mb-2 ml-1 text-sm font-semibold text-foreground">Yeni sifre</Text>
              <Input
                placeholder="En az 8 karakter, buyuk/kucuk harf ve rakam"
                value={newPassword}
                onChangeText={setNewPassword}
                secureTextEntry
                autoComplete="password-new"
                returnKeyType="next"
                editable={!changePasswordMutation.isPending}
                accessibilityLabel="Yeni sifre"
              />
            </View>

            <View>
              <Text className="mb-2 ml-1 text-sm font-semibold text-foreground">
                Yeni sifre (tekrar)
              </Text>
              <Input
                placeholder="••••••••"
                value={confirmPassword}
                onChangeText={setConfirmPassword}
                secureTextEntry
                autoComplete="password-new"
                returnKeyType="done"
                onSubmitEditing={handleSubmit}
                editable={!changePasswordMutation.isPending}
                accessibilityLabel="Yeni sifre tekrar"
              />
            </View>

            <Button
              size="lg"
              className="mt-2"
              onPress={handleSubmit}
              loading={changePasswordMutation.isPending}
              accessibilityLabel="Sifreyi degistir"
            >
              <Text>
                {changePasswordMutation.isPending ? 'Kaydediliyor...' : 'Sifreyi Degistir'}
              </Text>
            </Button>
          </Animated.View>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
