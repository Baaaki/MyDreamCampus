import React from 'react';
import { View } from 'react-native';
import FontAwesome from '@expo/vector-icons/FontAwesome';
import { Tabs } from 'expo-router';
import { IconButton, useTheme } from 'react-native-paper';

import { useColorScheme } from '@/components/useColorScheme';
import { useClientOnlyValue } from '@/components/useClientOnlyValue';
import { useTheme as useAppTheme } from '@/contexts/ThemeContext';
import { useAuthContext } from '@/contexts/AuthContext';

function TabBarIcon(props: {
  name: React.ComponentProps<typeof FontAwesome>['name'];
  color: string;
}) {
  return <FontAwesome size={24} style={{ marginBottom: -3 }} {...props} />;
}

export default function TabLayout() {
  const colorScheme = useColorScheme();
  const { toggleTheme, isDark } = useAppTheme();
  const { logout } = useAuthContext();
  const theme = useTheme();
  const { colors } = theme;

  return (
    <Tabs
      screenOptions={{
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.onSurfaceVariant,
        tabBarStyle: {
          backgroundColor: colors.surface,
          borderTopColor: colors.outlineVariant,
          elevation: 4,
        },
        headerStyle: {
          backgroundColor: colors.surface,
          elevation: 2,
        },
        headerTintColor: colors.onSurface,
        headerShown: useClientOnlyValue(false, true),
        headerRight: () => (
          <View style={{ flexDirection: 'row' }}>
            <IconButton
              icon={isDark ? 'white-balance-sunny' : 'moon-waning-crescent'}
              size={22}
              onPress={toggleTheme}
              iconColor={colors.onSurface}
            />
            <IconButton
              icon="logout"
              size={22}
              onPress={logout}
              iconColor={colors.onSurface}
            />
          </View>
        ),
      }}>
      <Tabs.Screen
        name="index"
        options={{
          title: 'Ana Sayfa',
          tabBarIcon: ({ color }) => <TabBarIcon name="home" color={color} />,
        }}
      />
      <Tabs.Screen
        name="two"
        options={{
          title: 'Derslerim',
          tabBarIcon: ({ color }) => <TabBarIcon name="book" color={color} />,
        }}
      />
    </Tabs>
  );
}
