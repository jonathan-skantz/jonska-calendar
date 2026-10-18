import { AuthScreen } from "./features/auth/AuthScreen";
import { useAuthActions, useCurrentUser } from "./features/auth/useAuth";
import { PreferencesProvider } from "./features/calendar/PreferencesProvider";
import { LoadingScreen } from "./components/LoadingScreen";
import { getErrorText } from "./lib/requestErrors";
import { CalendarApp } from "./CalendarApp";
import "./App.css";

// Signs the user in, then hands over to the calendar.
function App() {
  const { user, isLoading } = useCurrentUser();
  const { authenticate } = useAuthActions();
  if (isLoading) return <LoadingScreen />;
  if (!user) {
    return (
      <AuthScreen
        error={getErrorText(authenticate.error, "Unable to sign in")}
        pending={authenticate.isPending}
        onSubmit={(credentials) => authenticate.mutate(credentials)}
      />
    );
  }
  return (
    <PreferencesProvider userId={user.id}>
      <CalendarApp user={user} />
    </PreferencesProvider>
  );
}

export default App;
