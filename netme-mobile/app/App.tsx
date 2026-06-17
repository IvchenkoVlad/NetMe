import React from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/stack';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { QueryClientProvider, QueryClient } from '@tanstack/react-query';

// Import screens
import HomeScreen from './screens/HomeScreen';
// TODO: Import other screens as they're implemented
// import LoginScreen from './screens/auth/LoginScreen';
// import RegisterScreen from './screens/auth/RegisterScreen';
// import AccountsScreen from './screens/main/AccountsScreen';
// import TransactionsScreen from './screens/main/TransactionsScreen';
// import InsightsScreen from './screens/main/InsightsScreen';
// import SettingsScreen from './screens/main/SettingsScreen';

const Stack = createNativeStackNavigator();
const Tab = createBottomTabNavigator();

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 10 * 60 * 1000, // 10 minutes
    },
  },
});

// TODO: Implement auth navigation
// const AuthStack = () => (
//   <Stack.Navigator>
//     <Stack.Screen name="Login" component={LoginScreen} options={{ headerShown: false }} />
//     <Stack.Screen name="Register" component={RegisterScreen} />
//   </Stack.Navigator>
// );

// TODO: Implement main app navigation
// const MainTabs = () => (
//   <Tab.Navigator>
//     <Tab.Screen name="Dashboard" component={DashboardScreen} />
//     <Tab.Screen name="Accounts" component={AccountsScreen} />
//     <Tab.Screen name="Transactions" component={TransactionsScreen} />
//     <Tab.Screen name="Insights" component={InsightsScreen} />
//     <Tab.Screen name="Settings" component={SettingsScreen} />
//   </Tab.Navigator>
// );

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <SafeAreaProvider>
        <NavigationContainer>
          <Stack.Navigator
            screenOptions={{
              headerTitleAlign: 'center',
            }}
          >
            {/* Home screen for testing backend connection */}
            <Stack.Screen
              name="Home"
              component={HomeScreen}
              options={{ title: 'NetMe' }}
            />
            {/* TODO: Add auth screens (LoginScreen, RegisterScreen) */}
            {/* TODO: Add main app screens (AccountsScreen, TransactionsScreen, etc) */}
          </Stack.Navigator>
        </NavigationContainer>
      </SafeAreaProvider>
    </QueryClientProvider>
  );
}
