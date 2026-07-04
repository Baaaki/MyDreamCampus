import { Link, Stack } from 'expo-router';
import { View } from 'react-native';

import { Text } from '@/components/ui';

export default function NotFoundScreen() {
  return (
    <>
      <Stack.Screen options={{ title: 'Sayfa bulunamadi' }} />
      <View className="flex-1 items-center justify-center bg-background px-8">
        <Text className="mb-2 font-mono text-xs uppercase tracking-widest text-primary">404</Text>
        <Text className="mb-4 text-center text-xl font-bold text-foreground">
          Aradigin sayfa burada degil
        </Text>
        <Link href="/" className="py-3">
          <Text className="font-semibold text-primary">Ana sayfaya don</Text>
        </Link>
      </View>
    </>
  );
}
