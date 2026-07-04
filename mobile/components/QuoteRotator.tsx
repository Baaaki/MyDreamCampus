import Ionicons from '@expo/vector-icons/Ionicons';
import React, { useEffect, useState } from 'react';
import { View } from 'react-native';
import Animated, { FadeIn, FadeOut } from 'react-native-reanimated';

import { Text } from '@/components/ui';
import { QUOTE_ROTATE_MS, QUOTES } from '@/constants/quotes';
import { useTheme } from '@/contexts/ThemeContext';
import { COLORS } from '@/lib/theme';

// Ilham sozunu 2 dakikada bir degistirir; her gecis fade ile animasyonlu.
// Rastgele bir sozle baslar ki uygulama her acilista ayni sozle karsilamasin.
export function QuoteRotator() {
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];
  const [index, setIndex] = useState(() => Math.floor(Math.random() * QUOTES.length));

  useEffect(() => {
    const id = setInterval(() => {
      setIndex((prev) => (prev + 1) % QUOTES.length);
    }, QUOTE_ROTATE_MS);
    return () => clearInterval(id);
  }, []);

  const quote = QUOTES[index];

  return (
    <View className="overflow-hidden rounded-3xl bg-primary p-5">
      <Ionicons
        name="sparkles"
        size={18}
        color={colors.primaryForeground}
        style={{ opacity: 0.9 }}
      />
      {/* key={index} her degisimde bileseni yeniden mount eder -> fade tetiklenir */}
      <Animated.View key={index} entering={FadeIn.duration(500)} exiting={FadeOut.duration(200)}>
        <Text className="mt-3 text-lg font-semibold leading-relaxed text-primary-foreground">
          {quote.text}
        </Text>
        <Text className="mt-2 font-mono text-xs uppercase tracking-widest text-primary-foreground/70">
          {quote.author}
        </Text>
      </Animated.View>
    </View>
  );
}
