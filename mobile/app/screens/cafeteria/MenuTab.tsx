import { useState, useRef, useMemo } from 'react';
import { StyleSheet, ScrollView, Pressable, FlatList, View } from 'react-native';
import { Text, Surface, Divider, ActivityIndicator, useTheme, type MD3Theme } from 'react-native-paper';
import FontAwesome from '@expo/vector-icons/FontAwesome';

import { useMonthlyMenu } from '@/hooks/useMeals';
import { spacing, radius, semanticColors } from '@/constants/tokens';
import {
  monthNames,
  fullDayNames,
  shortDayNames,
  menuCategories,
  categoryColors,
  dayKeyMap,
  getMonthDays,
  getWeekOfMonth,
} from './helpers';

export function MenuTab() {
  const theme = useTheme();
  const { colors } = theme;
  const isDark = theme.dark;
  const now = new Date();
  const [selectedDate, setSelectedDate] = useState(now);
  const flatListRef = useRef<FlatList>(null);

  const year = selectedDate.getFullYear();
  const month = selectedDate.getMonth() + 1;
  const { data: menuResponse, isLoading, error } = useMonthlyMenu(year, month);

  const menuData = menuResponse?.data?.menu_data ?? null;
  const monthDays = useMemo(() => getMonthDays(now.getFullYear(), now.getMonth()), []);

  const dow = selectedDate.getDay();
  const isWeekend = dow === 0 || dow === 6;
  const dayKey = dayKeyMap[dow];
  const weekIdx = getWeekOfMonth(selectedDate);

  let normalItems: string[] | null = null;
  let veganItems: string[] | null = null;
  if (!isWeekend && dayKey && menuData) {
    try {
      const nWeek = menuData.normalMenus?.[weekIdx];
      const vWeek = menuData.veganMenus?.[weekIdx];
      if (nWeek?.[dayKey]?.items) normalItems = nWeek[dayKey].items;
      if (vWeek?.[dayKey]?.items) veganItems = vWeek[dayKey].items;
    } catch {}
  }

  const isToday = (d: Date) => d.getDate() === now.getDate() && d.getMonth() === now.getMonth() && d.getFullYear() === now.getFullYear();
  const isSel = (d: Date) => d.getDate() === selectedDate.getDate() && d.getMonth() === selectedDate.getMonth();

  return (
    <ScrollView style={styles.flex1} contentContainerStyle={styles.scrollContent}>
      <View style={styles.menuMonthRow}>
        <FontAwesome name="calendar" size={16} color={colors.primary} />
        <Text variant="titleMedium" style={{ color: colors.onBackground, fontWeight: '600' }}>
          {monthNames[now.getMonth()]} {now.getFullYear()} - Aylik Menu
        </Text>
      </View>

      <FlatList
        ref={flatListRef}
        data={monthDays}
        horizontal
        showsHorizontalScrollIndicator={false}
        keyExtractor={(d) => d.toISOString()}
        initialScrollIndex={Math.max(0, now.getDate() - 3)}
        getItemLayout={(_, index) => ({ length: 64, offset: 64 * index, index })}
        style={styles.dateCarousel}
        renderItem={({ item: d }) => {
          const dWknd = d.getDay() === 0 || d.getDay() === 6;
          const sel = isSel(d);
          const today = isToday(d);
          return (
            <Pressable
              onPress={() => setSelectedDate(d)}
              style={[styles.dateChip, {
                backgroundColor: sel ? colors.primary : dWknd ? colors.surfaceVariant : colors.surface,
                borderColor: sel ? colors.primary : colors.outlineVariant,
              }]}
              accessibilityRole="button"
              accessibilityLabel={`${d.getDate()} ${monthNames[d.getMonth()]}${today ? ', bugun' : ''}${sel ? ', secili' : ''}`}
            >
              <Text variant="labelSmall" style={{ color: sel ? '#fff' : colors.onSurfaceVariant }}>{shortDayNames[d.getDay()]}</Text>
              <Text variant="titleLarge" style={{ color: sel ? '#fff' : dWknd ? colors.onSurfaceVariant : colors.onSurface, fontWeight: '700' }}>{d.getDate()}</Text>
              {today && <View style={[styles.todayDot, { backgroundColor: sel ? '#fff' : colors.primary }]} />}
            </Pressable>
          );
        }}
      />

      {isLoading && <ActivityIndicator animating color={colors.primary} style={{ marginVertical: 20 }} />}

      {error && !isLoading && (
        <Surface style={[styles.infoBanner, { backgroundColor: semanticColors.warningLight }]} elevation={0} accessibilityRole="alert">
          <FontAwesome name="info-circle" size={14} color={semanticColors.warningDark} />
          <Text variant="bodySmall" style={{ color: '#92400e', flex: 1 }}>Bu ay icin henuz menu olusturulmamis veya backend'e baglanilmadi.</Text>
        </Surface>
      )}

      {isWeekend && (
        <Surface style={[styles.infoBanner, { backgroundColor: isDark ? '#1e3a5f' : '#eff6ff' }]} elevation={0} accessibilityRole="text">
          <FontAwesome name="info-circle" size={14} color="#3b82f6" />
          <Text variant="bodySmall" style={{ color: '#1d4ed8', flex: 1 }}>Hafta sonlari yemek servisi bulunmamaktadir.</Text>
        </Surface>
      )}

      <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant, marginBottom: spacing.md }}>
        {selectedDate.getDate()} {monthNames[selectedDate.getMonth()]} {selectedDate.getFullYear()}, {fullDayNames[selectedDate.getDay()]}
      </Text>

      {/* Normal Menu Card */}
      <MenuCard
        title="Standart Menu"
        icon="cutlery"
        color="#3b82f6"
        headerBg={isDark ? '#1e3a5f' : '#eff6ff'}
        items={normalItems}
        colors={colors}
      />

      {/* Vegan Menu Card */}
      <MenuCard
        title="Vegan Menu"
        icon="leaf"
        color="#22c55e"
        headerBg={isDark ? '#14332a' : '#f0fdf4'}
        items={veganItems}
        colors={colors}
      />

      <Surface style={[styles.footnote, { backgroundColor: colors.surfaceVariant }]} elevation={0}>
        <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>
          *Mucbir sebepler haricinde kesinlikle menu degisimi yapilmayacaktir.
        </Text>
      </Surface>
    </ScrollView>
  );
}

