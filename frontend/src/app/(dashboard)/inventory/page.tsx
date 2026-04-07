"use client";

import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { Warehouse, RefreshCw, AlertTriangle } from "lucide-react";
import { Header } from "@/components/layout/header";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";

interface InventoryItem {
  product_id: string;
  product_name: string;
  stock: number;
  reserved: number;
  available: number;
  updated_at: string;
}

export default function InventoryPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["inventory"],
    queryFn: () => api.get<{ success: boolean; data: InventoryItem[] }>("/inventory"),
    select: (res) => res.data.data ?? [],
  });

  const items = data ?? [];
  const lowStock = items.filter((i) => i.available < 10);

  return (
    <div className="flex flex-col">
      <Header title="Склад" description={`${items.length} позиций · ${lowStock.length} на исходе`} />
      <div className="p-6 space-y-4">
        {lowStock.length > 0 && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            className="flex items-center gap-3 p-4 bg-yellow-500/10 border border-yellow-500/20 rounded-lg"
          >
            <AlertTriangle className="w-5 h-5 text-yellow-500 flex-shrink-0" />
            <p className="text-sm text-yellow-600 dark:text-yellow-400">
              {lowStock.length} позиций заканчиваются на складе
            </p>
          </motion.div>
        )}

        <Card>
          <CardHeader className="py-4 px-6 border-b border-border">
            <div className="grid grid-cols-12 gap-4 text-xs font-medium text-muted-foreground uppercase tracking-wider">
              <div className="col-span-4">Товар</div>
              <div className="col-span-2 text-right">Всего</div>
              <div className="col-span-2 text-right">Зарезервировано</div>
              <div className="col-span-2 text-right">Доступно</div>
              <div className="col-span-2 text-right">Статус</div>
            </div>
          </CardHeader>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="flex items-center justify-center h-40">
                <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
              </div>
            ) : items.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-40 text-muted-foreground">
                <Warehouse className="w-8 h-8 mb-2 opacity-30" />
                <p className="text-sm">Склад пуст</p>
              </div>
            ) : (
              items.map((item, i) => (
                <motion.div
                  key={item.product_id}
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: i * 0.03 }}
                  className="grid grid-cols-12 gap-4 items-center px-6 py-4 border-b border-border last:border-0 hover:bg-muted/30 transition-colors"
                >
                  <div className="col-span-4">
                    <p className="text-sm font-medium">{item.product_name || item.product_id}</p>
                    <p className="text-xs font-mono text-muted-foreground">{item.product_id.slice(0, 12)}…</p>
                  </div>
                  <div className="col-span-2 text-right">
                    <p className="text-sm font-medium">{item.stock}</p>
                  </div>
                  <div className="col-span-2 text-right">
                    <p className="text-sm text-muted-foreground">{item.reserved}</p>
                  </div>
                  <div className="col-span-2 text-right">
                    <p className={cn("text-sm font-semibold", item.available < 5 ? "text-red-500" : item.available < 10 ? "text-yellow-500" : "text-green-500")}>
                      {item.available}
                    </p>
                  </div>
                  <div className="col-span-2 text-right">
                    <span className={cn(
                      "inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-semibold",
                      item.available === 0
                        ? "bg-red-500/10 text-red-500 border-red-500/20"
                        : item.available < 10
                        ? "bg-yellow-500/10 text-yellow-500 border-yellow-500/20"
                        : "bg-green-500/10 text-green-500 border-green-500/20"
                    )}>
                      {item.available === 0 ? "Нет" : item.available < 10 ? "Мало" : "В наличии"}
                    </span>
                  </div>
                </motion.div>
              ))
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
