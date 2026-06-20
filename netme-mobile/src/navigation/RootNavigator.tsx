import React from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { ActivityIndicator, View, Text } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { useAuth } from '../context/AuthContext';
import { LoginScreen } from '../screens/LoginScreen';
import { RegisterScreen } from '../screens/RegisterScreen';
import { ProfileScreen } from '../screens/ProfileScreen';
import SettingsScreen from '../screens/SettingsScreen';
import { AccountsScreen } from '../screens/AccountsScreen';
import { BudgetScreen } from '../screens/BudgetScreen';

const Stack = createNativeStackNavigator();
const Tab = createBottomTabNavigator();

const AuthStack = () => (
  <Stack.Navigator screenOptions={{ headerShown: false, animation: 'fade' }}>
    <Stack.Screen name="Login" component={LoginScreen} />
    <Stack.Screen name="Register" component={RegisterScreen} />
  </Stack.Navigator>
);

const AppStack = () => (
  <Tab.Navigator
    screenOptions={{
      headerTitleAlign: 'center',
      tabBarActiveTintColor: '#2dd4a7',
      tabBarInactiveTintColor: '#94a3b8',
      tabBarStyle: { borderTopColor: '#e2e8f0', backgroundColor: '#fff' },
    }}
  >
    <Tab.Screen
      name="Accounts"
      component={AccountsScreen}
      options={{
        headerShown: false,
        tabBarLabel: 'Accounts',
        tabBarIcon: ({ color }: { color: string }) => (
          <Text style={{ fontSize: 20, color }}>🏦</Text>
        ),
      }}
    />
    <Tab.Screen
      name="Budget"
      component={BudgetScreen}
      options={{
        headerShown: false,
        tabBarLabel: 'Budget',
        tabBarIcon: ({ color }: { color: string }) => (
          <Text style={{ fontSize: 20, color }}>📊</Text>
        ),
      }}
    />
    <Tab.Screen
      name="Profile"
      component={ProfileScreen}
      options={{
        title: 'Profile',
        tabBarIcon: ({ color }: { color: string }) => (
          <Text style={{ fontSize: 20, color }}>👤</Text>
        ),
      }}
    />
    <Tab.Screen
      name="Settings"
      component={SettingsScreen}
      options={{
        title: 'Settings',
        tabBarIcon: ({ color }: { color: string }) => (
          <Text style={{ fontSize: 20, color }}>⚙️</Text>
        ),
      }}
    />
  </Tab.Navigator>
);

const SplashScreen = () => (
  <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center', backgroundColor: '#0f172a' }}>
    <ActivityIndicator size="large" color="#2dd4a7" />
  </View>
);

export const RootNavigator: React.FC = () => {
  const { isAuthenticated, isLoading } = useAuth();

  return (
    <SafeAreaProvider>
      <NavigationContainer>
        {isLoading ? <SplashScreen /> : isAuthenticated ? <AppStack /> : <AuthStack />}
      </NavigationContainer>
    </SafeAreaProvider>
  );
};
