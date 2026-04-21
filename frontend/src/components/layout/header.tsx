"use client";

import { useState } from "react";
import { Bell, Moon, Sun, Search, ShoppingCart } from "lucide-react";
import { useTheme } from "next-themes";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuthStore, useCartStore, useNotificationStore } from "@/store/useStore";
import { notificationsApi, type Notification } from "@/lib/api";
import { CartDrawer } from "@/components/cart/cart-drawer";
import { motion } from "framer-motion";
import { formatDate } from "@/lib/utils";

interface HeaderProps {
  title: string;
  description?: string;
}

export function Header({ title, description }: HeaderProps) {
  const { theme, setTheme } = useTheme();
  const { user } = useAuthStore();
  const { items } = useCartStore();
  const { readIds, markRead } = useNotificationStore();
  const [cartOpen, setCartOpen] = useState(false);
  const [notifOpen, setNotifOpen] = useState(false);
  const isAdmin = user?.role === "admin";
  const cartCount = items.reduce((s, i) => s + i.quantity, 0);

  const { data: notifData } = useQuery({
    queryKey: ["notifications", user?.id, isAdmin],
    queryFn: () => isAdmin ? notificationsApi.listForAdmin() : notificationsApi.listForUser(user!.id),
    enabled: !!user,
    refetchInterval: 30_000,
    select: (res) => (res.data.data ?? []) as Notification[],
  });

  const notifications = notifData ?? [];
  const unreadCount = notifications.filter((n) => !readIds.includes(n.id)).length;

  function handleOpenNotif() {
    setNotifOpen((v) => !v);
  }

  function handleCloseNotif() {
    if (notifications.length > 0) {
      markRead(notifications.map((n) => n.id));
    }
    setNotifOpen(false);
  }

  return (
    <header className="sticky top-0 z-40 flex h-16 items-center gap-4 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 px-6">
      <div className="flex-1">
        <motion.div
          initial={{ opacity: 0, y: -4 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.2 }}
        >
          <h1 className="text-lg font-semibold">{title}</h1>
          {description && (
            <p className="text-xs text-muted-foreground">{description}</p>
          )}
        </motion.div>
      </div>

      <div className="hidden md:flex items-center gap-2 w-72">
        <div className="relative w-full">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Поиск..."
            className="pl-9 h-9 bg-muted/50 border-0 focus-visible:ring-1"
          />
        </div>
      </div>

      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="icon"
          className="h-9 w-9"
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
        >
          <Sun className="h-4 w-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
          <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
        </Button>

        <div className="relative">
          <Button
            variant="ghost"
            size="icon"
            className="h-9 w-9 relative"
            onClick={handleOpenNotif}
          >
            <Bell className="h-4 w-4" />
            {unreadCount > 0 && (
              <span className="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 rounded-full bg-primary text-primary-foreground text-[10px] font-bold flex items-center justify-center px-1">
                {unreadCount > 99 ? "99+" : unreadCount}
              </span>
            )}
          </Button>
          {notifOpen && (
            <>
              <div className="fixed inset-0 z-40" onClick={handleCloseNotif} />
              <div className="absolute right-0 top-11 z-50 w-80 bg-card border border-border rounded-xl shadow-xl overflow-hidden">
                <div className="px-4 py-3 border-b border-border flex items-center justify-between">
                  <p className="text-sm font-semibold">Уведомления</p>
                  <span className="text-xs text-muted-foreground">
                    {unreadCount > 0 ? `${unreadCount} новых` : "Всё прочитано"}
                  </span>
                </div>
                {notifications.length === 0 ? (
                  <div className="flex flex-col items-center justify-center py-8 text-muted-foreground gap-2">
                    <Bell className="w-8 h-8 opacity-20" />
                    <p className="text-sm">Нет новых уведомлений</p>
                  </div>
                ) : (
                  <div className="max-h-80 overflow-y-auto divide-y divide-border">
                    {notifications.map((n) => (
                      <div
                        key={n.id}
                        className={`px-4 py-3 hover:bg-muted/30 transition-colors ${!readIds.includes(n.id) ? "bg-primary/5" : ""}`}
                      >
                        <div className="flex items-start justify-between gap-2">
                          <p className="text-sm font-medium leading-tight">{n.subject}</p>
                          {!readIds.includes(n.id) && (
                            <span className="w-2 h-2 rounded-full bg-primary flex-shrink-0 mt-1" />
                          )}
                        </div>
                        <p className="text-xs text-muted-foreground mt-0.5 leading-relaxed">{n.body}</p>
                        <p className="text-[10px] text-muted-foreground/60 mt-1">{formatDate(n.created_at)}</p>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </>
          )}
        </div>

        {!isAdmin && (
          <Button
            variant="ghost"
            size="icon"
            className="h-9 w-9 relative"
            onClick={() => setCartOpen(true)}
            title="Корзина"
          >
            <ShoppingCart className="h-4 w-4" />
            {cartCount > 0 && (
              <span className="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 rounded-full bg-primary text-primary-foreground text-[10px] font-bold flex items-center justify-center px-1">
                {cartCount}
              </span>
            )}
          </Button>
        )}

        <div className="flex items-center gap-2 pl-2 border-l border-border">
          <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
            <span className="text-xs font-semibold text-primary">
              {user?.username?.slice(0, 2).toUpperCase() ?? "U"}
            </span>
          </div>
          <div className="hidden sm:block">
            <p className="text-sm font-medium leading-none">{user?.username ?? "User"}</p>
            <p className="text-xs text-muted-foreground capitalize">{user?.role ?? "user"}</p>
          </div>
        </div>
      </div>

      <CartDrawer open={cartOpen} onClose={() => setCartOpen(false)} />
    </header>
  );
}
