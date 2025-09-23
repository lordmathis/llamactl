import React from "react";
import { Badge } from "@/components/ui/badge";
import { BackendType, type BackendTypeValue } from "@/types/instance";
import { Server } from "lucide-react";

interface BackendBadgeProps {
  backend?: BackendTypeValue;
}

const BackendBadge: React.FC<BackendBadgeProps> = ({ backend }) => {
  if (!backend) {
    return null;
  }

  const getText = () => {
    switch (backend) {
      case BackendType.LLAMA_CPP:
        return "llama.cpp";
      case BackendType.MLX_LM:
        return "MLX";
      case BackendType.VLLM:
        return "vLLM";
      default:
        return backend;
    }
  };

  const getColorClasses = () => {
    switch (backend) {
      case BackendType.LLAMA_CPP:
        return "bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900 dark:text-blue-200 dark:border-blue-800";
      case BackendType.MLX_LM:
        return "bg-green-100 text-green-800 border-green-200 dark:bg-green-900 dark:text-green-200 dark:border-green-800";
      case BackendType.VLLM:
        return "bg-purple-100 text-purple-800 border-purple-200 dark:bg-purple-900 dark:text-purple-200 dark:border-purple-800";
      default:
        return "bg-gray-100 text-gray-800 border-gray-200 dark:bg-gray-900 dark:text-gray-200 dark:border-gray-800";
    }
  };

  return (
    <Badge
      variant="outline"
      className={`flex items-center gap-1.5 ${getColorClasses()}`}
    >
      <Server className="h-3 w-3" />
      <span className="text-xs">{getText()}</span>
    </Badge>
  );
};

export default BackendBadge;