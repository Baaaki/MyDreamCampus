import Ionicons from '@expo/vector-icons/Ionicons';
import { useRouter } from 'expo-router';
import React, { useState } from 'react';
import { KeyboardAvoidingView, Platform, Pressable, ScrollView, View } from 'react-native';
import Animated, { FadeInDown, FadeInUp } from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';

import { Button, Input, Text } from '@/components/ui';
import { useAuthContext } from '@/contexts/AuthContext';
import { useTheme } from '@/contexts/ThemeContext';
import { useLogin } from '@/hooks/useAuth';
import { useHaptic } from '@/hooks/useHaptic';
import { COLORS } from '@/lib/theme';

export default function LoginScreen() {
  const router = useRouter();
  const haptic = useHaptic();
  const { setUser } = useAuthContext();
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const loginMutation = useLogin();

  const handleLogin = () => {
    if (!email || !password) {
      haptic.error();
      setFormError('Lutfen e-posta ve sifreni gir');
      return;
    }

    setFormError(null);
    haptic.light();
    loginMutation.mutate(
      { email, password },
      {
        onSuccess: (data) => {
          haptic.success();
          setUser(data.user);

          if (data.force_password_change) {
            router.replace('/change-password');
            return;
          }

          router.replace('/(tabs)');
        },
        onError: (error: any) => {
          haptic.error();
          const errorData = error.response?.data;
          let message = 'Giris yapilamadi. Tekrar dene.';

          if (errorData?.error === 'ACCOUNT_LOCKED') {
            message = errorData.message || 'Hesabin gecici olarak kilitlendi';
          } else if (errorData?.error === 'ACCOUNT_DEACTIVATED') {
            message = 'Hesabin devre disi birakilmis';
          } else if (errorData?.error === 'INVALID_CREDENTIALS') {
            message = 'E-posta veya sifre hatali';
          } else if (errorData?.message) {
            message = errorData.message;
          }

          setFormError(message);
        },
      }
    );
  };

  return (
    <SafeAreaView className="flex-1 bg-background">
      <KeyboardAvoidingView
        className="flex-1"
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      >
        <ScrollView
          contentContainerClassName="flex-grow justify-center px-6 py-10"
          keyboardShouldPersistTaps="handled"
        >
          {/* Marka blogu */}
          <Animated.View entering={FadeInUp.duration(500)} className="mb-12">
            <View className="mb-6 h-16 w-16 items-center justify-center rounded-2xl bg-primary">
              <Ionicons name="qr-code" size={30} color={colors.primaryForeground} />
            </View>
            <Text className="mb-3 font-mono text-xs uppercase tracking-widest text-primary">
              MyDreamCampus · Ogrenci
            </Text>
            <Text className="text-4xl font-extrabold leading-tight text-foreground">
              Kampuse{'\n'}hos geldin.
            </Text>
            <Text className="mt-3 text-base text-muted-foreground">
              Yoklaman QR ile, saniyeler icinde.
            </Text>
          </Animated.View>

          {/* Form */}
          <Animated.View entering={FadeInDown.delay(150).duration(500)} className="gap-4">
            {formError && (
              <Animated.View
                entering={FadeInDown.duration(250)}
                className="flex-row items-center gap-2 rounded-2xl border border-destructive/30 bg-destructive/10 p-3"
              >
                <Ionicons name="alert-circle" size={18} color={colors.destructive} />
                <Text className="flex-1 text-sm font-medium text-destructive">{formError}</Text>
              </Animated.View>
            )}

            <View>
              <Text className="mb-2 ml-1 text-sm font-semibold text-foreground">E-posta</Text>
              <View className="relative justify-center">
                <View className="absolute left-4 z-10">
                  <Ionicons name="mail-outline" size={20} color={colors.mutedForeground} />
                </View>
                <Input
                  className="pl-12"
                  placeholder="ogrenci@kampus.edu.tr"
                  value={email}
                  onChangeText={setEmail}
                  autoCapitalize="none"
                  keyboardType="email-address"
                  autoComplete="email"
                  returnKeyType="next"
                  editable={!loginMutation.isPending}
                  accessibilityLabel="E-posta adresi"
                />
              </View>
            </View>

            <View>
              <Text className="mb-2 ml-1 text-sm font-semibold text-foreground">Sifre</Text>
              <View className="relative justify-center">
                <View className="absolute left-4 z-10">
                  <Ionicons name="lock-closed-outline" size={20} color={colors.mutedForeground} />
                </View>
                <Input
                  className="pl-12 pr-12"
                  placeholder="••••••••"
                  value={password}
                  onChangeText={setPassword}
                  secureTextEntry={!showPassword}
                  autoComplete="password"
                  returnKeyType="done"
                  onSubmitEditing={handleLogin}
                  editable={!loginMutation.isPending}
                  accessibilityLabel="Sifre"
                />
                <Pressable
                  className="absolute right-4 z-10"
                  onPress={() => {
                    haptic.selection();
                    setShowPassword((v) => !v);
                  }}
                  accessibilityRole="button"
                  accessibilityLabel={showPassword ? 'Sifreyi gizle' : 'Sifreyi goster'}
                >
                  <Ionicons
                    name={showPassword ? 'eye-off-outline' : 'eye-outline'}
                    size={20}
                    color={colors.mutedForeground}
                  />
                </Pressable>
              </View>
            </View>

            <Button
              size="lg"
              className="mt-2"
              onPress={handleLogin}
              loading={loginMutation.isPending}
              accessibilityLabel="Giris yap"
            >
              <Text>{loginMutation.isPending ? 'Giris yapiliyor...' : 'Giris Yap'}</Text>
            </Button>
          </Animated.View>

          <Animated.View entering={FadeInDown.delay(300).duration(500)}>
            <Text className="mt-10 text-center font-mono text-xs text-muted-foreground">
              QR YOKLAMA · DERS PROGRAMI · PROFIL
            </Text>
          </Animated.View>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
