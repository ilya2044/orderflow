import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatCurrency(amount: number, currency = "RUB"): string {
  return new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency,
    minimumFractionDigits: 0,
  }).format(amount);
}

export function formatDate(dateStr: string): string {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(dateStr));
}

export function formatDateShort(dateStr: string): string {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  }).format(new Date(dateStr));
}

export const ORDER_STATUS_MAP: Record<string, { label: string; color: string }> = {
  pending:    { label: "Ожидает",      color: "bg-yellow-500/10 text-yellow-500 border-yellow-500/20" },
  confirmed:  { label: "Подтверждён",  color: "bg-blue-500/10 text-blue-500 border-blue-500/20" },
  processing: { label: "Обрабатывается", color: "bg-purple-500/10 text-purple-500 border-purple-500/20" },
  shipped:    { label: "Отправлен",    color: "bg-indigo-500/10 text-indigo-500 border-indigo-500/20" },
  delivered:  { label: "Доставлен",   color: "bg-green-500/10 text-green-500 border-green-500/20" },
  cancelled:  { label: "Отменён",     color: "bg-red-500/10 text-red-500 border-red-500/20" },
  refunded:   { label: "Возвращён",   color: "bg-gray-500/10 text-gray-500 border-gray-500/20" },
};

export const PAYMENT_STATUS_MAP: Record<string, { label: string; color: string }> = {
  pending:    { label: "Ожидает",  color: "bg-yellow-500/10 text-yellow-500 border-yellow-500/20" },
  processing: { label: "Обработка", color: "bg-blue-500/10 text-blue-500 border-blue-500/20" },
  completed:  { label: "Оплачен", color: "bg-green-500/10 text-green-500 border-green-500/20" },
  failed:     { label: "Ошибка",  color: "bg-red-500/10 text-red-500 border-red-500/20" },
  refunded:   { label: "Возврат", color: "bg-gray-500/10 text-gray-500 border-gray-500/20" },
};

export function truncateId(id: string, length = 8): string {
  return id.slice(0, length).toUpperCase();
}
