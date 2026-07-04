import Ionicons from '@expo/vector-icons/Ionicons';
import { Tabs } from 'expo-router';
import React from 'react';
import { Pressable, View, type PressableProps } from 'react-native';

import { useTheme } from '@/contexts/ThemeContext';
import { useHaptic } from '@/hooks/useHaptic';
import { COLORS } from '@/lib/theme';

// Tab bar'in imza ogesi: ortada yukseltilmis, mercan renkli dairesel
// yoklama butonu. Standart tab butonunun yerine gecer.
function ScanTabButton({ onPress }: { onPress?: PressableProps['onPress'] }) {
  const haptic = useHaptic();
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];

  return (
    <View className="flex-1 items-center">
      <Pressable
        onPress={(e) => {
          haptic.medium();
          onPress?.(e);
        }}
        accessibilityRole="button"
        accessibilityLabel="Yoklama al"
        className="-mt-6 h-16 w-16 items-center justify-center rounded-full bg-primary active:opacity-90"
        style={{
          shadowColor: colors.primary,
          shadowOpacity: 0.45,
          shadowRadius: 12,
          shadowOffset: { width: 0, height: 6 },
          elevation: 10,
        }}
      >
        <Ionicons name="qr-code" size={26} color={colors.primaryForeground} />
      </Pressable>
    </View>
  );
}

export default function TabLayout() {
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];

  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.mutedForeground,
        tabBarStyle: {
          backgroundColor: colors.card,
          borderTopColor: colors.border,
        },
        tabBarLabelStyle: {
          fontSize: 11,
          fontWeight: '600',
        },
      }}
    >
      <Tabs.Screen
        name="index"
        options={{
          title: 'Ana Sayfa',
          tabBarIcon: ({ color, focused }) => (
            <Ionicons name={focused ? 'home' : 'home-outline'} size={24} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="scan"
        options={{
          title: 'Yoklama',
          tabBarButton: (props) => <ScanTabButton onPress={props.onPress} />,
        }}
      />
      <Tabs.Screen
        name="profile"
        options={{
          title: 'Profil',
          tabBarIcon: ({ color, focused }) => (
            <Ionicons name={focused ? 'person' : 'person-outline'} size={24} color={color} />
          ),
        }}
      />
    </Tabs>
  );
}
