"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { Search, Filter, Plus, Eye, RefreshCw } from "lucide-react";
import { Header } from "@/components/layout/header";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { OrderStatusBadge } from "@/components/orders/order-status-badge";
import { ordersApi, type Order } from "@/lib/api";
import { formatCurrency, formatDate, truncateId, ORDER_STATUS_MAP } from "@/lib/utils";
import { useAuthStore } from "@/store/useStore";

const STATUS_FILTERS = ["all", "pending", "confirmed", "processing", "shipped", "delivered", "cancelled"];

export default function OrdersPage() {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("all");
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["orders", { page, status }],
    queryFn: () => ordersApi.list({ page, limit: 20, ...(status !== "all" && { status }) }),
    select: (res) => res.data,
  });

  const updateStatusMutation = useMutation({
    mutationFn: ({ id, newStatus }: { id: string; newStatus: string }) =>
      ordersApi.updateStatus(id, newStatus),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      setSelectedOrder(null);
    },
  });

  const orders = data?.data ?? [];
  const meta = data?.meta;

  return (
    <div className="flex flex-col">
      <Header title="Заказы" description={`Всего: ${meta?.total ?? 0}`} />
      <div className="p-6 space-y-4">
        <div className="flex flex-col sm:flex-row gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input placeholder="Поиск по ID заказа..." className="pl-9" />
          </div>
          <div className="flex items-center gap-2 overflow-x-auto scrollbar-none">
            <Filter className="w-4 h-4 text-muted-foreground flex-shrink-0" />
            {STATUS_FILTERS.map((s) => (
              <Button
                key={s}
                variant={status === s ? "default" : "outline"}
                size="sm"
                onClick={() => { setStatus(s); setPage(1); }}
                className="flex-shrink-0 text-xs h-8"
              >
                {s === "all" ? "Все" : ORDER_STATUS_MAP[s]?.label ?? s}
              </Button>
            ))}
          </div>
        </div>

        <Card>
          <CardHeader className="py-4 px-6 border-b border-border">
            <div className="grid grid-cols-12 gap-4 text-xs font-medium text-muted-foreground uppercase tracking-wider">
              <div className="col-span-3">Заказ / Дата</div>
              <div className="col-span-3">Сумма</div>
              <div className="col-span-2">Товаров</div>
              <div className="col-span-2">Статус</div>
              <div className="col-span-2 text-right">Действия</div>
            </div>
          </CardHeader>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="flex items-center justify-center h-40">
                <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
              </div>
            ) : orders.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-40 text-muted-foreground">
                <p className="text-sm">Заказов не найдено</p>
              </div>
            ) : (
              orders.map((order, i) => (
                <motion.div
                  key={order.id}
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: i * 0.03 }}
                  className="grid grid-cols-12 gap-4 items-center px-6 py-4 border-b border-border last:border-0 hover:bg-muted/30 transition-colors"
                >
                  <div className="col-span-3">
                    <p className="text-sm font-mono font-medium">#{truncateId(order.id)}</p>
                    <p className="text-xs text-muted-foreground">{formatDate(order.created_at)}</p>
                  </div>
                  <div className="col-span-3">
                    <p className="text-sm font-semibold">{formatCurrency(order.total_price)}</p>
                    <p className="text-xs text-muted-foreground truncate max-w-[200px]">{order.shipping_address}</p>
                  </div>
                  <div className="col-span-2">
                    <p className="text-sm">{order.items?.length ?? 0} шт.</p>
                  </div>
                  <div className="col-span-2">
                    <OrderStatusBadge status={order.status} />
                  </div>
                  <div className="col-span-2 flex justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => setSelectedOrder(order)}
                    >
                      <Eye className="w-3.5 h-3.5" />
                    </Button>
                    {user?.role === "admin" && order.status === "pending" && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 text-xs text-green-500 hover:text-green-400 hover:bg-green-500/10"
                        onClick={() => updateStatusMutation.mutate({ id: order.id, newStatus: "confirmed" })}
                      >
                        Подтвердить
                      </Button>
                    )}
                  </div>
                </motion.div>
              ))
            )}
          </CardContent>
        </Card>

        {meta && meta.pages > 1 && (
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              Страница {page} из {meta.pages}
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                Назад
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= meta.pages}
                onClick={() => setPage((p) => p + 1)}
              >
                Вперёд
              </Button>
            </div>
          </div>
        )}

        {selectedOrder && (
          <div
            className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
            onClick={() => setSelectedOrder(null)}
          >
            <motion.div
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              className="bg-card border border-border rounded-xl p-6 w-full max-w-lg shadow-xl"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h2 className="text-lg font-semibold">Заказ #{truncateId(selectedOrder.id)}</h2>
                  <p className="text-sm text-muted-foreground">{formatDate(selectedOrder.created_at)}</p>
                </div>
                <OrderStatusBadge status={selectedOrder.status} />
              </div>
              <div className="space-y-3">
                <div className="bg-muted/50 rounded-lg p-3">
                  <p className="text-xs text-muted-foreground mb-1">Адрес доставки</p>
                  <p className="text-sm">{selectedOrder.shipping_address}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground mb-2">Товары</p>
                  {(selectedOrder.items ?? []).map((item) => (
                    <div key={item.id} className="flex justify-between text-sm py-1.5 border-b border-border last:border-0">
                      <span>{item.product_name} × {item.quantity}</span>
                      <span className="font-medium">{formatCurrency(item.price * item.quantity)}</span>
                    </div>
                  ))}
                </div>
                <div className="flex justify-between font-semibold text-base pt-2">
                  <span>Итого</span>
                  <span>{formatCurrency(selectedOrder.total_price)}</span>
                </div>
              </div>
              <Button className="w-full mt-4" variant="outline" onClick={() => setSelectedOrder(null)}>
                Закрыть
              </Button>
            </motion.div>
          </div>
        )}
      </div>
    </div>
  );
}
