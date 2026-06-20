import React, { useRef, useState, useCallback } from 'react';
import {
  View,
  Text,
  Image,
  TouchableOpacity,
  ScrollView,
  Dimensions,
  StyleSheet,
  StatusBar,
  Pressable,
  Modal,
} from 'react-native';
import { LinearGradient } from 'expo-linear-gradient';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import { AccountsScreen } from './AccountsScreen';
import { BudgetScreen } from './BudgetScreen';

const { width: W } = Dimensions.get('window');

const TABS = [
  { label: 'Accounts', screen: <AccountsScreen /> },
  { label: 'Budget', screen: <BudgetScreen /> },
];

const Logo = ({ onPress }: { onPress: () => void }) => (
  <TouchableOpacity onPress={onPress} activeOpacity={0.8} style={s.logoBtn}>
    <Image source={require('../../assets/logo.png')} style={s.logoImage} />
  </TouchableOpacity>
);

const LogoMenu = ({
  visible,
  onClose,
  anchorY,
}: {
  visible: boolean;
  onClose: () => void;
  anchorY: number;
}) => {
  const navigation = useNavigation<any>();
  const go = (screen: string) => { onClose(); navigation.navigate(screen); };

  return (
    <Modal transparent visible={visible} animationType="fade" onRequestClose={onClose}>
      <Pressable style={StyleSheet.absoluteFill} onPress={onClose} />
      <View style={[s.menu, { top: anchorY }]}>
        <TouchableOpacity style={s.menuItem} onPress={() => go('Profile')}>
          <Text style={s.menuIcon}>👤</Text>
          <Text style={s.menuLabel}>Account</Text>
        </TouchableOpacity>
        <View style={s.menuDivider} />
        <TouchableOpacity style={s.menuItem} onPress={() => go('Settings')}>
          <Text style={s.menuIcon}>⚙️</Text>
          <Text style={s.menuLabel}>Settings</Text>
        </TouchableOpacity>
      </View>
    </Modal>
  );
};

export const MainScreen = () => {
  const insets = useSafeAreaInsets();
  const pagerRef = useRef<ScrollView>(null);
  const [activeTab, setActiveTab] = useState(0);
  const [headerHeight, setHeaderHeight] = useState(0);
  const [menuVisible, setMenuVisible] = useState(false);

  const goToTab = useCallback((index: number) => {
    pagerRef.current?.scrollTo({ x: index * W, animated: true });
    setActiveTab(index);
  }, []);

  const onMomentumScrollEnd = useCallback((e: any) => {
    const index = Math.round(e.nativeEvent.contentOffset.x / W);
    setActiveTab(index);
  }, []);

  const pagePaddingTop = Math.max(0, headerHeight - insets.top);

  return (
    // Full-screen gradient — the single background for the whole app
    <LinearGradient colors={['#2dd4a7', '#1e3a5f', '#0f172a']} locations={[0, 0.3, 1]} style={s.root}>
      <StatusBar barStyle="light-content" />

      {/* Pager — transparent, gradient shows through */}
      <ScrollView
        ref={pagerRef}
        horizontal
        pagingEnabled
        showsHorizontalScrollIndicator={false}
        onMomentumScrollEnd={onMomentumScrollEnd}
        scrollEventThrottle={16}
        style={StyleSheet.absoluteFill}
      >
        {TABS.map((tab, i) => (
          <View key={i} style={{ width: W, flex: 1, paddingTop: pagePaddingTop }}>
            {tab.screen}
          </View>
        ))}
      </ScrollView>

      {/* Floating header — no background, floats on the gradient */}
      <View
        style={[s.header, { paddingTop: insets.top }]}
        onLayout={e => setHeaderHeight(e.nativeEvent.layout.height)}
      >
        <View style={s.logoRow}>
          <Logo onPress={() => setMenuVisible(true)} />
        </View>

        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={s.tabRow}
        >
          {TABS.map((tab, i) => (
            <TouchableOpacity
              key={i}
              onPress={() => goToTab(i)}
              style={[s.pill, activeTab === i && s.pillActive]}
            >
              <Text style={[s.pillText, activeTab === i && s.pillTextActive]}>
                {tab.label}
              </Text>
            </TouchableOpacity>
          ))}
        </ScrollView>
      </View>

      <LogoMenu
        visible={menuVisible}
        onClose={() => setMenuVisible(false)}
        anchorY={headerHeight + 8}
      />
    </LinearGradient>
  );
};

const s = StyleSheet.create({
  root: { flex: 1 },

  header: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    zIndex: 10,
  },
  logoRow: {
    alignItems: 'center',
    paddingVertical: 10,
  },
  logoBtn: {
    alignItems: 'center',
    justifyContent: 'center',
  },
  logoImage: {
    width: 44,
    height: 44,
    borderRadius: 12,
  },

  tabRow: {
    flexDirection: 'row',
    paddingHorizontal: 16,
    paddingBottom: 12,
    gap: 8,
  },
  pill: {
    paddingHorizontal: 18,
    paddingVertical: 7,
    borderRadius: 20,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.1)',
  },
  pillActive: {
    backgroundColor: 'rgba(45,212,167,0.15)',
    borderColor: 'rgba(45,212,167,0.45)',
  },
  pillText: {
    color: 'rgba(255,255,255,0.4)',
    fontSize: 14,
    fontWeight: '500',
  },
  pillTextActive: {
    color: '#2dd4a7',
    fontWeight: '600',
  },

  menu: {
    position: 'absolute',
    alignSelf: 'center',
    backgroundColor: 'rgba(15,30,60,0.97)',
    borderRadius: 14,
    borderWidth: 1,
    borderColor: 'rgba(45,212,167,0.25)',
    overflow: 'hidden',
    minWidth: 180,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.5,
    shadowRadius: 20,
    elevation: 14,
  },
  menuItem: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 14,
    paddingHorizontal: 18,
    gap: 12,
  },
  menuIcon: { fontSize: 18 },
  menuLabel: { color: '#e2e8f0', fontSize: 15, fontWeight: '500' },
  menuDivider: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: 'rgba(255,255,255,0.08)',
    marginHorizontal: 12,
  },
});
