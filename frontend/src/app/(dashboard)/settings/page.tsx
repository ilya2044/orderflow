"use client";

import { motion } from "framer-motion";
import { useTheme } from "next-themes";
import { Moon, Sun, Monitor, User, Bell, Shield } from "lucide-react";
import { Header } from "@/components/layout/header";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuthStore } from "@/store/useStore";
import { cn } from "@/lib/utils";

export default function SettingsPage() {
  const { theme, setTheme } = useTheme();
  const { user } = useAuthStore();

  const themes = [
    { id: "light", icon: Sun, label: "Светлая" },
    { id: "dark", icon: Moon, label: "Тёмная" },
    { id: "system", icon: Monitor, label: "Системная" },
  ];

  return (
    <div className="flex flex-col">
      <Header title="Настройки" description="Управление профилем и предпочтениями" />
      <div className="p-6 space-y-6 max-w-2xl">
        <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.1 }}>
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2">
                <User className="w-4 h-4 text-muted-foreground" />
                <CardTitle className="text-base">Профиль</CardTitle>
              </div>
              <CardDescription>Информация о вашем аккаунте</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label className="text-xs mb-1 block">Имя пользователя</Label>
                  <Input defaultValue={user?.username} readOnly className="bg-muted/50" />
                </div>
                <div>
                  <Label className="text-xs mb-1 block">Email</Label>
                  <Input defaultValue={user?.email} readOnly className="bg-muted/50" />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label className="text-xs mb-1 block">Имя</Label>
                  <Input defaultValue={user?.first_name} placeholder="Введите имя" />
                </div>
                <div>
                  <Label className="text-xs mb-1 block">Фамилия</Label>
                  <Input defaultValue={user?.last_name} placeholder="Введите фамилию" />
                </div>
              </div>
              <div>
                <Label className="text-xs mb-1 block">Телефон</Label>
                <Input defaultValue={user?.phone} placeholder="+7 (999) 999-99-99" />
              </div>
              <Button>Сохранить изменения</Button>
            </CardContent>
          </Card>
        </motion.div>

        <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.2 }}>
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2">
                <Monitor className="w-4 h-4 text-muted-foreground" />
                <CardTitle className="text-base">Оформление</CardTitle>
              </div>
              <CardDescription>Выберите тему интерфейса</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-3 gap-3">
                {themes.map(({ id, icon: Icon, label }) => (
                  <button
                    key={id}
                    onClick={() => setTheme(id)}
                    className={cn(
                      "flex flex-col items-center gap-2 p-4 rounded-lg border-2 transition-all",
                      theme === id
                        ? "border-primary bg-primary/5"
                        : "border-border hover:border-muted-foreground/50 hover:bg-muted/50"
                    )}
                  >
                    <Icon className={cn("w-5 h-5", theme === id ? "text-primary" : "text-muted-foreground")} />
                    <span className={cn("text-sm font-medium", theme === id ? "text-primary" : "text-muted-foreground")}>
                      {label}
                    </span>
                  </button>
                ))}
              </div>
            </CardContent>
          </Card>
        </motion.div>

        <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.3 }}>
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2">
                <Shield className="w-4 h-4 text-muted-foreground" />
                <CardTitle className="text-base">Безопасность</CardTitle>
              </div>
              <CardDescription>Управление паролем и сессиями</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label className="text-xs mb-1 block">Текущий пароль</Label>
                <Input type="password" placeholder="••••••••" />
              </div>
              <div>
                <Label className="text-xs mb-1 block">Новый пароль</Label>
                <Input type="password" placeholder="••••••••" />
              </div>
              <Button variant="outline">Изменить пароль</Button>
            </CardContent>
          </Card>
        </motion.div>
      </div>
    </div>
  );
}