// ─── MenuCard Sub-component ────────────────────────────────

function MenuCard({ title, icon, color, headerBg, items, colors }: {
  title: string;
  icon: 'cutlery' | 'leaf';
  color: string;
  headerBg: string;
  items: string[] | null;
  colors: MD3Theme['colors'];
}) {
  const hasItems = items && items.some((i) => i && i.trim());

  return (
    <Surface style={[styles.menuCard, { borderLeftColor: color }]} elevation={1} accessibilityLabel={`${title}`}>
      <View style={[styles.menuCardHeader, { backgroundColor: headerBg }]}>
        <FontAwesome name={icon} size={16} color={color} />
        <Text variant="titleSmall" style={{ color, fontWeight: '700' }}>{title}</Text>
      </View>
      {hasItems ? (
        <View style={styles.menuCardBody}>
          {menuCategories.map((cat, idx) => (
            <View key={cat} style={styles.menuCatRow}>
              <Text variant="labelSmall" style={{ color: categoryColors[cat] || colors.onSurfaceVariant, fontWeight: '700', textTransform: 'uppercase', letterSpacing: 0.5 }}>{cat}</Text>
              <Text variant="bodyMedium" style={{ color: colors.onSurface }}>{items![idx] || '-'}</Text>
              {idx < menuCategories.length - 1 && <Divider style={{ marginTop: 10, backgroundColor: colors.outlineVariant }} />}
            </View>
          ))}
        </View>
      ) : (
        <View style={styles.menuEmpty}>
          <FontAwesome name={icon} size={24} color={colors.onSurfaceVariant} style={{ opacity: 0.4 }} />
          <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant }}>Bu gun icin menu bulunmuyor</Text>
        </View>
      )}
    </Surface>
  );
}

const styles = StyleSheet.create({
  flex1: { flex: 1 },
  scrollContent: { padding: spacing.md, paddingBottom: spacing.xl },
  menuMonthRow: { flexDirection: 'row', alignItems: 'center', gap: spacing.sm, marginBottom: spacing.md },
  dateCarousel: { marginBottom: spacing.md },
  dateChip: { width: 56, alignItems: 'center', paddingVertical: 10, borderRadius: radius.md, borderWidth: 1, marginRight: spacing.sm },
  todayDot: { width: 5, height: 5, borderRadius: 3, marginTop: 3 },
  infoBanner: { flexDirection: 'row', alignItems: 'center', gap: spacing.sm, borderRadius: radius.md, padding: spacing.md, marginBottom: spacing.md },
  menuCard: { borderRadius: radius.lg, borderLeftWidth: 4, marginBottom: spacing.md, overflow: 'hidden' },
  menuCardHeader: { flexDirection: 'row', alignItems: 'center', gap: spacing.sm, paddingHorizontal: spacing.md, paddingVertical: spacing.md },
  menuCardBody: { padding: spacing.md },
  menuCatRow: { paddingVertical: spacing.sm, paddingHorizontal: spacing.xs },
  menuEmpty: { alignItems: 'center', paddingVertical: 28, gap: spacing.sm },
  footnote: { borderRadius: radius.sm, padding: spacing.md, marginTop: spacing.xs },
});
