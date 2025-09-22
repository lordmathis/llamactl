import React from "react";
import { Badge } from "@/components/ui/badge";
import { BackendType, type BackendTypeValue } from "@/types/instance";
import { Cpu, Zap, Server } from "lucide-react";

interface BackendBadgeProps {
  backend?: BackendTypeValue;
}

const BackendBadge: React.FC<BackendBadgeProps> = ({ backend }) => {
  if (!backend) {
    return null;
  }

  const getIcon = () => {
    switch (backend) {
      case BackendType.LLAMA_CPP:
        return <Cpu className="h-3 w-3" />;
      case BackendType.MLX_LM:
        return <Zap className="h-3 w-3" />;
      case BackendType.VLLM:
        return <Server className="h-3 w-3" />;
      default:
        return <Server className="h-3 w-3" />;
    }
  };

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

  const getVariant = () => {
    switch (backend) {
      case BackendType.LLAMA_CPP:
        return "secondary";
      case BackendType.MLX_LM:
        return "outline";
      case BackendType.VLLM:
        return "default";
      default:
        return "secondary";
    }
  };

  return (
    <Badge
      variant={getVariant()}
      className="flex items-center gap-1.5"
    >
      {getIcon()}
      <span className="text-xs">{getText()}</span>
    </Badge>
  );
};

export default BackendBadge;