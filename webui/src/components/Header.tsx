import { Button } from "@/components/ui/button";
import { HelpCircle, LogOut, Moon, Sun } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { useTheme } from "@/contexts/ThemeContext";

interface HeaderProps {
  onCreateInstance: () => void;
  onShowSystemInfo: () => void;
}

function Header({ onCreateInstance, onShowSystemInfo }: HeaderProps) {
  const { logout } = useAuth();
  const { theme, toggleTheme } = useTheme();

  const handleLogout = () => {
    if (confirm("Are you sure you want to logout?")) {
      logout();
    }
  };

  return (
    <header className="bg-card border-b border-border">
      <div className="container mx-auto max-w-4xl px-4 py-4">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-foreground">
            Llamactl Dashboard
          </h1>

          <div className="flex items-center gap-2">
            <Button onClick={onCreateInstance} data-testid="create-instance-button">
              Create Instance
            </Button>

            <Button
              variant="outline"
              size="icon"
              onClick={toggleTheme}
              data-testid="theme-toggle-button"
              title={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
            >
              {theme === 'light' ? <Moon className="h-4 w-4" /> : <Sun className="h-4 w-4" />}
            </Button>

            <Button
              variant="outline"
              size="icon"
              onClick={onShowSystemInfo}
              data-testid="system-info-button"
              title="System Info"
            >
              <HelpCircle className="h-4 w-4" />
            </Button>

            <Button
              variant="outline"
              size="icon"
              onClick={handleLogout}
              data-testid="logout-button"
              title="Logout"
            >
              <LogOut className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>
    </header>
  );
}

export default Header;