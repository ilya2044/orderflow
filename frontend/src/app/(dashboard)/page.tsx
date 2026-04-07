"use client";

import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import {
  ShoppingCart,
  Package,
  Users,
  DollarSign,
  ArrowUpRight,
  Clock,
} from "lucide-react";
import { Header } from "@/components/layout/header";
import { StatsCard } from "@/components/dashboard/stats-card";
import { RevenueChart, OrderStatusChart, CategorySalesChart } from "@/components/dashboard/charts";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { OrderStatusBadge } from "@/components/orders/order-status-badge";
import { ordersApi, productsApi, usersApi } from "@/lib/api";
import { formatCurrency, formatDate, truncateId } from "@/lib/utils";

export default function DashboardPage() {
  const { data: ordersData } = useQuery({
    queryKey: ["orders", { limit: 5 }],
    queryFn: () => ordersApi.list({ limit: 5, page: 1 }),
    select: (res) => res.data,
  });

  const { data: allOrdersData } = useQuery({
    queryKey: ["orders-stats"],
    queryFn: () => ordersApi.list({ limit: 1 }),
    select: (res) => res.data,
  });

  const { data: productsData } = useQuery({
    queryKey: ["products-stats"],
    queryFn: () => productsApi.list({ limit: 1 }),
    select: (res) => res.data,
  });

  const { data: usersData } = useQuery({
    queryKey: ["users-stats"],
    queryFn: () => usersApi.list({ limit: 1 }),
    select: (res) => res.data,
  });

  const recentOrders = ordersData?.data ?? [];
  const totalOrders = allOrdersData?.meta.total ?? 0;
  const totalProducts = productsData?.meta.total ?? 0;
  const totalUsers = usersData?.meta.total ?? 0;

  return (
    <div className="flex flex-col">
      <Header title="Дашборд" description="Обзор системы обработки заказов" />
      <div className="p-6 space-y-6">
        <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
          <StatsCard
            title="Выручка (месяц)"
            value={formatCurrency(341000)}
            change={12.5}
            changeLabel="vs прошлый месяц"
            icon={DollarSign}
            iconColor="text-green-500"
            delay={0}
          />
          <StatsCard
            title="Заказы"
            value={totalOrders.toString()}
            change={8.2}
            changeLabel="vs прошлый месяц"
            icon={ShoppingCart}
            iconColor="text-blue-500"
            delay={0.1}
          />
          <StatsCard
            title="Товары"
            value={totalProducts.toString()}
            change={3.1}
            changeLabel="новых"
            icon={Package}
            iconColor="text-purple-500"
            delay={0.2}
          />
          <StatsCard
            title="Пользователи"
            value={totalUsers.toString()}
            change={15.3}
            changeLabel="vs прошлый месяц"
            icon={Users}
            iconColor="text-orange-500"
            delay={0.3}
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <RevenueChart />
          <OrderStatusChart />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.5, duration: 0.4 }}
            className="lg:col-span-2"
          >
            <Card>
              <CardHeader className="flex flex-row items-center justify-between">
                <div>
                  <CardTitle>Последние заказы</CardTitle>
                  <CardDescription>Недавно оформленные заказы</CardDescription>
                </div>
                <a
                  href="/orders"
                  className="text-xs text-primary flex items-center gap-1 hover:underline"
                >
                  Все заказы <ArrowUpRight className="w-3 h-3" />
                </a>
              </CardHeader>
              <CardContent>
                <div className="space-y-0">
                  {recentOrders.length === 0 ? (
                    <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                      <Clock className="w-8 h-8 mb-2 opacity-50" />
                      <p className="text-sm">Заказов пока нет</p>
                    </div>
                  ) : (
                    recentOrders.map((order, i) => (
                      <motion.div
                        key={order.id}
                        initial={{ opacity: 0, x: -10 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: 0.6 + i * 0.05 }}
                        className="flex items-center justify-between py-3 border-b border-border last:border-0"
                      >
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                            <ShoppingCart className="w-3.5 h-3.5 text-primary" />
                          </div>
                          <div>
                            <p className="text-sm font-medium">#{truncateId(order.id)}</p>
                            <p className="text-xs text-muted-foreground">{formatDate(order.created_at)}</p>
                          </div>
                        </div>
                        <div className="flex items-center gap-4">
                          <OrderStatusBadge status={order.status} />
                          <p className="text-sm font-semibold">{formatCurrency(order.total_price)}</p>
                        </div>
                      </motion.div>
                    ))
                  )}
                </div>
              </CardContent>
            </Card>
          </motion.div>

          <CategorySalesChart />
        </div>
      </div>
    </div>
  );
}
