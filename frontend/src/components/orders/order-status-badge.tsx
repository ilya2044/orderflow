import { cn, ORDER_STATUS_MAP } from "@/lib/utils";

interface OrderStatusBadgeProps {
  status: string;
  className?: string;
}

export function OrderStatusBadge({ status, className }: OrderStatusBadgeProps) {
  const config = ORDER_STATUS_MAP[status] ?? { label: status, color: "bg-gray-500/10 text-gray-500 border-gray-500/20" };

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold",
        config.color,
        className
      )}
    >
      {config.label}
    </span>
  );
}
