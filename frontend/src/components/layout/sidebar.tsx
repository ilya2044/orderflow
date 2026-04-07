"use client";

import { usePathname } from "next/navigation";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import {
  LayoutDashboard,
  ShoppingCart,
  Package,
  Users,
  CreditCard,
  Warehouse,
  Settings,
  LogOut,
  ChevronLeft,
  Boxes,
  BarChart3,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useUIStore, useAuthStore } from "@/store/useStore";
import { authApi } from "@/lib/api";
import { useRouter } from "next/navigation";

const navItems = [
  { href: "/", icon: LayoutDashboard, label: "Дашборд" },
  { href: "/orders", icon: ShoppingCart, label: "Заказы" },
  { href: "/products", icon: Package, label: "Товары" },
  { href: "/users", icon: Users, label: "Пользователи", adminOnly: true },
  { href: "/payments", icon: CreditCard, label: "Платежи" },
  { href: "/inventory", icon: Warehouse, label: "Склад", adminOnly: true },
  { href: "/analytics", icon: BarChart3, label: "Аналитика" },
];

export function Sidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const { sidebarOpen, toggleSidebar } = useUIStore();
  const { user, clearAuth, refreshToken } = useAuthStore() as {
    user: { role: string } | null;
    clearAuth: () => void;
    refreshToken: string | null;
  };

  const handleLogout = async () => {
    try {
      if (refreshToken) await authApi.logout(refreshToken);
    } catch {}
    clearAuth();
    router.push("/login");
  };

  const filteredNav = navItems.filter((item) => !item.adminOnly || user?.role === "admin");

  return (
    <motion.aside
      initial={false}
      animate={{ width: sidebarOpen ? 240 : 72 }}
      transition={{ duration: 0.2, ease: "easeInOut" }}
      className="relative flex flex-col h-screen bg-card border-r border-border overflow-hidden flex-shrink-0"
    >
      <div className="flex items-center h-16 px-4 border-b border-border">
        <AnimatePresence mode="wait">
          {sidebarOpen ? (
            <motion.div
              key="full"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="flex items-center gap-2 overflow-hidden"
            >
              <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center flex-shrink-0">
                <Boxes className="w-4 h-4 text-primary-foreground" />
              </div>
              <span className="font-bold text-base whitespace-nowrap">OrderFlow</span>
            </motion.div>
          ) : (
            <motion.div
              key="icon"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center mx-auto"
            >
              <Boxes className="w-4 h-4 text-primary-foreground" />
            </motion.div>
          )}
        </AnimatePresence>
        <button
          onClick={toggleSidebar}
          className={cn(
            "ml-auto p-1.5 rounded-md hover:bg-accent transition-colors flex-shrink-0",
            !sidebarOpen && "mx-auto"
          )}
        >
          <motion.div animate={{ rotate: sidebarOpen ? 0 : 180 }} transition={{ duration: 0.2 }}>
            <ChevronLeft className="w-4 h-4 text-muted-foreground" />
          </motion.div>
        </button>
      </div>

      <nav className="flex-1 px-3 py-4 space-y-1 overflow-hidden">
        {filteredNav.map((item) => {
          const isActive = pathname === item.href;
          return (
            <Link key={item.href} href={item.href}>
              <motion.div
                whileHover={{ x: 2 }}
                className={cn(
                  "flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors cursor-pointer",
                  isActive
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                )}
              >
                <item.icon className={cn("w-4 h-4 flex-shrink-0", isActive && "text-primary")} />
                <AnimatePresence>
                  {sidebarOpen && (
                    <motion.span
                      initial={{ opacity: 0, width: 0 }}
                      animate={{ opacity: 1, width: "auto" }}
                      exit={{ opacity: 0, width: 0 }}
                      className="whitespace-nowrap overflow-hidden"
                    >
                      {item.label}
                    </motion.span>
                  )}
                </AnimatePresence>
              </motion.div>
            </Link>
          );
        })}
      </nav>

      <div className="px-3 py-4 border-t border-border space-y-1">
        <Link href="/settings">
          <motion.div
            whileHover={{ x: 2 }}
            className="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground cursor-pointer transition-colors"
          >
            <Settings className="w-4 h-4 flex-shrink-0" />
            <AnimatePresence>
              {sidebarOpen && (
                <motion.span
                  initial={{ opacity: 0, width: 0 }}
                  animate={{ opacity: 1, width: "auto" }}
                  exit={{ opacity: 0, width: 0 }}
                  className="whitespace-nowrap overflow-hidden"
                >
                  Настройки
                </motion.span>
              )}
            </AnimatePresence>
          </motion.div>
        </Link>
        <motion.button
          whileHover={{ x: 2 }}
          onClick={handleLogout}
          className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium text-muted-foreground hover:bg-destructive/10 hover:text-destructive cursor-pointer transition-colors"
        >
          <LogOut className="w-4 h-4 flex-shrink-0" />
          <AnimatePresence>
            {sidebarOpen && (
              <motion.span
                initial={{ opacity: 0, width: 0 }}
                animate={{ opacity: 1, width: "auto" }}
                exit={{ opacity: 0, width: 0 }}
                className="whitespace-nowrap overflow-hidden"
              >
                Выйти
              </motion.span>
            )}
          </AnimatePresence>
        </motion.button>
      </div>
    </motion.aside>
  );
}
