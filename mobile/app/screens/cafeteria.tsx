import { useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { Surface, useTheme, SegmentedButtons } from 'react-native-paper';

import type { Tab } from './cafeteria/helpers';
import { SelectTab } from './cafeteria/SelectTab';
import { MenuTab } from './cafeteria/MenuTab';
import { HistoryTab } from './cafeteria/HistoryTab';

export default function CafeteriaScreen() {
  const { colors } = useTheme();
  const [activeTab, setActiveTab] = useState<Tab>('select');

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <Surface style={styles.tabBarContainer} elevation={1}>
        <SegmentedButtons
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as Tab)}
          buttons={[
            { value: 'select', label: 'Yemek Sec', icon: 'silverware-fork-knife' },
            { value: 'menu', label: 'Menu', icon: 'book-open-variant' },
            { value: 'history', label: 'Gecmis', icon: 'history' },
          ]}
        />
      </Surface>

      {activeTab === 'select' && <SelectTab />}
      {activeTab === 'menu' && <MenuTab />}
      {activeTab === 'history' && <HistoryTab />}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  tabBarContainer: { paddingHorizontal: 16, paddingVertical: 12 },
});
