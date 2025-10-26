// ui/src/components/HealthBadge.tsx
import React from "react";
import { Badge } from "@/components/ui/badge";
import type { HealthStatus } from "@/types/instance";
import { CheckCircle, Loader2, XCircle, Clock, AlertCircle } from "lucide-react";

interface HealthBadgeProps {
  health?: HealthStatus;
}

const HealthBadge: React.FC<HealthBadgeProps> = ({ health }) => {
  if (!health) {
    return null;
  }

  const getIcon = () => {
    switch (health.state) {
      case "ready":
        return <CheckCircle className="h-3 w-3" />;
      case "loading":
        return <Loader2 className="h-3 w-3 animate-spin" />;
      case "starting":
        return <Loader2 className="h-3 w-3 animate-spin" />;
      case "restarting":
        return <Loader2 className="h-3 w-3 animate-spin" />;
      case "stopped":
        return <Clock className="h-3 w-3" />;
      case "error":
        return <AlertCircle className="h-3 w-3" />;
      case "failed":
        return <XCircle className="h-3 w-3" />;
    }
  };

  const getVariant = () => {
    switch (health.state) {
      case "ready":
        return "default";
      case "loading":
        return "outline";
      case "starting":
        return "outline";
      case "restarting":
        return "outline";
      case "stopped":
        return "secondary";
      case "error":
        return "destructive";
      case "failed":
        return "destructive";
    }
  };

  const getText = () => {
    switch (health.state) {
      case "ready":
        return "Ready";
      case "loading":
        return "Loading";
      case "starting":
        return "Starting";
      case "restarting":
        return "Restarting";
      case "stopped":
        return "Stopped";
      case "error":
        return "Error";
      case "failed":
        return "Failed";
    }
  };

  return (
    <Badge
      variant={getVariant()}
      className={`flex items-center gap-1.5 ${
        health.state === "ready"
          ? "bg-green-100 text-green-800 border-green-200 dark:bg-green-900 dark:text-green-200 dark:border-green-800"
          : ""
      }`}
      title={health.error || `Source: ${health.source}`}
    >
      {getIcon()}
      <span className="text-xs">{getText()}</span>
    </Badge>
  );
};

export default HealthBadge;
