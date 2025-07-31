import { Button } from "@/components/ui/button";
import { HelpCircle, LogOut } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";

interface HeaderProps {
  onCreateInstance: () => void;
  onShowSystemInfo: () => void;
}

function Header({ onCreateInstance, onShowSystemInfo }: HeaderProps) {
  const { logout } = useAuth();

  const handleLogout = () => {
    if (confirm("Are you sure you want to logout?")) {
      logout();
    }
  };

  return (
    <header className="bg-white border-b border-gray-200">
      <div className="container mx-auto max-w-4xl px-4 py-4">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-gray-900">
            Llamactl Dashboard
          </h1>

          <div className="flex items-center gap-2">
            <Button onClick={onCreateInstance} data-testid="create-instance-button">
              Create Instance
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