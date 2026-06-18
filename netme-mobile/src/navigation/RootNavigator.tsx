import React from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { ActivityIndicator, View } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { useAuth } from '../context/AuthContext';
import { LoginScreen } from '../screens/LoginScreen';
import { RegisterScreen } from '../screens/RegisterScreen';
import { ProfileScreen } from '../screens/ProfileScreen';
import SettingsScreen from '../screens/SettingsScreen';

const Stack = createNativeStackNavigator();
const Tab = createBottomTabNavigator();

const AuthStack = () => (
  <Stack.Navigator screenOptions={{ headerShown: false, animation: 'fade' }}>
    <Stack.Screen name="Login" component={LoginScreen} />
    <Stack.Screen name="Register" component={RegisterScreen} />
  </Stack.Navigator>
);

const AppStack = () => (
  <Tab.Navigator screenOptions={{ headerTitleAlign: 'center' }}>
    <Tab.Screen name="Profile" component={ProfileScreen} options={{ title: 'Profile' }} />
    <Tab.Screen name="Settings" component={SettingsScreen} options={{ title: 'Settings' }} />
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
