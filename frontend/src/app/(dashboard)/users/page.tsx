"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { Search, RefreshCw, ShieldCheck, User as UserIcon } from "lucide-react";
import { Header } from "@/components/layout/header";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { usersApi } from "@/lib/api";
import { formatDateShort, truncateId } from "@/lib/utils";

export default function UsersPage() {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["users", { page, search }],
    queryFn: () => usersApi.list({ page, limit: 20, ...(search && { search }) }),
    select: (res) => res.data,
  });

  const users = data?.data ?? [];
  const meta = data?.meta;

  return (
    <div className="flex flex-col">
      <Header title="Пользователи" description={`Всего: ${meta?.total ?? 0}`} />
      <div className="p-6 space-y-4">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Поиск по email или имени..."
            className="pl-9"
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
          />
        </div>

        <Card>
          <CardHeader className="py-4 px-6 border-b border-border">
            <div className="grid grid-cols-12 gap-4 text-xs font-medium text-muted-foreground uppercase tracking-wider">
              <div className="col-span-4">Пользователь</div>
              <div className="col-span-3">Email</div>
              <div className="col-span-2">Роль</div>
              <div className="col-span-2">Статус</div>
              <div className="col-span-1">Дата</div>
            </div>
          </CardHeader>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="flex items-center justify-center h-40">
                <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
              </div>
            ) : users.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-40 text-muted-foreground">
                <p className="text-sm">Пользователей не найдено</p>
              </div>
            ) : (
              users.map((user, i) => (
                <motion.div
                  key={user.id}
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: i * 0.03 }}
                  className="grid grid-cols-12 gap-4 items-center px-6 py-4 border-b border-border last:border-0 hover:bg-muted/30 transition-colors"
                >
                  <div className="col-span-4 flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                      {user.role === "admin" ? (
                        <ShieldCheck className="w-4 h-4 text-primary" />
                      ) : (
                        <UserIcon className="w-4 h-4 text-primary" />
                      )}
                    </div>
                    <div>
                      <p className="text-sm font-medium">{user.username}</p>
                      <p className="text-xs text-muted-foreground font-mono">#{truncateId(user.id)}</p>
                    </div>
                  </div>
                  <div className="col-span-3">
                    <p className="text-sm truncate">{user.email}</p>
                  </div>
                  <div className="col-span-2">
                    <Badge
                      variant="outline"
                      className={
                        user.role === "admin"
                          ? "border-purple-500/30 text-purple-400 bg-purple-500/10"
                          : "border-blue-500/30 text-blue-400 bg-blue-500/10"
                      }
                    >
                      {user.role === "admin" ? "Админ" : "Пользователь"}
                    </Badge>
                  </div>
                  <div className="col-span-2">
                    <Badge
                      variant="outline"
                      className={
                        user.is_active
                          ? "border-green-500/30 text-green-400 bg-green-500/10"
                          : "border-red-500/30 text-red-400 bg-red-500/10"
                      }
                    >
                      {user.is_active ? "Активен" : "Неактивен"}
                    </Badge>
                  </div>
                  <div className="col-span-1">
                    <p className="text-xs text-muted-foreground">{formatDateShort(user.created_at)}</p>
                  </div>
                </motion.div>
              ))
            )}
          </CardContent>
        </Card>

        {meta && meta.pages > 1 && (
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">Страница {page} из {meta.pages}</p>
            <div className="flex gap-2">
              <button
                className="text-sm text-primary hover:underline disabled:opacity-50 disabled:no-underline"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                Назад
              </button>
              <button
                className="text-sm text-primary hover:underline disabled:opacity-50 disabled:no-underline"
                disabled={page >= meta.pages}
                onClick={() => setPage((p) => p + 1)}
              >
                Вперёд
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
