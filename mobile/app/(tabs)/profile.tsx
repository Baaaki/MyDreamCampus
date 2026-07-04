import Ionicons from '@expo/vector-icons/Ionicons';
import { useRouter } from 'expo-router';
import React from 'react';
import { Alert, Pressable, ScrollView, Switch, View } from 'react-native';
import Animated, { FadeInDown } from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';

import { Badge, Button, Card, Text } from '@/components/ui';
import { useAuthContext } from '@/contexts/AuthContext';
import { useTheme } from '@/contexts/ThemeContext';
import { useActiveSemester } from '@/hooks/useCatalog';
import { useMyEnrollments } from '@/hooks/useEnrollment';
import { useHaptic } from '@/hooks/useHaptic';
import { COLORS } from '@/lib/theme';

function initialsOf(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('');
}

export default function ProfileScreen() {
  const router = useRouter();
  const haptic = useHaptic();
  const { user, logout } = useAuthContext();
  const { isDark, toggleTheme } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];

  const semesterQuery = useActiveSemester();
  const semester = semesterQuery.data?.name ?? '';
  const enrollmentsQuery = useMyEnrollments(semester || undefined, 'approved');
  const program = enrollmentsQuery.data?.programs?.[0];

  const displayName = program?.student_name ?? user?.email?.split('@')[0] ?? 'Ogrenci';

  const handleLogout = () => {
    haptic.light();
    Alert.alert('Cikis yap', 'Hesabindan cikmak istedigine emin misin?', [
      { text: 'Vazgec', style: 'cancel' },
      {
        text: 'Cikis Yap',
        style: 'destructive',
        onPress: () => {
          haptic.medium();
          logout();
        },
      },
    ]);
  };

  return (
    <SafeAreaView className="flex-1 bg-background" edges={['top']}>
      <ScrollView contentContainerClassName="px-5 pb-10 pt-4">
        <Animated.View entering={FadeInDown.duration(400)}>
          <Text className="mb-6 font-mono text-xs uppercase tracking-widest text-primary">
            Profil
          </Text>

          {/* Kimlik karti */}
          <Card className="mb-6 items-center p-8">
            <View className="mb-4 h-24 w-24 items-center justify-center rounded-full bg-primary">
              <Text className="text-3xl font-extrabold text-primary-foreground">
                {initialsOf(displayName) || '?'}
              </Text>
            </View>
            <Text className="text-2xl font-extrabold text-foreground">{displayName}</Text>
            {user?.email && (
              <Text className="mt-1 text-sm text-muted-foreground">{user.email}</Text>
            )}
            <View className="mt-4 flex-row flex-wrap justify-center gap-2">
              {program?.student_number && (
                <Badge variant="secondary">
                  <Text className="font-mono text-xs font-semibold text-secondary-foreground">
                    {program.student_number}
                  </Text>
                </Badge>
              )}
              {(program?.department ?? user?.department) && (
                <Badge variant="outline">
                  <Text className="text-xs font-semibold text-foreground">
                    {program?.department ?? user?.department}
                  </Text>
                </Badge>
              )}
              {program?.class_level ? (
                <Badge variant="outline">
                  <Text className="text-xs font-semibold text-foreground">
                    {program.class_level}. Sinif
                  </Text>
                </Badge>
              ) : null}
            </View>
            {semester ? (
              <Text className="mt-4 font-mono text-xs uppercase tracking-widest text-muted-foreground">
                Aktif donem · {semester}
              </Text>
            ) : null}
          </Card>
        </Animated.View>

        {/* Ayarlar */}
        <Animated.View entering={FadeInDown.delay(120).duration(400)}>
          <Text className="mb-3 font-mono text-xs uppercase tracking-widest text-muted-foreground">
            Ayarlar
          </Text>
          <Card className="mb-6">
            <View className="flex-row items-center justify-between p-4">
              <View className="flex-row items-center gap-3">
                <View className="h-10 w-10 items-center justify-center rounded-xl bg-secondary">
                  <Ionicons
                    name={isDark ? 'moon' : 'sunny'}
                    size={20}
                    color={colors.foreground}
                  />
                </View>
                <Text className="font-semibold text-foreground">Karanlik Mod</Text>
              </View>
              <Switch
                value={isDark}
                onValueChange={() => {
                  haptic.selection();
                  toggleTheme();
                }}
                trackColor={{ true: colors.primary }}
                accessibilityLabel="Karanlik modu ac veya kapat"
              />
            </View>

            <View className="mx-4 h-px bg-border" />

            <Pressable
              className="flex-row items-center justify-between p-4 active:opacity-70"
              onPress={() => {
                haptic.light();
                router.push('/change-password');
              }}
              accessibilityRole="button"
              accessibilityLabel="Sifre degistir"
            >
              <View className="flex-row items-center gap-3">
                <View className="h-10 w-10 items-center justify-center rounded-xl bg-secondary">
                  <Ionicons name="key-outline" size={20} color={colors.foreground} />
                </View>
                <Text className="font-semibold text-foreground">Sifre Degistir</Text>
              </View>
              <Ionicons name="chevron-forward" size={18} color={colors.mutedForeground} />
            </Pressable>
          </Card>
        </Animated.View>

        <Animated.View entering={FadeInDown.delay(200).duration(400)}>
          <Button variant="destructive" onPress={handleLogout} accessibilityLabel="Cikis yap">
            <Ionicons name="log-out-outline" size={18} color="#fff" />
            <Text>Cikis Yap</Text>
          </Button>

          <Text className="mt-8 text-center font-mono text-xs text-muted-foreground">
            MyDreamCampus Mobile · v1.0.0
          </Text>
        </Animated.View>
      </ScrollView>
    </SafeAreaView>
  );
}
